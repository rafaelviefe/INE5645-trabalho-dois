# Relatório Técnico — Trabalho 2: Programação Distribuída

**INE 5645 — Programação Paralela e Distribuída — Semestre 2026/1**  
**Universidade Federal de Santa Catarina — UFSC**

**Alunos:** Arthur Schurhaus, Rafael Vieira e Uriel Jaloto

---

## 1. Introdução

Este relatório descreve a arquitetura e implementação de um protótipo de sistema de operação em mercado automatizado (*trading*) como parte do Trabalho 2 da disciplina INE 5645. O sistema simula um fluxo de compra de dois ativos a partir de dados recebidos de sistemas de cotação e análise de risco, utilizando processos distribuídos.

O sistema é composto por seis processos independentes:

- **Sistema de Operação (Orquestrador):** coordena o fluxo da transação distribuída via CLI interativa;
- **Sistema de Cotação:** fornece preços simulados e TTL para os pares de ativos;
- **Sistema de Risco:** analisa a viabilidade da operação com base em taxa de sucesso configurável;
- **Sistema de Compra (Trade):** executa as ordens de compra e venda (compensação);
- **Broker (PubSub):** barramento de eventos que recebe publicações dos serviços e as disponibiliza para consumidores;
- **Monitor:** consumidor de eventos que persiste o histórico em `trace.jsonl` e exibe em tempo real.

A comunicação entre todos os processos é feita via **sockets Berkeley (TCP) puros**, implementados do zero em Go, sem uso de bibliotecas externas de comunicação. Quatro padrões de projeto para programação distribuída foram aplicados: **Replicas com Load Balancing (Round-Robin Pool)**, **SAGA (Orchestrator)** com **Transações Compensatórias**, e **PubSub**.

---

## 2. Arquitetura Geral

O sistema segue uma arquitetura de **microsserviços**, onde cada processo é um binário independente que se comunica via TCP. O Orquestrador centraliza o fluxo da transação, consultando os serviços satélites sequencialmente. O Broker atua como barramento de eventos, recebendo publicações da Cotação e servindo como fonte única de verdade para o Monitor.

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│ Monitor  │◄────│    Broker    │◄────│ Cotação  │
│ (local)  │pull │   :8090      │ pub  │ :8081/84 │
└──────────┘     └──────────────┘     └──────────┘

┌───────────────────────────────────────────────────┐
│              Orquestrador (CLI)                     │
│              cmd/operation/main.go                  │
├──────────┬──────────────┬──────────────────────────┤
│ Cotação  │    Risco     │      Compra (Trade)       │
│ :8081/84 │  :8082/85    │     :8083/86              │
└──────────┴──────────────┴──────────────────────────┘
```

Cada serviço de infraestrutura (Cotação, Risco, Compra) possui **duas réplicas** para resiliência. O Orquestrador conecta-se a elas via um pool round-robin, que alterna entre as réplicas e aplica cooldown em caso de falha de rede.

O Broker expõe duas operações via TCP:
- **pub** (`{"action":"pub","channel":"saga","data":{...}}`): publica um evento, que recebe numeração sequencial (`seq`) e timestamp UTC.
- **pull** (`{"action":"pull","channel":"saga","since":N}`): retorna todos os eventos do canal com `seq > N`.

### Fluxo de uma Transação (Ordem)

1. Orquestrador recebe ordem (ex: `order ETH/USDT USD/BRL 1.5`)
2. Consulta **Cotação** → recebe preços e TTL
3. Cotação publica evento `quotation.received` no Broker (fire-and-forget)
4. Verifica **TTL** — se excedido, aborta
5. Envia dados ao **Risco** → recebe aprovação ou rejeição
6. Verifica **TTL** novamente
7. Compra **Ativo 1** no sistema de **Compra**
8. Verifica **TTL**
9. Compra **Ativo 2** no sistema de **Compra**
10. Se todas as etapas bem-sucedidas → operação concluída
11. Se qualquer compra falhar → aciona **compensação** (venda Last In First Out dos ativos já comprados)

O **Monitor** faz polling a cada 1 segundo no Broker (`pull`), obtém eventos novos (seq > since), escreve em `trace.jsonl` e exibe na tela.

---

## 3. Implementação dos Sockets Berkeley

A camada de comunicação foi construída inteiramente sobre o pacote `net` da linguagem Go, que expõe uma abstração direta sobre as chamadas de sistema POSIX (`socket`, `bind`, `listen`, `accept`, `connect`).

### 3.1. Mensageria com Length-Prefixed Framing (`pkg/tcp/message.go`)

TCP é um protocolo de fluxo (*stream*): uma mensagem enviada por `send()` pode ser fragmentada em múltiplos pacotes, ou múltiplas mensagens agrupadas em um só `recv()`. Para resolver isso, implementamos o padrão **Length-Prefixed Framing**: antes de cada payload JSON, enviamos um cabeçalho fixo de **4 bytes** contendo o tamanho do payload em *big-endian* (*network byte order*). O receptor lê exatamente 4 bytes, interpreta o tamanho, e então lê exatamente esse número de bytes.

```go
func WriteMessage(conn net.Conn, payload []byte) error {
    length := uint32(len(payload))
    lengthBuf := make([]byte, 4)
    binary.BigEndian.PutUint32(lengthBuf, length)
    conn.Write(lengthBuf)
    _, err := conn.Write(payload)
    return err
}

