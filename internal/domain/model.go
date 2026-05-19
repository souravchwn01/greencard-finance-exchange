package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type QuoteRequest struct {
	From          string `form:"from" validate:"required,iso4217"`
	To            string `form:"to" validate:"required,iso4217,nefield=From"`
	SenderCountry string `form:"sender_country" validate:"required,supported_country"`
}

type QuoteResponse struct {
	ID                 uuid.UUID       `json:"id"`
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
}
