# Trabalho 2 - INE5645: Programação Distribuída

**Alunos:** Arthur Schurhaus, Rafael Vieira e Uriel Jaloto

---

## Pré-requisitos (Instalação do Zero)

Para executar este projeto em uma máquina sem configurações prévias, você precisará das seguintes ferramentas:

1. **Go (Golang) v1.22+**: Necessário para compilar e rodar todos os serviços.
   * *Linux*: `sudo apt install golang`
   * *Windows/Mac*: Baixe o instalador no [site oficial do Go](https://go.dev/dl/).

2. **Make**: (Opcional, mas recomendado) Para utilizar os atalhos de execução. No Linux/Mac geralmente já vem instalado (ou `sudo apt install make`). No Windows, instale via `choco install make` ou utilize os comandos puros listados no `Makefile`.

3. **Docker e Docker Compose**: (Alternativo) Necessário apenas se quiser rodar os microsserviços em containers isolados. Instale o [Docker Desktop](https://www.docker.com/products/docker-desktop/) (Windows/Mac) ou via gerenciador de pacotes no Linux (`sudo apt install docker-compose-plugin`).

---

## Como Executar o Projeto

Há duas formas de rodar o projeto.

### Modo Rápido (Recomendado) - `make run-all`

```bash
make run-all
```

Isso sobe o broker, cotação, risco e compra em background e abre o CLI interativo. Para sair, digite `exit` ou pressione `Ctrl+C`.

### Monitor de Eventos

Em outro terminal:

```bash
make run-monitor
```

O monitor se conecta ao broker, faz polling a cada 1s e exibe eventos em tempo real, salvando em `trace.jsonl`.

### Modo Alternativo - Docker

```bash
make docker-up
make run-cli
make run-monitor   # opcional: monitorar eventos
make docker-down
```

---

## Validação dos Cenários

O sistema lê as configurações do arquivo `config.json`.

### Cenário 1: Sucesso na Operação

`success_rate` de Risco e Purchase em `100.0`, `ttl_ms` alto (ex: `5000`). A ordem passa por cotação, risco e compra dos dois ativos.

### Cenário 2: Falha por TTL Excedido

Altere `"ttl_ms"` para `5`. O sistema aborta com `TTL excedido` antes ou durante as etapas seguintes.

### Cenário 3: Falha na Compra do Ativo (Compensação)

Altere `"success_rate"` de `purchase` para `0.0`. A compra falha e o orquestrador executa a transação compensatória (SELL) para estornar ativos comprados.

### Cenário 4: Resiliência com Pool Round-Robin + Replicas

Com `quotation_addrs`, `risk_addrs` e `purchase_addrs` configurados com múltiplos endereços, se uma réplica falhar (erro de rede), o pool pula para a próxima após o `pool_cooldown_ms`.

---

## Arquitetura

### Componentes

| Componente | Porta | Descrição |
|---|---|---|
| Broker | `:8090` | PubSub in-memory, armazena eventos com numeração sequencial |
| Monitor | CLI | Conecta ao broker, persiste em `trace.jsonl` |
| Quotation | `:8081` / `:8084` | Gera cotações com TTL |
| Risk | `:8082` / `:8085` | Análise de risco com latência simulada |
| Purchase (Trade) | `:8083` / `:8086` | Executa compras e vendas compensatórias |
| Operation (CLI) | stdin | Orquestrador SAGA com CLI interativo |

### Padrões de Projeto

1. **Length-Prefixed Framing** (`pkg/tcp/message.go`): Protocolo TCP com cabeçalho de 4 bytes indicando tamanho do payload.

2. **Worker Pool** (`pkg/tcp/server.go`): Semáforo com channel bufferizado limita goroutines concorrentes.

3. **SAGA Orchestrator** (`pkg/application/saga/orchestrator.go`): Orquestrador centralizado que coordena cotação → TTL → risco → TTL → compra 1 → compra 2.

4. **Compensating Transaction**: Se uma compra falha após outra ter sido efetivada, o orquestrador emite uma venda (SELL) para estornar.

5. **Round-Robin Pool** (`pkg/adapter/outbound/pool.go`): Distribui requisições entre réplicas, com cooldown para endereços com falha de rede.

6. **PubSub (Event Publisher)** (`pkg/adapter/outbound/pubsub.go`): Fire-and-forget, publicação assíncrona para o broker.