func ReadMessage(conn net.Conn) ([]byte, error) {
    lengthBuf := make([]byte, 4)
    io.ReadFull(conn, lengthBuf)
    length := binary.BigEndian.Uint32(lengthBuf)
    msgBuf := make([]byte, length)
    io.ReadFull(conn, msgBuf)
    return msgBuf, nil
}
```

### 3.2. Cliente TCP — Padrão Request-Reply (`pkg/tcp/client.go`)

O cliente implementa comunicação síncrona: abre conexão TCP, envia requisição, aguarda resposta na mesma conexão e a retorna.

```go
func SendRequest(address string, payload []byte) ([]byte, error) {
    conn, err := net.DialTimeout("tcp", address, 3*time.Second)
    defer conn.Close()
    conn.SetDeadline(time.Now().Add(5 * time.Second))
    WriteMessage(conn, payload)
    return ReadMessage(conn)
}
```

*Timeouts* de dial (3s) e operação (5s) evitam bloqueios permanentes.

### 3.3. Servidor TCP — Padrão Worker Pool (`pkg/tcp/server.go`)

O servidor aceita conexões e as processa concorrentemente, limitando o paralelismo com um semáforo baseado em channel bufferizado:

```go
semaphore := make(chan struct{}, maxWorkers)

for {
    conn, _ := listener.Accept()
    semaphore <- struct{}{}
    go func(c net.Conn) {
        defer func() {
            <-semaphore
            c.Close()
        }()
        req, _ := ReadMessage(c)
        res := s.handler(req)
        WriteMessage(c, res)
    }(conn)
}
```

O buffer do canal (`maxWorkers` = 100) limita as *goroutines* simultâneas. Excedido o limite, novas conexões ficam enfileiradas no `Accept()`.

### 3.4. Adaptadores de Saída — Pool Round-Robin (`pkg/adapter/outbound/`)

Cada serviço possui um cliente que traduz chamadas de domínio em requisições TCP. Todos compartilham um **pool round-robin** que distribui carga entre réplicas e aplica cooldown em caso de erro de rede:

```go
type QuotationClient struct {
    pool *Pool
}

