**Universidade Federal de Santa Catarina - UFSC** **Departamento em Informática e Estatística** **INE 5645 - Programação Paralela e Distribuída** **Semestre 2026/1** 

---

Definição do Trabalho 2: Programação Distribuída 

Este trabalho explora o uso de padrões de comunicação em sistemas distribuídos, tendo como objetivo exercitar a comunicação por troca de mensagens entre processos distribuídos, a adoção de abstrações e a praticidade oferecida por padrões de projeto no desenvolvimento de aplicações distribuídas. 

Como tarefa, você deverá implementar um protótipo de sistema de operação em mercado (trading) automatizado, simulando um fluxo de compra de dois ativos a partir de dados recebidos de sistemas de cotação e análise de risco. O sistema de operação deve ser configurável a fim de permitir o processamento de até N ordens de compra, realizando as quatro integrações a processos independentes (cotação, análise, compra ativo 1, compra ativo 2) de forma confiável e atômica. 

Deve-se assumir que a cotação dos pares possui um tempo limite de vida (TTL), onde em caso excedido antes e depois de cada integração, deve-se abortar a operação de compra dos ativos. Uma falha na compra de qualquer ativo também deve resultar no cancelamento da operação. 

### Exemplos de Execução

Como exemplo, considere os seguinte rastro de execução válido: 

**Exemplo 1: Compras efetuadas com sucesso:** 

1. Sistema de operação gera a ordem de compra dos mercados ETH/USDT (Ethereum/Tether) e USD/BRL (Dólar americano/Real); 


2. Sistema de operação realiza uma requisição ao sistema de cotação informando os dois ativos, e recebe as cotações ETH/USDT: $1,300.00 e USD/BRL $5.00 com um TTL de 300ms; 


3. Sistema de operação envia os dados recebidos ao sistema de risco, que decide por prosseguir com a operação após um tempo de processamento (latência) aleatório; 


4. Sistema de operação envia a requisição de compra do par ETH/USDT ao sistema de compras, que a realiza com sucesso; 


5. Sistema de operação envia a requisição de compra do par USD/BRL ao sistema de compras, que a realiza com sucesso. 



**Exemplo 2: Falha na compra do segundo ativo:** 

1. Sistema de operação gera a ordem de compra dos mercados ETH/USDT e USD/BRL; 


2. Sistema de operação realiza uma requisição ao sistema de cotação informando os dois ativos, e recebe as cotações ETH/USDT: $1,250.00 e USD/BRL $4.90 com um TTL de 400ms; 


3. Sistema de operação envia os dados recebidos ao sistema de risco, que decide por prosseguir com a operação após um tempo de processamento (latência) aleatório; 


4. Sistema de operação envia a requisição de compra do par ETH/USDT ao sistema de compras, que a realiza com sucesso; 


5. Sistema de operação envia a requisição de compra do par USD/BRL ao sistema de compras, que falha; 


6. Sistema de operação aborta a operação realizando a venda do ativo ETH/USDT. 



### Requisitos Técnicos

Como requisito de implementação, é necessário que cada sistema possua uma taxa de sucesso e tempo de processamento (sleep) configuráveis, a fim de modelar diferentes cenários de falha e lentidão na integração. Você deve implementar os quatro processos (sistemas de operação, cotação, risco e compra) em processos distribuídos independentes, utilizando de pelo menos 3 padrões de projeto para programação distribuída em suas integrações. 

---

Requisitos e Avaliação 

Os requisitos específicos são: 

* Implementar os quatro processos utilizando programação distribuída, aplicando padrões de projeto de comunicação; 


* Deve ser entregue o código com a descrição (ex. arquivo README) e detalhes para compilação, implantação e execução de cada processo da aplicação; 


* A linguagem de programação é de livre escolha, mas a implementação da comunicação distribuída deve ser baseada em sockets Berkeley; 


* Relatório redigido em um arquivo .pdf à parte com a explicação sobre a arquitetura de software utilizada e da escolha e adoção dos padrões escolhidos; 


* O relatório deve conter ilustração de casos de uso utilizados para teste da solução, com explicação sobre as saídas de execução. 


* Devem ser apresentados pelo menos os cenários de: (i) sucesso na operação; (ii) falha devido a exceder TTL de cotação; e (iii) falha na integração de compra do ativo 2. 


* Defesa em aula do código. 



---

Entrega 

* O trabalho pode ser realizado em grupos de até 3 participantes. 


* As defesas de código ocorrerão em laboratório, em horário de aula, seguindo o conforme cronograma da disciplina, com o agendamento por grupo informado no moodle. 


* O código-fonte e relatório .pdf devem ser enviados pelo Moodle para análise e avaliação. 


* O nome dos membros do grupo deve aparecer nos artefatos gerados (código e relatório). 


* Nomes que não estejam explícitos nos artefatos entregues não serão considerados na avaliação. 


* Certifique-se de nomear o arquivo zip adequadamente seguindo o formato: Grupo1-NomeA-NomeB.zip 



---

## Referências e Padrões de Projeto

Alguns padrões de projeto foram apresentados em aula, mas não são os únicos. Os grupos podem pesquisar outros padrões. Neste caso, confirmem com o professor caso escolham outro padrão, para evitar escolhas equivocadas sem relação com a programação distribuída. Algumas referências para padrões de projeto para computação distribuída aparecem em: 

* [1] Microsoft Cloud Design Patterns: [https://learn.microsoft.com/en-us/azure/architecture/patterns/](https://learn.microsoft.com/en-us/azure/architecture/patterns/) 


* [2] Enterprise Integration Patterns: [https://www.enterpriseintegrationpatterns.com/patterns/messaging/](https://www.enterpriseintegrationpatterns.com/patterns/messaging/)
