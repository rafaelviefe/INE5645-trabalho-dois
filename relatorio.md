# Relatório do Trabalho 2

**Universidade Federal de Santa Catarina - UFSC**
**Disciplina:** INE 5645 - Programação Paralela e Distribuída
**Semestre:** 2026/1
**Equipe:** Arthur Schurhaus, Rafael Vieira, Uriel Jaloto

---

## 1. Introdução
Este relatório apresenta um protótipo de sistema de operação em mercado automatizado (*trading*), desenvolvido para explorar conceitos de programação distribuída. O sistema simula o fluxo completo de compra de dois ativos: um orquestrador central coordena a transação, consultando serviços independentes de cotação, risco e execução de compras.

A comunicação entre todos os processos é feita exclusivamente via **sockets Berkeley (TCP)**, implementados manualmente sobre o pacote `net` da linguagem Go — que expõe diretamente as chamadas de sistema POSIX (`socket`, `bind`, `listen`, `accept`, `connect`). Nenhuma biblioteca externa de comunicação foi utilizada, atendendo ao requisito de implementação baseada em sockets Berkeley.

## 2. Arquitetura do Sistema
O sistema é composto por seis processos independentes, cada um rodando como um binário separado:

- **Orquestrador:** coordena a transação via linha de comando
- **Cotação:** fornece preços simulados e prazo de validade (TTL)
- **Risco:** avalia se a operação é viável
- **Compra (Trade):** executa as ordens de compra e venda
- **Broker (PubSub):** barramento de eventos para rastreabilidade
- **Monitor:** consome eventos do Broker e gera histórico

Cada serviço de infraestrutura (Cotação, Risco, Compra) possui **duas réplicas** para resiliência. O Orquestrador alterna entre elas com um pool round-robin que aplica um período de "cooldown" quando uma réplica falha.

Toda a comunicação entre processos usa **sockets Berkeley (TCP)** — o Orquestrador abre conexões TCP para cada serviço, envia requisições em formato JSON e aguarda respostas no mesmo stream. Para lidar com a natureza de fluxo contínuo do TCP (que pode fragmentar ou agregar mensagens), implementamos um cabeçalho de 4 bytes antes de cada payload delimitando o tamanho da mensagem.

```
┌──────────┐     ┌──────────────┐     ┌──────────┐
│ Monitor  │◄────│    Broker    │◄────│ Cotação  │
│          │pull │              │ pub │          │
└──────────┘     └──────────────┘     └──────────┘

┌──────────────────────────────────────────────────┐
│              Orquestrador (CLI)                    │
├──────────┬──────────────┬─────────────────────────┤
│ Cotação  │    Risco     │     Compra (Trade)       │
│ :8081/84 │  :8082/85    │    :8083/86              │
└──────────┴──────────────┴─────────────────────────┘
```

### Fluxo de uma Transação
1. Orquestrador recebe ordem (ex: `order ETH/USDT USD/BRL 1.5`)
2. Consulta **Cotação** → recebe preços e TTL
3. Cotação publica evento no Broker (fire-and-forget)
4. Verifica **TTL** — se excedido, aborta
5. Consulta **Risco** → recebe aprovação ou rejeição
6. Compra **Ativo 1** no sistema de **Compra**
7. Compra **Ativo 2** no sistema de **Compra**
8. Se ambas bem-sucedidas → operação concluída
9. Se qualquer compra falhar → **compensação** (vende os ativos já comprados, em ordem LIFO)

## 3. Padrões de Projeto Utilizados
Aplicamos **3 padrões de projeto** para programação distribuída:

### 3.1. Round-Robin Pool (Réplicas)
Cada serviço de infraestrutura (Cotação, Risco, Compra) possui duas réplicas. O Orquestrador usa um pool circular que distribui as requisições entre elas. Quando uma réplica apresenta erro de rede, entra em um período de cooldown configurável e é pulada nas rodadas seguintes. Isso garante distribuição de carga e tolerância a falhas parciais sem intervenção manual.

### 3.2. SAGA com Transação Compensatória
O Orquestrador coordena a transação distribuída em passos sequenciais: consulta Cotação, consulta Risco, compra Ativo 1, compra Ativo 2. Se a compra do Ativo 1 vai bem mas a do Ativo 2 falha, o sistema automaticamente estorna o Ativo 1 (vende de volta). As compensações seguem ordem LIFO — a última compra é a primeira a ser desfeita. Isso garante atomicidade sem usar transação distribuída tradicional, que não faria sentido entre serviços independentes.