func (c *QuotationClient) Get(req domain.QuotationRequest) (*domain.QuotationResponse, error) {
    addr, err := c.pool.Next()
    payload, _ := json.Marshal(req)
    resPayload, err := tcp.SendRequest(addr, payload)
    if err != nil {
        // Se erro de rede, marca endereço para cooldown
        c.pool.MarkUnhealthy(addr)
        return nil, err
    }
    var res domain.QuotationResponse
    json.Unmarshal(resPayload, &res)
    return &res, nil
}
```

Adaptadores similares existem para Risco (`RiskClient`) e Compra (`TradeClient`).

---

## 4. Padrões de Projeto Utilizados

Foram aplicados **6 padrões de projeto** para programação distribuída:

### 4.1. Length-Prefixed Framing

**Onde:** `pkg/tcp/message.go`

**Descrição:** Cabeçalho de 4 bytes (big-endian) antes de cada payload JSON delimita as mensagens no fluxo TCP, resolvendo o problema de fragmentação (*TCP reassembly*).

**Justificativa:** Sem esse padrão, mensagens poderiam ser recebidas truncadas ou mescladas, inviabilizando a comunicação confiável.

### 4.2. Request-Reply (Integração Síncrona)

**Onde:** `pkg/tcp/client.go` e todos os adaptadores de saída.

**Descrição:** O Orquestrador envia uma requisição a um serviço e bloqueia aguardando a resposta no mesmo stream TCP.

**Justificativa:** O fluxo SAGA é sequencial por natureza — cada etapa depende do resultado da anterior. O Request-Reply reflete esse acoplamento temporal de forma direta e legível.

### 4.3. Worker Pool (Controle de Concorrência)

**Onde:** `pkg/tcp/server.go`

**Descrição:** Semáforo implementado com channel bufferizado limita o número máximo de conexões processadas concorrentemente.

**Justificativa:** O enunciado exige computação paralela. O Worker Pool evita *resource starvation* que surgiria ao criar goroutines ilimitadas por conexão.

### 4.4. Round-Robin Pool (Resiliência e Balanceamento)

**Onde:** `pkg/adapter/outbound/pool.go`

**Descrição:** Pool circular que distribui requisições entre N endereços configurados. Quando um endereço apresenta erro de rede, ele entra em cooldown por um período configurável (`pool_cooldown_ms`), sendo pulado nas rodadas seguintes.

**Justificativa:** Serviços de cotação, risco e compra possuem duas réplicas cada. O pool garante distribuição de carga e tolerância a falhas parciais sem intervenção manual.

### 4.5. SAGA — Orchestrator (Distributed Transaction)

**Onde:** `pkg/application/saga/orchestrator.go`

**Descrição:** Classe central que coordena a transação distribuída em múltiplos passos sequenciais. Cada passo é uma chamada de rede a um serviço diferente. Se qualquer passo falha, ações compensatórias desfazem os passos já concluídos.

```go
func (o *Orchestrator) ExecuteOrder(order domain.OrderRequest) error {
    quoteRes, _ := o.quotation.Get(quoteReq)        // Passo 1: Cotação
    deadline := time.Now().Add(TTLms)
    riskRes, _ := o.risk.Evaluate(riskReq)           // Passo 2: Risco
    // Verifica TTL entre cada passo
    purchaseRes1, _ := o.order.Execute(purchaseReq1) // Passo 3: Compra 1
    purchaseRes2, _ := o.order.Execute(purchaseReq2) // Passo 4: Compra 2
}
```

**Justificativa:** Diferentes serviços não compartilham estado — a falha interna de um não anula o sucesso de outro. SAGA é o padrão estabelecido para consistência eventual em transações distribuídas.

### 4.6. Transação Compensatória (Compensating Transaction)

**Onde:** `pkg/application/saga/orchestrator.go`

**Descrição:** Quando a compra do Ativo 1 é bem-sucedida mas a compra do Ativo 2 falha, o Orquestrador executa uma venda (estorno) do Ativo 1. As compensações são executadas em ordem LIFO (*Last In, First Out*):

```go
defer func() {
    for i := len(rollbacks) - 1; i >= 0; i-- {
        o.rollback(rollbacks[i])
    }
}()
```

Após o sucesso completo, a lista de *rollbacks* é limpa (`rollbacks = nil`) para impedir compensações indevidas.

### 4.7. PubSub — Publicador de Eventos

**Onde:** `pkg/adapter/outbound/pubsub.go` (publicador), `cmd/broker/main.go` (broker), `cmd/monitor/main.go` (consumidor)

**Descrição:** Após processar cada requisição de cotação, o serviço de Cotação publica um evento no Broker contendo os preços e o TTL retornados. O Monitor faz polling periódico e persiste os eventos em `trace.jsonl`.

```go
// Publicação (quotation handler)
event := map[string]any{
    "type":   "quotation.received",
    "asset1": req.Asset1,
    "price1": res.Price1,
    "asset2": req.Asset2,
    "price2": res.Price2,
    "ttl_ms": res.TTLms,
}
h.publisher.Publish("saga", event)
```

**Justificativa:** Permite rastreabilidade e auditoria sem acoplar a lógica de negócio a um sistema de logging específico. O Broker em memória com pull mantém o desacoplamento.

### Quadro Resumo dos Padrões

| Padrão | Localização | Propósito |
|---|---|---|
| Length-Prefixed Framing | `pkg/tcp/message.go` | Delimitação de mensagens em stream TCP |
| Request-Reply | `pkg/tcp/client.go` + adaptadores | Integração síncrona entre processos |
| Worker Pool | `pkg/tcp/server.go` | Controle de concorrência nos servidores |
| Round-Robin Pool | `pkg/adapter/outbound/pool.go` | Balanceamento entre réplicas com cooldown |
| SAGA Orchestrator | `pkg/application/saga/orchestrator.go` | Coordenação da transação distribuída |
| Compensating Transaction | `pkg/application/saga/orchestrator.go` | Reversão de operações em caso de falha |
| PubSub | `cmd/broker/main.go` + `pubsub.go` | Barramento de eventos assíncrono |

---

## 5. Atomicidade e Controle de TTL

### 5.1. Garantia de Atomicidade

A atomicidade é garantida pela combinação de três mecanismos:

1. **Orquestração Centralizada:** O SAGA Orchestrator executa os passos sequencialmente e mantém uma lista de ações bem-sucedidas (`rollbacks`). Se qualquer passo falha, as ações são desfeitas na ordem inversa (LIFO).

2. **Compensação Explícita:** Cada compra bem-sucedida é registrada na slice `rollbacks`. O `defer` no início de `ExecuteOrder` garante que, em caso de erro, todas as compensações sejam executadas antes do retorno. Em caso de sucesso, `rollbacks` é setado para `nil`, impedindo compensações indevidas.

3. **Concorrência isolada por pedido:** A transação de um pedido é estritamente sequencial dentro de uma única goroutine. Concorrência existe apenas entre pedidos distintos (via comando `batch`), cada um com seu próprio estado de rollback isolado.

### 5.2. Controle de TTL (Time-To-Live)

O TTL é recebido da Cotação junto com os preços. Após recebê-lo, o Orquestrador calcula um *deadline*:

```go
deadline := time.Now().Add(time.Duration(quoteRes.TTLms) * time.Millisecond)
```

Antes de cada chamada de rede (Risco, Compra 1, Compra 2), o TTL é verificado:

```go
func checkTTL(deadline time.Time) error {
    if time.Now().After(deadline) {
        return ErrTTLExceeded
    }
    return nil
}
```

Se o TTL for excedido em qualquer ponto, a operação é abortada imediatamente. Se já houver compras realizadas, as compensações são acionadas.

---

## 6. Estrutura do Projeto (Organização em Pacotes)

```
cmd/
├── broker/main.go              ← Broker PubSub (:8090)
├── monitor/main.go             ← Monitor → trace.jsonl
├── operation/main.go           ← Orquestrador (CLI interativo)
├── quotation/main.go           ← Serviço de Cotação
├── risk/main.go                ← Serviço de Risco
└── trade/main.go               ← Serviço de Compra
pkg/
├── tcp/
│   ├── message.go              ← Length-Prefixed Framing
│   ├── client.go               ← Cliente TCP (Request-Reply)
│   └── server.go               ← Servidor TCP (Worker Pool)
├── config/
│   └── config.go               ← Leitura do config.json
├── domain/
│   ├── dto.go                  ← DTOs compartilhados
│   └── ports/
│       ├── inbound.go          ← Interfaces dos serviços
│       └── outbound.go         ← Interfaces dos clientes
├── adapter/
│   ├── inbound/
│   │   ├── quotation.go        ← Handler TCP da Cotação (+ publica evento)
│   │   ├── risk.go             ← Handler TCP do Risco
│   │   └── trade.go            ← Handler TCP da Compra
│   └── outbound/
│       ├── pool.go             ← Pool Round-Robin com cooldown
│       ├── pubsub.go           ← Publicador de eventos (TCP → Broker)
│       ├── quotation.go        ← Cliente de Cotação
│       ├── risk.go             ← Cliente de Risco
│       └── trade.go            ← Cliente de Compra
└── application/
    ├── service/
    │   ├── quotation.go        ← Lógica de cotação (preços aleatórios)
    │   ├── risk.go             ← Lógica de risco (latência + aprovação)
    │   └── trade.go            ← Lógica de compra (latência + sucesso)
    └── saga/
        ├── orchestrator.go     ← Orquestrador SAGA
        └── orchestrator_test.go ← 11 testes unitários
