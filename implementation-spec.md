### Histórico de Implementação (Roadmap)

#### Fase 1: Fundação da Comunicação (Agnóstica ao Domínio)

* 1.1. ✅ Protocolo de Mensageria Customizado — Empacotamento length-prefixed em `pkg/tcp/message.go`
* 1.2. ✅ Padrão *Request-Reply* — Cliente e servidor TCP genéricos em `pkg/tcp/client.go` e `pkg/tcp/server.go`
* 1.3. ✅ Padrão *Worker Pool* — Semáforo com channel bufferizado no servidor

#### Fase 2: Padrões de Projeto Distribuídos e Core do Domínio (SOLID)

* 2.1. ✅ Definição dos Contratos (DTOs) — `pkg/domain/dto.go`
* 2.2. ✅ Padrão SAGA (Orquestração) — `pkg/application/saga/orchestrator.go`
* 2.3. ✅ Transações Compensatórias — Rollback via SELL em caso de falha
* 2.4. ✅ Injeção de Dependência — Interfaces em `pkg/domain/ports/`

#### Fase 3: Desenvolvimento dos Sistemas Satélites (Mock Services)

* 3.1. ✅ Sistema de Configuração — `pkg/config/config.go`
* 3.2. ✅ Sistema de Cotação — `cmd/quotation/main.go`
* 3.3. ✅ Sistema de Análise de Risco — `cmd/risk/main.go`
* 3.4. ✅ Sistema de Compras — `cmd/trade/main.go`

#### Fase 4: O Coração do Sistema — O Sistema de Operação e UX

* 4.1. ✅ Controle Rígido de TTL — Verificação antes/depois de cada integração
* 4.2. ✅ Usabilidade (CLI) — Shell interativo com comandos `order` e `batch`
* 4.3. ✅ Testes dos Casos de Uso — Unitários e integração com cenários de falha

#### Fase 5: Infraestrutura, Entrega e Relatório

* 5.1. ✅ Docker e Docker Compose — `Dockerfile` multi-stage, `docker-compose.yml`
* 5.2. ✅ Round-Robin Pool — `pkg/adapter/outbound/pool.go` com cooldown para falhas de rede
* 5.3. ✅ PubSub Broker + Monitor — `cmd/broker/main.go` e `cmd/monitor/main.go`
* 5.4. ✅ Correção de Bug — `pkg/application/service/quotation.go` (price2 usava s.maxPrice duas vezes)
* 5.5. ✅ Remoção do Circuit Breaker — Substituído pelo Pool (cooldown não bloqueia compensação)
