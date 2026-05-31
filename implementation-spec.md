### O Plano de Implementação (Roadmap)

#### Fase 1: Fundação da Comunicação (Agnóstica ao Domínio)

O objetivo desta fase é criar nossa própria "biblioteca" de comunicação distribuída usando Sockets TCP puros, isolando totalmente a lógica de rede da lógica de negócios.

* 1.1. Protocolo de Mensageria Customizado: Como Sockets TCP orientados a fluxo (*stream*)  sofrem com o problema de "quebra" de mensagens, criaremos um empacotamento simples (ex: cabeçalho com o tamanho do payload + payload em JSON).


* **1.2. Padrão *Request-Reply* (Síncrono/Assíncrono):** Implementação de um cliente e um servidor TCP genéricos. O servidor será capaz de escutar na porta usando concorrência (uma *goroutine* por conexão), satisfazendo a premissa de uso de computação paralela.
* 1.3. Padrão *Worker Pool* (Concorrência): Baseado no Exercício 4 do slide 13, implementaremos um *pool* de *threads* (*goroutines*) no servidor para não sobrecarregar a máquina caso haja milhares de requisições, controlando o processamento em paralelo de forma segura.



#### Fase 2: Padrões de Projeto Distribuídos e Core do Domínio (SOLID)

*Definir as interfaces, os contratos de dados e implementar o esqueleto do Orquestrador.*

* **2.1. Definição dos Contratos (DTOs):** Criação das estruturas (structs) JSON que trafegarão pela rede: Requisições de Ordem, Respostas de Cotação, Análise de Risco, Sucesso/Falha de Compra.
* **2.2. O Padrão SAGA (Orquestração):** Modelaremos o "Sistema de Operação" como um **Saga Orchestrator**. Ele será uma máquina de estados que executará os passos sequenciais.
* 2.3. Transações Compensatórias (*Compensating Transactions*): Como exigido, se a compra do ativo 2 falhar , o Saga Orchestrator deve acionar uma requisição de compensação (venda/cancelamento) do ativo 1. Isso será desenhado explicitamente no código.


* **2.4. Injeção de Dependência:** As regras de negócios não farão chamadas diretas ao TCP. Criaremos interfaces (ex: `QuotationService`) que serão injetadas no Orquestrador.

#### Fase 3: Desenvolvimento dos Sistemas Satélites (Mock Services)

*Criação dos três microsserviços periféricos com suporte a simulação do mundo real (caos).*

* **3.1. Sistema de Configuração (YAML/JSON):** Criar um módulo para ler arquivos de configuração. Cada sistema terá parâmetros para: `porta_tcp`, `probabilidade_de_falha` (%), `tempo_min_sleep_ms`, `tempo_max_sleep_ms`. Isso é vital para simular latência de rede e falhas no Risco/Compra.


* 3.2. Sistema de Cotação: Implementar o servidor que recebe o par de ativos e devolve valores aleatórios atrelados a um **TTL dinâmico e configurável**.


* 3.3. Sistema de Análise de Risco: Implementar o servidor que avalia a requisição, aplica a latência simulada (sleep aleatório) e retorna aprovação/rejeição com base na taxa de sucesso configurada.


* 3.4. Sistema de Compras: Implementar o servidor capaz de executar as ordens (compra e venda/compensação), também sujeito a falhas programadas.



#### Fase 4: O Coração do Sistema - O Sistema de Operação e UX

*Unir todas as peças no Orquestrador e criar a interface com o usuário.*

* **4.1. Controle Rígido de TTL:** Implementar a lógica crucial do projeto: a verificação estrita do tempo. Criaremos um temporizador/cronômetro (`time.Since`) que será validado *antes e depois* de cada integração (Risco, Compra 1, Compra 2) para garantir o aborto se exceder o TTL da cotação.


* **4.2. Usabilidade (Interface via CLI):** O Sistema de Operação terá um shell interativo (CLI) muito amigável. O usuário poderá digitar comandos como:
* `> order --asset1 ETH/USDT --asset2 USD/BRL --qty 1.5`
* Isso aciona o fluxo SAGA de forma visível, imprimindo logs formatados e coloridos no terminal para o professor visualizar exatamente em que passo a transação está.


* 4.3. Testes dos Casos de Uso: Garantir via logs ou testes de integração que os 3 cenários obrigatórios  ocorrem perfeitamente (Sucesso, Falha por TTL, Falha na Compra 2).



#### Fase 5: Infraestrutura, Entrega e Relatório

*Pacote final do protótipo.*

* **5.1. Docker e Docker Compose:** Escreveremos o `Dockerfile` para os binários em Go (usando *multi-stage build* para ficarem minúsculos, com poucos MBs) e o `docker-compose.yml`. Com um único `docker-compose up`, os 4 nós subirão em portas distintas, comunicando-se pela rede interna do Docker.
* 5.2. Relatório de Arquitetura (Bônus): Posso ajudar a estruturar os tópicos e textos para o relatório `.pdf` exigido, justificando o uso do SAGA, Request-Reply e Worker Pool.


* 5.3. Scripts de Inicialização Rápida (`Makefile` ou scripts `.sh`): Para compilar, rodar local ou via Docker facilmente, ajudando na defesa.