config.json                     ← Configuração para execução local
config.docker.json              ← Configuração para Docker (broker via hostname)
Dockerfile                      ← Build multi-stage (alpine)
docker-compose.yml              ← Orquestração de containers
Makefile                        ← Atalhos de execução
README.md                       ← Guia de uso
```

### Separação por Camadas

- **`pkg/tcp/`** — Infraestrutura de comunicação (sockets). Independe do domínio.
- **`pkg/domain/`** — Definições de dados e contratos (interfaces). Núcleo do domínio.
- **`pkg/adapter/`** — Implementações concretas: `inbound/` recebe requisições TCP, `outbound/` faz chamadas TCP para outros serviços e para o Broker.
- **`pkg/application/`** — Lógica de negócio pura (serviços) e orquestração da transação (SAGA).

Essa separação segue o princípio da **Inversão de Dependência (DIP)**: o domínio define interfaces; os adaptadores implementam a comunicação; o Orquestrador depende apenas das interfaces, não de detalhes de transporte.

---

## 7. Casos de Uso (Simulação de Caos)

Os cenários abaixo demonstram o comportamento do sistema sob diferentes configurações. Execute localmente com `make run-all` ou via Docker.

### 7.1. Cenário 1: Sucesso na Operação

**Configuração:**
- `risk.success_rate: 100.0`
- `purchase.success_rate: 100.0`
- `quotation.ttl_ms: 5000`

**Execução:**
```
> order ETH/USDT USD/BRL 1.8
[SAGA] Iniciando transação distribuída...
 -> [COTAÇÃO] Solicitando preços e TTL para ETH/USDT e USD/BRL...
 <- [COTAÇÃO] ETH/USDT=$2454.35 | USD/BRL=$10.00 | TTL=500ms
 -> [RISCO] Analisando viabilidade da operação...
 <- [RISCO] Operação APROVADA
 -> [COMPRA] Efetuando COMPRA de 1.80 ETH/USDT...
 <- [COMPRA] Ação BUY para ETH/USDT BEM-SUCEDIDA
 -> [COMPRA] Efetuando COMPRA de 1.80 USD/BRL...
 <- [COMPRA] Ação BUY para USD/BRL BEM-SUCEDIDA
