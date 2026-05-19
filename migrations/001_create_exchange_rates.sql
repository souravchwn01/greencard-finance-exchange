CREATE TABLE IF NOT EXISTS exchange_rates (
  id BIGSERIAL PRIMARY KEY,
  event_id TEXT UNIQUE NOT NULL,
  provider_id TEXT NOT NULL,
  pair TEXT NOT NULL,
  rate NUMERIC(12, 4) NOT NULL,
  created_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_exchange_rates_pair_created_at
  ON exchange_rates (pair, created_at DESC);

