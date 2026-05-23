package transactions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) CreateQuoteTransaction(ctx context.Context, qt QuoteTransaction) error {
	var providerRatesJSON []byte
	var err error
	if len(qt.ProviderRates) > 0 {
		providerRatesJSON, err = json.Marshal(qt.ProviderRates)
		if err != nil {
			return fmt.Errorf("marshal provider rates: %w", err)
		}
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO quote_transactions (
			id, provider_id, provider_rates, selected_provider_id,
			exchange_from, exchange_to, sender_country,
			base_rate, markup_percentage, final_rate,
			fee_type, transaction_fee,
			quote_expiry_seconds, generated_at, quote_expires_at,
			evidence_url, notes, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
	`, qt.ID, qt.ProviderID, providerRatesJSON, qt.SelectedProviderID,
		qt.ExchangeFrom, qt.ExchangeTo, qt.SenderCountry,
		qt.BaseRate, qt.MarkupPercentage, qt.FinalRate,
		qt.FeeType, qt.TransactionFee,
		qt.QuoteExpirySeconds, qt.GeneratedAt, qt.QuoteExpiresAt,
		qt.EvidenceURL, qt.Notes, qt.CreatedAt, qt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert quote transaction: %w", err)
	}
	return nil
}

func (s *Store) GetQuoteTransactionByID(ctx context.Context, id uuid.UUID) (*QuoteTransaction, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, provider_id, exchange_from, exchange_to, sender_country,
			base_rate, markup_percentage, final_rate, fee_type, transaction_fee,
			quote_expiry_seconds, generated_at, quote_expires_at, evidence_url,
			provider_rates, selected_provider_id, notes, created_at, updated_at
		FROM quote_transactions
		WHERE id = $1
	`, id)

	var qt QuoteTransaction
	var providerRatesRaw []byte
	if err := row.Scan(
		&qt.ID, &qt.ProviderID, &qt.ExchangeFrom, &qt.ExchangeTo, &qt.SenderCountry,
		&qt.BaseRate, &qt.MarkupPercentage, &qt.FinalRate, &qt.FeeType, &qt.TransactionFee,
		&qt.QuoteExpirySeconds, &qt.GeneratedAt, &qt.QuoteExpiresAt, &qt.EvidenceURL,
		&providerRatesRaw, &qt.SelectedProviderID, &qt.Notes,
		&qt.CreatedAt, &qt.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrQuoteNotFound
		}
		return nil, fmt.Errorf("query quote transaction: %w", err)
	}
	if len(providerRatesRaw) > 0 {
		if err := json.Unmarshal(providerRatesRaw, &qt.ProviderRates); err != nil {
			return nil, fmt.Errorf("unmarshal provider rates: %w", err)
		}
	}
	return &qt, nil
}

func (s *Store) ListQuoteTransactions(ctx context.Context, limit, offset int) ([]QuoteTransaction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, provider_id, exchange_from, exchange_to, sender_country,
			base_rate, markup_percentage, final_rate, fee_type, transaction_fee,
			quote_expiry_seconds, generated_at, quote_expires_at, evidence_url,
			provider_rates, selected_provider_id, notes, created_at, updated_at
		FROM quote_transactions
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list quote transactions: %w", err)
	}
	defer rows.Close()

	var results []QuoteTransaction
	for rows.Next() {
		var qt QuoteTransaction
		var providerRatesRaw []byte
		if err := rows.Scan(
			&qt.ID, &qt.ProviderID, &qt.ExchangeFrom, &qt.ExchangeTo, &qt.SenderCountry,
			&qt.BaseRate, &qt.MarkupPercentage, &qt.FinalRate, &qt.FeeType, &qt.TransactionFee,
			&qt.QuoteExpirySeconds, &qt.GeneratedAt, &qt.QuoteExpiresAt, &qt.EvidenceURL,
			&providerRatesRaw, &qt.SelectedProviderID, &qt.Notes,
			&qt.CreatedAt, &qt.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan quote transaction: %w", err)
		}
		if len(providerRatesRaw) > 0 {
			if err := json.Unmarshal(providerRatesRaw, &qt.ProviderRates); err != nil {
				return nil, fmt.Errorf("unmarshal provider rates: %w", err)
			}
		}
		results = append(results, qt)
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("iterate quote transactions: %w", rows.Err())
	}
	return results, nil
}

func (s *Store) UpdateQuoteTransaction(ctx context.Context, qt QuoteTransaction) error {
	var providerRatesJSON []byte
	var err error
	if len(qt.ProviderRates) > 0 {
		providerRatesJSON, err = json.Marshal(qt.ProviderRates)
		if err != nil {
			return fmt.Errorf("marshal provider rates: %w", err)
		}
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE quote_transactions
		SET provider_id = $2,
			provider_rates = $3,
			selected_provider_id = $4,
			exchange_from = $5,
			exchange_to = $6,
			sender_country = $7,
			base_rate = $8,
			markup_percentage = $9,
			final_rate = $10,
			fee_type = $11,
			transaction_fee = $12,
			quote_expiry_seconds = $13,
			generated_at = $14,
			quote_expires_at = $15,
			evidence_url = $16,
			notes = $17,
			updated_at = $18
		WHERE id = $1
	`, qt.ID, qt.ProviderID, providerRatesJSON, qt.SelectedProviderID,
		qt.ExchangeFrom, qt.ExchangeTo, qt.SenderCountry,
		qt.BaseRate, qt.MarkupPercentage, qt.FinalRate,
		qt.FeeType, qt.TransactionFee, qt.QuoteExpirySeconds, qt.GeneratedAt,
		qt.QuoteExpiresAt, qt.EvidenceURL, qt.Notes, qt.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update quote transaction: %w", err)
	}
	return nil
}

func (s *Store) DeleteQuoteTransaction(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM quote_transactions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete quote transaction: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrQuoteNotFound
	}
	return nil
}
