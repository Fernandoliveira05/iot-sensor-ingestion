# IoT Sensor Ingestion

## Objetivo do projeto

Este projeto implementa uma arquitetura de ingestão assíncrona de telemetria para dispositivos IoT. O sistema recebe leituras de sensores por meio de uma API HTTP, publica essas mensagens em uma fila RabbitMQ e, em um processo separado, um consumer consome a fila e persiste os dados em um banco PostgreSQL.

O objetivo principal é desacoplar a etapa de recebimento da etapa de persistência, aumentando a escalabilidade e reduzindo o tempo de resposta da API mesmo sob carga concorrente.

## Arquitetura

Fluxo principal:

1. O cliente envia uma requisição `POST /telemetria` para o backend.
2. O backend valida o payload e publica a mensagem na fila `task_queue` no RabbitMQ.
3. O consumer consome mensagens da fila.
4. O consumer converte e persiste os dados na tabela `telemetria` no PostgreSQL.

```mermaid 
   graph LR
    A[K6] -->|POST /telemetria| B(Backend Go)
    B -->|Publish| C{RabbitMQ}
    C -->|Consume| D(Consumer Go)
    D -->|Insert| E[(PostgreSQL)]
```

### Separação entre backend e consumer

- **Backend**: responsável apenas por receber, validar e enfileirar.
- **Consumer**: responsável por consumir da fila e persistir no banco.

Essa separação evita que a API fique bloqueada por operações de escrita em banco e permite escalar consumidores independentemente do backend.

## Decisões técnicas

- **Go** para backend e consumer, pela simplicidade de concorrência e bom desempenho. Além de ser o material dado pelo professor.
- **Gin** para expor a API HTTP. Go deixa desenvolvermos servidores sem a necessidade de bibliotecas, mas o GIN facilita a nossa vida. 
- **RabbitMQ** como fila para desacoplamento entre ingestão e persistência.
- **PostgreSQL** para persistência relacional das leituras.
- **Docker Compose** para orquestrar todo o ambiente.
- **k6** para testes de carga.

## Serviços

- `backend`: API HTTP que recebe e enfileira leituras.
- `consumer`: processo assíncrono que lê a fila e grava no banco.
- `rabbitmq`: broker de mensagens.
- `postgres`: banco de dados relacional.
- `k6`: executor de carga para validar throughput, latência e taxa de erro.

## Variáveis de ambiente

### Backend e Consumer

- `DB_URL`
  - Finalidade: string de conexão com o PostgreSQL.
  - Valor padrão:
    - `postgres://postgres:postgres@postgres:5432/telemetria?sslmode=disable`

- `RABBIT_URL`
  - Finalidade: string de conexão com o RabbitMQ.
  - Valor padrão:
    - `amqp://guest:guest@rabbitmq:5672/`

- `QUEUE_NAME`
  - Finalidade: nome da fila consumida/publicada.
  - Valor padrão:
    - `task_queue`

## Formato do payload

### Leitura analógica

```json
{
  "device_id": "sensor-001",
  "timestamp": "2026-03-23T04:58:48Z",
  "sensor_type": "temperature",
  "reading_type": "analog",
  "value": 25.4
}
```

### Leitura discreta
```json 
{
  "device_id": "sensor-002",
  "timestamp": "2026-03-23T04:58:48Z",
  "sensor_type": "presence",
  "reading_type": "discrete",
  "value": "detected"
}
```

Também são aceitos valores booleanos em leituras discretas, que podem ser persistidos como "true" ou "false". Coloquei desse jeito pois o enunciado menciona leituras de diversos tipos, então decidi generalizar. Importante salientar que para isso precisei de auxílio de Inteligência Artificial, pois não tinha domínio de como deixar uma coluna mais generalista em SQL. O resultado você pode ver no arquivo "init.sql". 

## Contrato da API

### GET /ping
Endpoint de Health Check da API. É o clássico ping-pong. 

Exemplo de resposta:
```json
{
  "message": "pong"
}
```

### POST /telemetria
Recebe um JSON, valida os campos e publica a mensagem na fila. 

Resposta de sucesso:
- Status: 202 Accepted

Exemplo de resposta:

```json
{
  "message": "Enviado para processamento assíncrono",
  "data": {
    "device_id": "sensor-001",
    "timestamp": "2026-03-23T04:58:48Z",
    "sensor_type": "temperature",
    "reading_type": "analog",
    "value": 25.4
  }
}
```

> ### Tive um erro, Fernando! O que fazer?
> Calma! Esses erros existem mesmo, mas são bem fáceis de entender. Confira as causas abaixo:
> 
> * **`400 Bad Request` (Erro de Validação):**
>     * JSON malformado (faltando aspas ou vírgulas).
>     * `timestamp` ausente ou fora do padrão ISO 8601.
>     * `reading_type` inválido (deve ser `analog` ou `discrete`).
>     * Valor incompatível com o tipo de leitura enviado.
> * **`500 Internal Server Error`:**
>     * Falha na comunicação com o broker.
>     * Erro ao tentar publicar a mensagem na fila do RabbitMQ.

### GET /telemetria
Get simples para receber o que existe no seu banco de dados!

Exemplo de resposta:
```json
{
    "device_id":"186-57",
    "timestamp":"2026-03-23T05:02:57.207Z",
    "sensor_type":"temperature",
    "reading_type":"analog",
    "value":20.576501395117795
}
```

## Como rodar o projeto?
Coloquei tudo em containers do Docker para facilitar a sua vida! basta rodar na **raíz do projeto** (/iot-sensor-ingestion ou também conhecido como lugar onde o **docker-compose.yaml** está) o seguinte comando:

```bash
docker compose up --build
```

Para derrubar os containers rode:

```bash
docker compose down
```

