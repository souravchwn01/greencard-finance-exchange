package transactions

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Service struct {
	store *Store
}

func NewService(store *Store) *Service {
	return &Service{store: store}
}

var ErrQuoteNotFound = fmt.Errorf("quote transaction not found")

func (s *Service) Create(ctx context.Context, req CreateQuoteTransactionRequest) (*QuoteTransaction, error) {
	if req.ExchangeFrom == "" || req.ExchangeTo == "" || req.SenderCountry == "" {
		return nil, fmt.Errorf("missing required quote transaction information")
	}
	if req.BaseRate.LessThan(decimal.Zero) {
		return nil, fmt.Errorf("base_rate must be zero or positive")
	}
	if req.FinalRate.LessThan(decimal.Zero) {
		return nil, fmt.Errorf("final_rate must be zero or positive")
	}
	if req.QuoteExpirySeconds <= 0 {
		return nil, fmt.Errorf("quote_expiry_seconds must be greater than zero")
	}
	if req.GeneratedAt.IsZero() {
		req.GeneratedAt = time.Now().UTC()
	}
	if req.QuoteID == uuid.Nil {
		req.QuoteID = uuid.New()
	}

	quote := QuoteTransaction{
		ID:                 req.QuoteID,
		ProviderID:         req.ProviderID,
		ProviderRates:      req.ProviderRates,
		SelectedProviderID: req.SelectedProviderID,
		ExchangeFrom:       req.ExchangeFrom,
		ExchangeTo:         req.ExchangeTo,
		SenderCountry:      req.SenderCountry,
		BaseRate:           req.BaseRate,
		MarkupPercentage:   req.MarkupPercentage,
		FinalRate:          req.FinalRate,
		FeeType:            req.FeeType,
		TransactionFee:     req.TransactionFee,
		QuoteExpirySeconds: req.QuoteExpirySeconds,
		GeneratedAt:        req.GeneratedAt,
		QuoteExpiresAt:     req.GeneratedAt.Add(time.Duration(req.QuoteExpirySeconds) * time.Second),
		EvidenceURL:        req.EvidenceURL,
		Notes:              req.Notes,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := s.store.CreateQuoteTransaction(ctx, quote); err != nil {
		return nil, err
	}
	return &quote, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*QuoteTransaction, error) {
	qt, err := s.store.GetQuoteTransactionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return qt, nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]QuoteTransaction, error) {
	return s.store.ListQuoteTransactions(ctx, limit, offset)
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateQuoteTransactionRequest) (*QuoteTransaction, error) {
	quote, err := s.store.GetQuoteTransactionByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.BaseRate != nil {
		quote.BaseRate = *req.BaseRate
	}
	if req.MarkupPercentage != nil {
		quote.MarkupPercentage = *req.MarkupPercentage
	}
	if req.FinalRate != nil {
		quote.FinalRate = *req.FinalRate
	}
	if req.FeeType != nil {
		quote.FeeType = *req.FeeType
	}
	if req.TransactionFee != nil {
		quote.TransactionFee = *req.TransactionFee
	}
	if req.QuoteExpirySeconds != nil {
		quote.QuoteExpirySeconds = *req.QuoteExpirySeconds
	}
	if req.GeneratedAt != nil {
		quote.GeneratedAt = *req.GeneratedAt
	}
	if req.EvidenceURL != nil {
		quote.EvidenceURL = *req.EvidenceURL
	}
	if req.Notes != nil {
		quote.Notes = *req.Notes
	}
	if req.ProviderRates != nil {
		quote.ProviderRates = *req.ProviderRates
	}
	if req.SelectedProviderID != nil {
		quote.SelectedProviderID = *req.SelectedProviderID
	}

	quote.QuoteExpiresAt = quote.GeneratedAt.Add(time.Duration(quote.QuoteExpirySeconds) * time.Second)
	quote.UpdatedAt = time.Now().UTC()

	if err := s.store.UpdateQuoteTransaction(ctx, *quote); err != nil {
		return nil, err
	}

	return quote, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.DeleteQuoteTransaction(ctx, id)
}
