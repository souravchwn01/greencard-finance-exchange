CREATE TABLE IF NOT EXISTS quote_transactions (
  id UUID PRIMARY KEY,
  provider_id TEXT,
  exchange_from TEXT NOT NULL,
  exchange_to TEXT NOT NULL,
  sender_country TEXT NOT NULL,
  base_rate NUMERIC(24,12) NOT NULL,
  markup_percentage NUMERIC(12,6) NOT NULL DEFAULT 0,
  final_rate NUMERIC(24,12) NOT NULL,
  fee_type TEXT NOT NULL,
  transaction_fee NUMERIC(24,12) NOT NULL DEFAULT 0,
  quote_expiry_seconds INT NOT NULL,
  generated_at TIMESTAMPTZ NOT NULL,
  quote_expires_at TIMESTAMPTZ NOT NULL,
  evidence_url TEXT,
  provider_rates JSONB,
  selected_provider_id TEXT,
  notes TEXT,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_quote_transactions_created_at
  ON quote_transactions (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_quote_transactions_exchange_pair
  ON quote_transactions (exchange_from, exchange_to);