### Como rodar o K6?
O k6 é uma ferramenta para testes de carga e performance, desenvolvida pela Grafana Labs. Ele permite que escrevamos cenários de teste usando JavaScript, simulando milhares de usuários simultâneos (VUs, Virtual Users) para validar a escalabilidade e a estabilidade de APIs, microsserviços e infraestruturas críticas. <br/>
Ele tá presente no nosso projeto! O código dele tá em "/loadtest/loadtest.js". Lá você pode entender um pouco melhor sobre como o K6 funciona em código! <br/>
Vale mencionar também que a primeira versão do código do K6 foi gerado por Inteligência Artificial, sendo posteriormente corrigido por mim. 

Eu adicionei a inicialização do K6 ao docker compose principal do projeto, mas você pode querer rodá-lo mais uma vez. Nesse caso, você pode rodar o seguinte comando:

```bash
docker compose run --rm k6
```

## Como verificar o RabbitMQ e o PostgreSQL?
O RabbitMQ e o PostgreSQL são uma parte importante do nosso projeto! O RabbitMQ trata de enfileirar as nossas muitas requisições e o PostgreSQL armazena tudo isso de forma organizada em seu banco de dados. 

### RabbitMQ

Acesse:

- http://localhost:15673

Credenciais:

- Usuário: guest
- Senha: guest

No painel, verifique:

- existência da fila "task_queue"
- número de mensagens prontas
- número de consumidores
- taxa de publish e consume

**Figura 1 - Exemplo do RabbitMQ aberto**

![RabbitMQ rodando](assets/rabbitmq.png)

*Fonte: O autor (2026).*

### PostgreSQL:
Entrar no container:

```bash
docker exec -it telemetry_postgres psql -U postgres -d telemetria
```

Consultar dados:

```sql
SELECT * 
FROM telemetria
```

**Figura 2 - Exemplo de consulta no PostgreSQL via terminal**

![PostgreSQL sendo usado para consulta via terminal, através do Docker](assets/postgres.png)

*Fonte: O autor (2026).*


## Limitações conhecidas

- O startup depende do tempo de prontidão do RabbitMQ e do PostgreSQL, por isso foram necessários retries. Sem eles, o sistema quase nunca iniciava corretamente. 

- O schema atual não impede explicitamente preenchimento simultâneo de analog_value e discrete_value. Dependemos do bom senso. 

- O k6 inicialmente exigiu ajustes de sincronização para não iniciar antes da stack.


## Testes de carga

Os testes de carga foram feitos com o objetivo de avaliar o comportamento da arquitetura sob volume crescente de requisições concorrentes. Mais especificamente para verificar se a API de ingestão de telemetria conseguia manter baixa latência, baixa taxa de erro e bom throughput mesmo com múltiplos clientes enviando dados ao mesmo tempo.

Como a arquitetura adotada é assíncrona, o foco principal desses testes não foi medir o tempo de persistência final no banco, mas sim a capacidade do backend de receber requisições, validá-las e enfileirá-las rapidamente no RabbitMQ, retornando resposta imediata ao cliente.

### Ferramenta utilizada

Os testes foram executados com **k6** (falamos dele lá em cima), integrado ao ambiente via Docker.

### Cenário executado

Foi utilizado um cenário com executor `ramping-arrival-rate`, simulando aumento progressivo da taxa de chegada de requisições:

- taxa inicial: `10 req/s`
- duração por estágio:
  - `150 req/s` por `30s`
  - `300 req/s` por `30s`
  - `600 req/s` por `30s`
  - redução para `0 req/s` em `10s`

Configuração adicional:

- `preAllocatedVUs: 20`
- `maxVUs: 300`

### Payload utilizado no teste

Exemplo de payload enviado durante o teste:

```json
{
  "device_id": "12-345",
  "timestamp": "2026-03-23T04:58:48Z",
  "sensor_type": "temperature",
  "reading_type": "analog",
  "value": 25.4
}
```

### Resultados obtidos
Resultado consolidado do cenário principal:

- checks_total: 19547
- checks_succeeded: 100%
- checks_failed: 0%
- http_req_failed: 0.00%
- http_reqs: 19547
- throughput observado: ~194.79 req/s

#### Latência
- média: 3.25 ms
- mediana: 1.23 ms
- mínimo: 284.45 µs
- máximo: 299.74 ms
- p90: 2.83 ms
- p95: 4.38 ms

### Interpretação dos resultados
Os testes mostraram que a API conseguiu responder bem às requisições válidas com status 202 Accepted, indicando que o backend recebeu os dados e os encaminhou para processamento assíncrono com sucesso.

### Observação sobre dropped iterations

Durante o teste, foram registradas 6102 dropped_iterations. Inicialmente eu achei que isso representava erros na API. Mas posteriormente entendi que esse valor indica que o gerador de carga não conseguiu iniciar todas as iterações desejadas com a quantidade de VUs configuradas. 

Por isso, a conclusão correta não é que o sistema sustentou 600 req/s de ponta a ponta, mas sim que, no cenário executado, ele sustentou com sucesso cerca de 195 requisições por segundo com:

0% de falhas HTTP
100% dos checks aprovados
baixa latência na camada de ingestão 

Você pode conferir dois logs de execução do K6 na pasta "/evidencias"

## Uso de IA
Utilizei inteligência artificial em alguns momentos para gerar a primeira versão de arquivos que eu não conhecia. GO é uma linguagem muito nova pra mim, então utilizei IA para entender melhor como algumas bibliotecas funcionavam. Além disso, os arquivos do Docker também foram feitos com auxílio, pois tive dificuldades em gerar o executável pelo Docker. Esse readme tem correção ortográfica de IA, mas foi gerado majoritariamente por mim. 