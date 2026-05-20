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
| `SUPABASE_URL`              | Yes      | None                            | Supabase project URL (for evidence uploads).               |
| `SUPABASE_SERVICE_ROLE_KEY` | Yes      | None                            | Service role key for Supabase storage uploads.             |
| `SUPABASE_STORAGE_BUCKET`   | Yes      | None                            | Supabase storage bucket name for evidence files.           |

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

The collection now includes saved-quote examples for the internal quote dashboard (list, create, update, get, delete). See `api-collection/10_quotes.bru` for requests that exercise the new `/api/v1/quotes` endpoints.

## API Documentation

Swagger UI is served at [http://localhost:8080/docs](http://localhost:8080/docs) when the server is running.

The raw OpenAPI 3.0 spec is available at `/docs/openapi.yaml`.

### Saved Quotes (new)

The service exposes a saved-quote CRUD API used by the internal dashboard and frontend evidence workflow:

- `GET /api/v1/quotes` — list recent saved quotes (query params: `limit`, `offset`).
- `GET /api/v1/quotes/:id` — get a saved quote by UUID.
- `POST /api/v1/quotes` — create a saved quote (multipart/form-data, supports `evidence_file` upload).
- `PUT /api/v1/quotes/:id` — update a saved quote (multipart/form-data, supports `evidence_file` upload).
- `DELETE /api/v1/quotes/:id` — delete a saved quote.

Example (create):

```bash
curl -X POST "http://localhost:8080/api/v1/quotes" \
  -F "exchange_from=USD" \
  -F "exchange_to=NGN" \
  -F "sender_country=United States" \
  -F "base_rate=1700" \
  -F "final_rate=1742.5" \
  -F "fee_type=FIXED" \
  -F "quote_expiry_seconds=900" \
  -F "evidence_file=@/path/to/proof.png"
```

## Frontend Wireframe

A sample UI wireframe showing how to display and select from multiple provider rates is available at:

```
docs/wireframe-quotes-dashboard.html
```

Open this file in a browser to see:
- List of recent quotes
- Provider rate comparison cards
- Margin/fee adjustment controls
- Evidence/WhatsApp upload section
- Sample JSON API response

### Multi-Provider Rate Support

When creating or updating a saved quote, you can specify multiple provider rates and select the active one:

```bash
curl -X POST "http://localhost:8080/api/v1/quotes" \
  -F "exchange_from=USD" \
  -F "exchange_to=NGN" \
  -F "sender_country=United States" \
  -F "base_rate=1700.50" \
  -F "final_rate=1742.75" \
  -F "fee_type=FIXED" \
  -F "quote_expiry_seconds=900" \
  -F "selected_provider_id=alpha_finance" \
  -F 'provider_rates=[{"provider_id":"alpha_finance","rate":"1700.50"},{"provider_id":"beta_exchange","rate":"1705.00"},{"provider_id":"gamma_rates","rate":"1698.25"}]' \
  -F "evidence_file=@/path/to/whatsapp-screenshot.png" \
  -F "notes=Provider confirmed rate valid for 30 mins"
```

The response includes all provider options and the currently selected provider:

```json
{
  "provider_rates": [
    {"provider_id": "alpha_finance", "rate": "1700.50"},
    {"provider_id": "beta_exchange", "rate": "1705.00"},
    {"provider_id": "gamma_rates", "rate": "1698.25"}
  ],
  "selected_provider_id": "alpha_finance",
  "base_rate": "1700.50",
  "final_rate": "1742.75"
}
```

Update a quote to switch to a different provider:

```bash
curl -X PUT "http://localhost:8080/api/v1/quotes/{quote_id}" \
  -F "selected_provider_id=gamma_rates" \
  -F "final_rate=1726.53" \
  -F "markup_percentage=1.8"
```

## Make Targets

- `make run`
- `make build`
- `make test`
- `make docker-build`
- `make lint`
