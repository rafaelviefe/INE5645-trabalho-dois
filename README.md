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

Há duas formas de rodar o projeto. A **recomendada** é com `make run-dev`, que executa todos os serviços localmente com `go run` (sem cache, sem Docker, lê o `config.json` sempre atualizado). A alternativa via Docker também está disponível.

### Modo Rápido (Recomendado) - `make run-dev`

Na raiz do projeto, execute:

```bash
make run-dev
```

Isso sobe os 3 serviços (Cotação, Risco, Compra) em background com `go run` e abre o CLI interativo. Para sair, digite `exit` ou pressione `Ctrl+C` — os serviços em background são encerrados automaticamente.

### Modo Alternativo - Docker

Os serviços podem rodar em containers Docker, enquanto o **CLI (Orquestrador)** roda localmente no terminal. Use este modo se preferir isolar os microsserviços.

**Passo 1:** Subir os serviços satélites:

```bash
make docker-up
```

*(Equivalente a `docker compose up --build -d`)*

**Passo 2:** Em outro terminal, iniciar o CLI:

```bash
make run-cli
```

**Passo 3:** Executar ordens como `order ETH/USDT USD/BRL 1.5`.

Para encerrar:

```bash
make docker-down
```

---

## Validando os Casos de Uso (Simulação de Caos)

O sistema lê as configurações de um arquivo central `config.json`. Ao alterá-lo, podemos forçar os cenários de falha exigidos.

**Importante sobre o config.json:**
- Com `make run-dev` (recomendado): os serviços leem o arquivo ao vivo — basta salvar as alterações, reiniciar com `make run-dev` e os novos valores serão aplicados. **Sem cache.**
- Com Docker: a imagem tem o `config.json` copiado em build time. É necessário rodar `make docker-up` novamente para rebuildar as imagens com o novo arquivo.

### Cenário 1: Sucesso na Operação

Abra o `config.json` e garanta que `success_rate` de Risco e Purchase estão em `100.0` e o `ttl_ms` da Cotação seja alto (ex: `5000`).

* **Teste:** Digite a ordem no CLI. O fluxo passará pela cotação, será aprovado no risco e efetuará a compra dos dois ativos atômica e sequencialmente.

### Cenário 2: Falha devido a exceder TTL de Cotação

Abra o `config.json` e altere o parâmetro `"ttl_ms"` da propriedade `quotation` para um valor absurdamente baixo, como `5` milissegundos.

* **Teste:** Digite a ordem no CLI. O sistema buscará a cotação e, devido ao controle rígido de tempo (implementado via `time.Since` em Go) checado antes e depois de cada integração de rede, a operação será cancelada quase imediatamente com o aviso `[SAGA] OPERAÇÃO ABORTADA / ROLLBACK: TTL excedido`.

### Cenário 3: Falha na Integração de Compra do Ativo 2 (Transação Compensatória)

Abra o `config.json` e altere o parâmetro `"success_rate"` da propriedade `purchase` para `0.0`.

* **Teste:** Digite a ordem no CLI. A transação falhará ao tentar comprar os ativos. Quando isso ocorrer após a primeira compra (ou durante a segunda), o Orquestrador SAGA invocará automaticamente a função de compensação. Você verá o terminal emitir um alerta em amarelo informando: `[COMPENSAÇÃO] Efetuando VENDA (estorno)`.

---

## Arquitetura e Padrões de Projeto Distribuídos

Para atender ao escopo da disciplina e desacoplar o domínio da infraestrutura, o sistema implementa os seguintes padrões de projeto:

### 1. Comunicação Base (Sockets Berkeley & Mensageria)

* A comunicação em rede foi construída totalmente do zero utilizando a primitiva `net` do Go, que abstrai as chamadas C nativas de sockets POSIX (`socket`, `bind`, `listen`, `accept`, `connect`).

* **Padrão *Length-Prefixed Framing* (Mensageria):** Para resolver o problema inerente de quebra de pacotes em streams TCP, implementamos em `pkg/tcp/message.go` um protocolo customizado onde um cabeçalho de 4 bytes (um inteiro de 32 bits indicando o tamanho do payload) é enviado sempre antes do JSON. Isso garante que a mensagem inteira seja lida de forma atômica no destino.

### 2. Controle de Concorrência (Worker Pool)

* Visando suportar processamento paralelo massivo, os servidores implementam o padrão de *Worker Pool* controlado por Semáforos. Em `pkg/tcp/server.go`, utilizamos um `Channel` bufferizado do Go. Isso limita a quantidade máxima de Goroutines (threads) executando simultaneamente e enfileira novas conexões, protegendo a máquina de esgotamento de recursos e aplicando o aprendizado prático de computação paralela exigido.

### 3. Padrões de Integração e Orquestração

* **Padrão *Request-Reply*:** A comunicação entre os serviços se dá de forma síncrona sobre o protocolo TCP. O cliente (Orquestrador) envia um payload de requisição e se bloqueia aguardando a resposta no mesmo stream (`pkg/tcp/client.go`), simplificando a máquina de estados.
* **Padrão *SAGA (Orchestrator)*:** A lógica central (`pkg/saga/orchestrator.go`) funciona como um Orquestrador centralizado que dita o fluxo da transação. Ele chama o sistema de Cotação, avalia o TTL, envia para Risco, reavalia o TTL, tenta Comprar o Ativo 1 e, por fim, o Ativo 2.
* **Padrão *Compensating Transaction*:** Conectado ao SAGA, esse padrão garante a atomicidade em ambientes onde não existe "commit/rollback" nativo como em bancos de dados relacionais. Se qualquer etapa falhar após uma etapa anterior ter sido efetivada (ex: Compra 2 falhar ), o Orquestrador executa ativamente uma requisição de Ação Reversa (`domain.ActionSell`) para estornar os fundos do ativo comprado, restabelecendo a consistência e cancelando a operação principal.