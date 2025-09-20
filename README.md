# desafio-full-cylcle
desafios realizados durante a pos graduacao go expert da full cycle
# Rate Limiter em Go

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/kauesilva/desafio-full-cylcle)](https://goreportcard.com/report/github.com/kauesilva/desafio-full-cylcle)
[![Test Coverage](https://img.shields.io/badge/coverage-em_breve-lightgrey)](./)
[![Release](https://img.shields.io/github/v/release/kauesilva/desafio-full-cylcle)](https://github.com/kauesilva/desafio-full-cylcle/releases)

Um middleware de rate limiting (limitador de requisições) configurável e de alta performance para aplicações Go, utilizando Redis como backend distribuído. Este projeto foi desenvolvido como parte do desafio da pós-graduação Go Expert da Full Cycle.

## ✨ Funcionalidades

*   **Modos de Limitação Flexíveis**: Suporta limitação por IP, por Token de API, ou uma combinação onde os limites do token têm precedência.
*   **Distribuído e Escalável**: Usa Redis para armazenar os dados de limite, permitindo uma limitação consistente entre múltiplas instâncias da sua aplicação.
*   **Altamente Configurável**: Todas as configurações podem ser controladas por variáveis de ambiente.
    *   Defina políticas padrão para limitação por IP e por token.
    *   Configure requisições por segundo (RPS), duração do bloqueio e janela de tempo.
    *   Sobrescreva as políticas padrão para tokens específicos.
*   **Integração Fácil**: Implementado como um middleware `http.Handler` padrão do Go.
*   **Suporte a Proxy**: Identifica corretamente o IP do cliente por trás de um proxy confiável.

## 🚀 Começando

### Pré-requisitos

*   Go (versão 1.18+ recomendada)
*   Docker & Docker Compose

### Executando Localmente

1.  **Clone o repositório:**
    ```sh
    git clone https://github.com/kauesilva/desafio-full-cylcle.git
    cd desafio-full-cylcle
    ```

2.  **Configure seu ambiente:**
    Crie um arquivo `.env` no diretório raiz. Você pode copiar o exemplo abaixo.

3.  **Execute a aplicação:**
    Use o Docker Compose para construir e executar a aplicação e o serviço Redis.
    ```sh
    docker-compose up --build
    ```

4.  **Acesse o serviço:**
    O rate limiter estará em execução e ouvindo por requisições em `http://localhost:8080`.

## ⚙️ Configuração

A aplicação é configurada usando variáveis de ambiente. Crie um arquivo `.env` na raiz do projeto ou defina as variáveis no seu ambiente de deploy.

### Exemplo de arquivo `.env`

```dotenv
# .env file

# --- Configuração Geral ---
# O modo pode ser: "ip", "token", ou "both"
MODE=both
# Endereço onde o servidor HTTP irá escutar
HTTP_ADDR=:8080
# Confia nos cabeçalhos X-Forwarded-For e similares. Defina como "false" se não estiver atrás de um proxy.
TRUST_PROXY=true
# Nome do cabeçalho para encontrar a API Key/Token
API_KEY_HEADER=API_KEY

# --- Configuração do Redis ---
# O hostname 'rl_redis' vem do nome do serviço no docker-compose.yml
REDIS_ADDR=rl_redis:6379
REDIS_PASSWORD=
REDIS_DB=0

# --- Políticas de Rate Limiting ---
# Janela de tempo global para checagem de limites, em milissegundos
RATE_LIMIT_WINDOWS_MS=1000

# Política padrão para limitação baseada em IP
RATE_LIMIT_IP_RPS=5
RATE_LIMIT_IP_BLOCK=5m

# Política padrão para limitação baseada em Token
RATE_LIMIT_TOKEN_DEFAULT_RPS=10
RATE_LIMIT_TOKEN_DEFAULT_BLOCK=5m

# Array JSON para definir limites específicos para certos tokens
# A duração 'block' usa o formato time.ParseDuration do Go (ex: "1h", "30s", "5m")
RATE_LIMIT_TOKENS_JSON=[{"token":"meu-super-token","rps":100,"block":"1m"}]
```

### Variáveis de Ambiente

| Variável                       | Descrição                                                                                               | Padrão           |
| ------------------------------ | ------------------------------------------------------------------------------------------------------- | ---------------- |
| `MODE`                         | Modo de limitação: `ip`, `token`, ou `both`. No modo `both`, a política de um token válido tem preferência sobre a política de IP. | `both`           |
| `HTTP_ADDR`                    | Endereço do servidor HTTP.                                                                              | `:8080`          |
| `TRUST_PROXY`                  | Se `true`, confia em cabeçalhos como `X-Forwarded-For` para identificar o IP do cliente.                  | `true`           |
| `API_KEY_HEADER`               | Cabeçalho HTTP de onde o token da API será lido.                                                        | `API_KEY`        |
| `REDIS_ADDR`                   | Endereço do servidor Redis.                                                                             | `localhost:6379` |
| `REDIS_PASSWORD`               | Senha do Redis.                                                                                         | `""`             |
| `REDIS_DB`                     | Número do banco de dados do Redis.                                                                      | `0`              |
| `RATE_LIMIT_WINDOWS_MS`        | A janela de tempo para checagem de limites, em milissegundos.                                           | `1000`           |
| `RATE_LIMIT_IP_RPS`            | Requisições Por Segundo (RPS) padrão permitidas para um determinado endereço IP.                        | `5`              |
| `RATE_LIMIT_IP_BLOCK`          | Duração do bloqueio de um IP após exceder seu limite (ex: `5m`, `1h`).                                  | `5m`             |
| `RATE_LIMIT_TOKEN_DEFAULT_RPS` | RPS padrão para requisições com um token de API que não possui uma regra específica.                     | `10`             |
| `RATE_LIMIT_TOKEN_DEFAULT_BLOCK` | Duração de bloqueio padrão para tokens.                                                                 | `5m`             |
| `RATE_LIMIT_TOKENS_JSON`       | Uma string JSON com um array para definir políticas para tokens específicos. Veja o exemplo acima.        | `[]`             |

## 🕹️ Uso

O serviço atua como um middleware. Quando uma requisição chega, ele determina a política de rate limit com base na configuração (`MODE`) e no endereço IP ou no cabeçalho `API_KEY` da requisição.

*   Se a requisição estiver dentro do limite permitido, ela é encaminhada para o serviço subjacente, que atualmente responde com `{"message": "Hello, World!"}` e status `200 OK`.
*   Se o limite for excedido, o middleware responde imediatamente com um erro `429 Too Many Requests` e um corpo JSON:
    ```json
    {
      "message": "you have reached the maximum number of requests or actions allowed within a certain time frame"
    }
    ```

### Exemplo de Requisição

Assumindo `MODE=both`, `RATE_LIMIT_IP_RPS=2`, e `API_KEY_HEADER=API_KEY`.

**1. As duas primeiras requisições de um IP (sem token):**
```sh
curl http://localhost:8080/
# Responde com 200 OK

curl http://localhost:8080/
# Responde com 200 OK
```

**2. Terceira requisição do mesmo IP:**
```sh
curl http://localhost:8080/
# Responde com 429 Too Many Requests
```

**3. Requisição com um token válido:**
Uma requisição com um token usará a política do token, que pode permitir mais requisições.
```sh
curl -H "API_KEY: meu-token-comum" http://localhost:8080/
# Responde com 200 OK (até o limite de RPS do token)
```

## 🧪 Executando Testes

Para executar os testes unitários e de integração, execute o seguinte comando a partir do diretório raiz:

```sh
go test ./...
```

Para gerar um relatório de cobertura de testes:
```sh
go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out
```

## 📄 Licença

Este projeto está licenciado sob a Licença MIT. Veja o arquivo `LICENSE` para mais detalhes.