[SAGA] OPERAÇÃO CONCLUÍDA COM SUCESSO!
```

**Análise:** O fluxo completo é executado sem interrupções. A cotação retorna preços e TTL, o risco aprova, ambas as compras são bem-sucedidas. Eventos são publicados no Broker e podem ser observados no Monitor.

### 7.2. Cenário 2: Falha por TTL Excedido

**Configuração:**
- `quotation.ttl_ms: 5` (extremamente baixo)

**Execução:**
```
> order ETH/USDT USD/BRL 1.8
[SAGA] Iniciando transação distribuída...
 -> [COTAÇÃO] Solicitando preços e TTL para ETH/USDT e USD/BRL...
 <- [COTAÇÃO] ETH/USDT=$2864.23 | USD/BRL=$10.00 | TTL=5ms
 -> [RISCO] Analisando viabilidade da operação...
 <- [RISCO] Operação APROVADA
 -> [COMPRA] Efetuando COMPRA de 1.80 ETH/USDT...
[SAGA] OPERAÇÃO ABORTADA / ROLLBACK: TTL excedido
```

**Análise:** O TTL de 5ms é tão curto que a latência simulada do risco excede o prazo. O Orquestrador detecta na verificação antes da compra. Como nenhuma compra foi realizada, não há compensação.

### 7.3. Cenário 3: Falha na Compra do Ativo 2 com Compensação

**Configuração:**
- `risk.success_rate: 100.0`
- `purchase.success_rate: 0.0` (força falha em todas as compras)

**Execução:**
```
> order ETH/USDT USD/BRL 1.8
[SAGA] Iniciando transação distribuída...
 -> [COTAÇÃO] Solicitando preços e TTL para ETH/USDT e USD/BRL...
 <- [COTAÇÃO] ETH/USDT=$3629.78 | USD/BRL=$10.00 | TTL=500ms
 -> [RISCO] Analisando viabilidade da operação...
 <- [RISCO] Operação APROVADA
 -> [COMPRA] Efetuando COMPRA de 1.80 ETH/USDT...
 <- [COMPRA] Ação BUY para ETH/USDT BEM-SUCEDIDA
 -> [COMPRA] Efetuando COMPRA de 1.80 USD/BRL...
 <- [COMPRA] Ação BUY para USD/BRL FALHOU
 -> [COMPENSAÇÃO] Efetuando VENDA (estorno) de 1.80 ETH/USDT...
