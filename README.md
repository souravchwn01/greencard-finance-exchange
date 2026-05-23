# Exchange Rate API

Exchange-rate quote API built with Go 1.25 and Gin.

## Prerequisites

- Go 1.25+
- Docker and Docker Compose v2

## Running the Full Stack (Recommended)

The entire system — PostgreSQL, Redis, Go API, and the Nginx-served frontend — runs with a single command:

```bash
make up
```

This builds the Go API image, starts all four services with health-check ordering, runs database migrations automatically on first startup, and serves the internal dashboard at **http://localhost:3000**.

To run in the background:

```bash
make up-detach
```

Tail all service logs:

```bash
make logs
```

Stop and remove containers:

```bash
make down
```

Stop and remove containers **plus** the persistent PostgreSQL volume (full reset):

```bash
make down-volumes
```

### How the stack is wired

```
Browser → http://localhost:3000
              ↓
         Nginx (frontend container)
              ├── / → serves frontend/index.html (static)
              ├── /api/ → proxy → Go API :8080
              ├── /internal/ → proxy → Go API :8080
              └── /health → proxy → Go API :8080
                              ↓
                         Go API container
                              ├── PostgreSQL container (quote storage, rate history)
                              └── Redis container (rate cache, stream ingestion)
```

Nginx reverse-proxies all API traffic so the browser makes **no cross-origin requests** — CORS is eliminated entirely.

## Local Development Setup (API only)

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

### Saved Quotes

The service exposes a saved-quote CRUD API used by the internal dashboard and frontend evidence workflow:

- `GET /api/v1/quotes` — list recent saved quotes (query params: `limit` max 100, `offset`).
- `GET /api/v1/quotes/:id` — get a saved quote by UUID. Returns `404` when not found.
- `POST /api/v1/quotes` — create a saved quote (multipart/form-data, supports `evidence_file` upload).
- `PUT /api/v1/quotes/:id` — update a saved quote (multipart/form-data, supports `evidence_file` upload, `provider_rates`, `selected_provider_id`). Returns `404` when not found.
- `DELETE /api/v1/quotes/:id` — delete a saved quote. Returns `404` when not found.

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

## Frontend — Internal Quote Dashboard

The production frontend lives at `frontend/index.html` and is served by the Nginx container at **http://localhost:3000** when the stack is running.

It is a fully self-contained single-page app (no build step, no framework dependencies) with the following features:

| Section | What it does |
|---|---|
| **Provider Rate Grid** | Add provider rows with pair, raw rate, margin %, and notes. Calculates the final rate in real time. |
| **Countdown timers** | Each row tracks expiry. Status badges cycle: `pending → active → expiring (≤10 s) → expired`. |
| **Push to Feed** | Sends the final rate to `POST /internal/v1/rates` (requires API key). Publishes to Redis Stream for async processing. |
| **Evidence upload** | Attach a WhatsApp screenshot per row. Uploaded to Supabase Storage on save; a preview thumbnail is shown inline. |
| **Save Quote** | Builds a multipart `POST /api/v1/quotes` with all fields (rates, provider IDs, expiry, evidence file, notes). |
| **Customer Preview** | Calls `GET /api/v1/exchange-rate` for each active pair and renders the live customer-facing rate card. |
| **Saved Quotes table** | Paginated list via `GET /api/v1/quotes` with delete support. |
| **Settings modal** | Enter and persist the `X-API-Key` and Bearer token in `localStorage` — no server-side session needed. |
| **Health indicator** | Polls `GET /health` every 30 s and shows a live status pill in the header. |

On first load, if no API key is stored the settings modal opens automatically.

### Wireframe reference

The original design wireframe is at `docs/wireframe-quotes-dashboard.html` — open it in a browser to see the layout the implementation was built from.

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

| Target | Description |
|---|---|
| `make run` | Run the Go API locally (no Docker) |
| `make build` | Compile the binary to `bin/server` |
| `make test` | Run all Go tests with verbose output |
| `make lint` | Run `go vet` |
| `make docker-build` | Build the API Docker image only |
| `make up` | Build and start the full stack (postgres + redis + api + frontend) |
| `make up-detach` | Same as `up` but runs in the background |
| `make down` | Stop and remove all containers |
| `make down-volumes` | Stop and remove containers **and** the PostgreSQL data volume |
| `make logs` | Tail logs from all running services |
