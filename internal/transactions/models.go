package transactions

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type QuoteTransaction struct {
	ID                 uuid.UUID       `json:"id"`
	ProviderID         string          `json:"provider_id"`
	ProviderRates      []ProviderRate  `json:"provider_rates,omitempty"`
	SelectedProviderID string          `json:"selected_provider_id,omitempty"`
	ExchangeFrom       string          `json:"exchange_from"`
	ExchangeTo         string          `json:"exchange_to"`
	SenderCountry      string          `json:"sender_country"`
	BaseRate           decimal.Decimal `json:"base_rate"`
	MarkupPercentage   decimal.Decimal `json:"markup_percentage"`
	FinalRate          decimal.Decimal `json:"final_rate"`
	FeeType            string          `json:"fee_type"`
	TransactionFee     decimal.Decimal `json:"transaction_fee"`
	QuoteExpirySeconds int             `json:"quote_expiry_seconds"`
	GeneratedAt        time.Time       `json:"generated_at"`
	QuoteExpiresAt     time.Time       `json:"quote_expires_at"`
	EvidenceURL        string          `json:"evidence_url,omitempty"`
	Notes              string          `json:"notes,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

type CreateQuoteTransactionRequest struct {
	QuoteID            uuid.UUID
	ProviderID         string
	ProviderRates      []ProviderRate
	SelectedProviderID string
	ExchangeFrom       string
	ExchangeTo         string
	SenderCountry      string
	BaseRate           decimal.Decimal
	MarkupPercentage   decimal.Decimal
	FinalRate          decimal.Decimal
	FeeType            string
	TransactionFee     decimal.Decimal
	QuoteExpirySeconds int
	GeneratedAt        time.Time
	EvidenceURL        string
	Notes              string
}

type UpdateQuoteTransactionRequest struct {
	BaseRate           *decimal.Decimal
	MarkupPercentage   *decimal.Decimal
	FinalRate          *decimal.Decimal
	FeeType            *string
	TransactionFee     *decimal.Decimal
	QuoteExpirySeconds *int
	GeneratedAt        *time.Time
	EvidenceURL        *string
	Notes              *string
	ProviderRates      *[]ProviderRate
	SelectedProviderID *string
}

type ProviderRate struct {
	ProviderID string          `json:"provider_id"`
	Rate       decimal.Decimal `json:"rate"`
}