[SAGA] OPERAÇÃO ABORTADA / ROLLBACK: falha na compra do ativo 2
```

**Análise:** Compra do Ativo 1 bem-sucedida, Ativo 2 falha (success_rate: 0.0). O Orquestrador executa a transação compensatória: vende o ETH/USDT comprado, restaurando o estado anterior.

### 7.4. Testes Unitários

O arquivo `pkg/application/saga/orchestrator_test.go` contém **11 testes** que cobrem:

- Fluxo de sucesso (`TestSuccessfulOrderExecution`)
- Falha na cotação (`TestOrderFailsOnQuotationError`)
- Rejeição pelo risco (`TestOrderFailsOnRiskRejection`)
- Erro de rede no risco (`TestOrderFailsOnRiskError`)
- Compensação na falha do Ativo 2 (`TestOrderCompensatesOnFirstAssetPurchaseFailure`)
- Sucesso com ambos os ativos (`TestOrderSucceedsWhenBothAssetsPurchased`)
- TTL expirado (`TestTTLExceededImmediately`)
- Ordem LIFO da compensação (`TestCompensationRevertsInCorrectOrder`)
- Erro de rede no serviço de compra (`TestOrderFailsOnPurchaseServiceError`)
- Independência entre ordens (`TestMultipleOrdersIndependent`)
- Ausência de compensação no sucesso (`TestCompensationDoesNotRunOnSuccess`)

---

## 8. Guia de Execução

### Pré-requisitos

1. **Go 1.22+**: [Download oficial](https://go.dev/dl/)
2. **Make** (opcional, recomendado): já vem no macOS/Linux; no Windows instale via `choco install make`
3. **Docker + Docker Compose** (alternativo): [Docker Desktop](https://www.docker.com/products/docker-desktop/)

### Verificação das dependências

```bash
go version        # go version go1.22.0 ou superior
make --version    # GNU Make 4.x (opcional)
docker compose version  # (opcional)
```

### Execução local (tudo na máquina)

```bash
# Clonar e entrar no diretório
git clone <repo>
cd INE5645-trabalho-dois

# Opção 1: Tudo em um terminal
make run-all
# Isso sobe broker, cotação, risco, compra em background,
# inicia o monitor em background (log em monitor.log + trace.jsonl),
# e abre o CLI interativo.
# Digite 'exit' ou Ctrl+C para encerrar tudo.

# Opção 2: Terminais separados
# Terminal 1: Broker
make run-broker
# Terminal 2: Cotação
make run-quotation
# Terminal 3: Risco
make run-risk
# Terminal 4: Compra
make run-purchase
# Terminal 5: Monitor (opcional)
make run-monitor
# Terminal 6: CLI
make run-cli
```

### Execução com Docker (infraestrutura em containers, CLI local)

```bash
# Sobe broker, cotação, risco, compra em containers
docker compose up --build -d

# Terminal 2: CLI local (conecta nas portas expostas pelo Docker)
make run-cli

# Terminal 3: Monitor local (opcional)
make run-monitor

# Para parar os containers
make docker-down
```

### Comandos do CLI

```
> order ETH/USDT USD/BRL 1.5     # ordem única
> batch ETH/USDT USD/BRL 1.5 5   # lote de 5 ordens concorrentes
> exit                            # sair
```

### Arquivos gerados

| Arquivo | Descrição |
|---|---|
| `trace.jsonl` | Histórico de eventos (uma linha JSON por evento) |
| `monitor.log` | Saída do monitor (apenas no `make run-all`) |

---

## 9. Conclusão

O trabalho implementou com sucesso um sistema de operação distribuído utilizando **sockets Berkeley (TCP) puros** e **7 padrões de projeto** para programação distribuída: **Length-Prefixed Framing**, **Request-Reply**, **Worker Pool**, **Round-Robin Pool**, **SAGA Orchestrator**, **Transação Compensatória** e **PubSub**.

A arquitetura em microsserviços com orquestrador centralizado atende aos requisitos de atomicidade, controle de TTL e tratamento de falhas. Os três cenários obrigatórios (sucesso, falha por TTL, falha com compensação) foram demonstrados e validados com 11 testes unitários automatizados.

A separação do código em camadas (infraestrutura TCP, domínio, adaptadores, serviços e saga) segue os princípios SOLID — especialmente a Inversão de Dependência — permitindo que a lógica de negócio seja testada independentemente do transporte de rede.

A adição do Broker PubSub e do Monitor trouxe rastreabilidade ao sistema sem acoplar a lógica dos serviços a mecanismos de logging específicos, e o pool round-robin com cooldown garante resiliência contra falhas parciais de rede nas réplicas.

---

## Referências

- [1] Microsoft Azure Architecture Center — Cloud Design Patterns. https://learn.microsoft.com/en-us/azure/architecture/patterns/
- [2] Enterprise Integration Patterns. https://www.enterpriseintegrationpatterns.com/
- [3] Chris Richardson — *Microservices Patterns*. Manning, 2018. (Cap. 4: Saga)
- [4] Go net package documentation. https://pkg.go.dev/net
