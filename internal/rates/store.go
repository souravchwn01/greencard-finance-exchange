package rates

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) InsertEvent(ctx context.Context, ev RateEvent) error {
	// Idempotent at DB level as well.
	_, err := s.pool.Exec(ctx, `
		INSERT INTO exchange_rates (event_id, provider_id, pair, rate, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO NOTHING
	`, ev.EventID, ev.ProviderID, NormalizePair(ev.Pair), ev.Rate, ev.Timestamp.UTC())
	if err != nil {
		return fmt.Errorf("insert exchange rate: %w", err)
	}
	return nil
}

func (s *Store) UpsertRate(ctx context.Context, ev RateEvent) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO exchange_rates (event_id, provider_id, pair, rate, created_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (event_id) DO UPDATE
		SET rate = EXCLUDED.rate,
		    provider_id = EXCLUDED.provider_id,
		    created_at = EXCLUDED.created_at
	`, ev.EventID, ev.ProviderID, NormalizePair(ev.Pair), ev.Rate, ev.Timestamp.UTC())
	if err != nil {
		return fmt.Errorf("upsert exchange rate: %w", err)
	}
	return nil
}

