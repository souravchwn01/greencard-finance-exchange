# Exchange Rate API

Simple exchange-rate quote API built with Go 1.22 and Gin.

## Prerequisites

- Go 1.22+
- Docker

## Local Setup

1. Copy environment file:

```bash
cp .env.example .env
```

2. Install dependencies:

```bash
go mod tidy
```

3. Run the server:

```bash
make run
```

Server starts on `APP_PORT`.

## Architecture (Dynamic Rates)

Provider rate events are ingested via an internal API and processed asynchronously:

`POST /internal/v1/rates` → Redis Stream (`rate_stream`) → worker pool → PostgreSQL history + Redis cache (`rate:{pair}`).

Public quote requests read **only** from Redis cache (no hardcoded rates, no live PostgreSQL queries).

## Environment Variables

| Name                        | Required | Default                         | Description                                                |
| --------------------------- | -------- | ------------------------------- | ---------------------------------------------------------- |
| `APP_PORT`                  | Yes      | None                            | HTTP server port (example: `8080`).                        |
| `APP_ENV`                   | Yes      | None                            | Runtime environment (`development` or `production`).       |
| `APP_VERSION`               | Yes      | None                            | Version returned by `/health`.                             |
| `LOG_LEVEL`                 | No       | `info`                          | Logger level (`debug`, `info`, `warn`, `error`).           |
| `QUOTE_EXPIRY_SECONDS`      | No       | `30`                            | Quote expiry value in response payload.                    |
| `DEFAULT_MARKUP_PERCENTAGE` | No       | None                            | Deprecated (provider sends final rate).                    |
| `SUPPORTED_COUNTRIES_FILE`  | No       | `data/supported_countries.json` | Path to supported countries JSON file loaded at startup.   |
| `REDIS_ADDR`                | No       | `localhost:6379`                | Redis address.                                             |
| `REDIS_PASSWORD`            | No       | None                            | Redis password.                                            |
| `REDIS_DB`                  | No       | `0`                             | Redis DB index.                                            |
| `POSTGRES_DSN`              | Yes      | None                            | PostgreSQL connection string for rate history persistence. |
| `INTERNAL_API_KEY`          | Yes\*    | None                            | Internal provider API key (set this or bearer token).      |
| `INTERNAL_BEARER_TOKEN`     | Yes\*    | None                            | Internal provider bearer token (set this or API key).      |

## Supported Countries

`data/supported_countries.json` is the source of truth for `sender_country` validation.
Adding or removing a country takes effect after server restart, with no code changes.

If the configured `SUPPORTED_COUNTRIES_FILE` path is missing or malformed JSON, startup fails with exit code `1` and a descriptive log message.

## Endpoints

### Health Check

```bash
curl "http://localhost:8080/health"
```

### Success: NGN to CAD

```bash
curl "http://localhost:8080/api/v1/exchange-rate?from=NGN&to=CAD&sender_country=Nigeria"
```

### Internal: Ingest Provider Rate (Async)

```bash
curl -X POST "http://localhost:8080/internal/v1/rates" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: YOUR_KEY" \
  -d '{"event_id":"67af452c-caf7-4af4-957b-a2881b173471","provider_id":"provider_1","pair":"USD_BDT","rate":119.45,"timestamp":"2026-05-19T10:30:00Z"}'
```

### Error: Missing Parameter

```bash
curl "http://localhost:8080/api/v1/exchange-rate?from=NGN&to=CAD"
```

### Error: Invalid Currency

```bash
curl "http://localhost:8080/api/v1/exchange-rate?from=NGN&to=XYZ&sender_country=Nigeria"
```

### Error: Same Currency

```bash
curl "http://localhost:8080/api/v1/exchange-rate?from=NGN&to=NGN&sender_country=Nigeria"
```

### Error: Unsupported Pair

```bash
curl "http://localhost:8080/api/v1/exchange-rate?from=NGN&to=GBP&sender_country=Nigeria"
```

### Error: Invalid Country

```bash
curl "http://localhost:8080/api/v1/exchange-rate?from=NGN&to=CAD&sender_country=Germany"
```

## API Collection

A complete [Bruno](https://www.usebruno.com/) API collection is available in the `api-collection` directory. This includes pre-configured requests for health checks, valid conversions, and error scenarios.

To use it:

1. Download Bruno.
2. Click **Open Collection**.
3. Select the `api-collection` directory in this project.

## API Documentation

Swagger UI is served at [http://localhost:8080/docs](http://localhost:8080/docs) when the server is running.

The raw OpenAPI 3.0 spec is available at `/docs/openapi.yaml`.

## Make Targets

- `make run`
- `make build`
- `make test`
- `make docker-build`
- `make lint`