### 3.3. PubSub
A Cotação publica eventos no Broker após cada requisição de preços. O Monitor faz polling periódico no Broker, obtém os eventos novos e persiste tudo em `trace.jsonl`, além de exibir em tempo real. Isso dá rastreabilidade total sem acoplar a lógica de negócio a um sistema de logging específico.

## 4. Casos de Uso

### 4.1. Sucesso na Operação

**Configuração:** `risk.success_rate: 100%`, `purchase.success_rate: 100%`

```
> order ETH/USDT USD/BRL 1.8
 -> [COTAÇÃO] Solicitando preços...
 <- [COTAÇÃO] ETH/USDT=$2454.35 | USD/BRL=$10.00 | TTL=500ms
 -> [RISCO] Analisando viabilidade...
 <- [RISCO] Operação APROVADA
 -> [COMPRA] COMPRA de 1.80 ETH/USDT...
 <- [COMPRA] BEM-SUCEDIDA
 -> [COMPRA] COMPRA de 1.80 USD/BRL...
 <- [COMPRA] BEM-SUCEDIDA
OPERAÇÃO CONCLUÍDA COM SUCESSO!
```

**Análise:** Fluxo completo executado sem interrupções — cotação retorna preços, risco aprova, ambas as compras são bem-sucedidas. Todos os eventos são publicados no Broker e ficam visíveis no Monitor. Exatamente o "caminho feliz".

### 4.2. Falha por TTL Excedido

**Configuração:** `quotation.ttl_ms: 5` (extremamente baixo)

```
> order ETH/USDT USD/BRL 1.8
 -> [COTAÇÃO] Solicitando preços...
 <- [COTAÇÃO] ETH/USDT=$2864.23 | USD/BRL=$10.00 | TTL=5ms
 -> [RISCO] Analisando viabilidade...
 <- [RISCO] Operação APROVADA
 -> [COMPRA] COMPRA de 1.80 ETH/USDT...
OPERAÇÃO ABORTADA: TTL excedido
```

**Análise:** TTL de 5ms é tão curto que a latência simulada do risco (sleep configurável) excede o prazo. O Orquestrador detecta na verificação antes da primeira compra e aborta. Como nenhuma compra foi realizada, não há compensação. O sistema prefere abortar a operar com dados vencidos.

### 4.3. Falha na Compra do Ativo 2 com Compensação

**Configuração:** `risk.success_rate: 100%`, `purchase.success_rate: 0%` (força falha)

```
> order ETH/USDT USD/BRL 1.8
 -> [COTAÇÃO] Solicitando preços...
 <- [COTAÇÃO] ETH/USDT=$3629.78 | USD/BRL=$10.00 | TTL=500ms
 -> [RISCO] Analisando viabilidade...
 <- [RISCO] Operação APROVADA
 -> [COMPRA] COMPRA de 1.80 ETH/USDT...
 <- [COMPRA] BEM-SUCEDIDA
 -> [COMPRA] COMPRA de 1.80 USD/BRL...
 <- [COMPRA] FALHOU
 -> [COMPENSAÇÃO] VENDA (estorno) de 1.80 ETH/USDT...
OPERAÇÃO ABORTADA: falha na compra do ativo 2
```

**Análise:** Risco aprova, compra do Ativo 1 vai bem, mas a compra do Ativo 2 falha. O Orquestrador imediatamente executa a compensação: vende o Ativo 1 já comprado. O estado final é consistente — como se a ordem nunca tivesse existido. A compensação segue ordem LIFO (último a comprar é o primeiro a vender).

### 4.4. Testes Unitários
Onze testes automatizados cobrem sucesso, falha em cada etapa, TTL expirado, compensação LIFO, erros de rede, independência entre ordens concorrentes e ausência de compensação indevida no sucesso.

## 5. Conclusão
O trabalho implementou com sucesso um sistema de operação distribuída usando **sockets Berkeley (TCP)** — implementados manualmente em Go sem bibliotecas externas — e **3 padrões de projeto** (Round-Robin Pool, SAGA com compensação, PubSub). A arquitetura atende aos requisitos de atomicidade, controle de TTL e tolerância a falhas parciais.

Os três cenários obrigatórios (sucesso, falha por TTL, falha com compensação) foram demonstrados com execuções reais e validados por 11 testes automatizados. O código-fonte, guia de compilação e instruções de execução estão disponíveis no README do repositório.
