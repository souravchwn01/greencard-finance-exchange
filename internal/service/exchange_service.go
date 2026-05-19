package service

import (
	"context"
	"time"

	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/domain"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/errors"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/rates"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type QuoteRequest = domain.QuoteRequest

type QuoteResponse = domain.QuoteResponse

type ExchangeService interface {
	GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error)
}

type exchangeService struct {
	quoteExpiry      int
	rateService      rates.Reader
}

func NewExchangeService(rateService rates.Reader, quoteExpirySeconds int) (ExchangeService, error) {
	return &exchangeService{
		quoteExpiry:      quoteExpirySeconds,
		rateService:      rateService,
	}, nil
}

func (s *exchangeService) GetQuote(ctx context.Context, req QuoteRequest) (*QuoteResponse, error) {
	pair := rates.NormalizePair(req.From) + "_" + rates.NormalizePair(req.To)
	latest, err := s.rateService.GetLatest(ctx, pair)
	if err != nil {
		if err == rates.ErrRateNotFound {
			return nil, errors.ErrUnsupportedPair
		}
		return nil, err
	}

	r := decimal.NewFromFloat(latest.Rate)

	return &QuoteResponse{
		ID:                 uuid.New(),
		ExchangeFrom:       req.From,
		ExchangeTo:         req.To,
		SenderCountry:      req.SenderCountry,
		BaseRate:           r,
		MarkupPercentage:   decimal.Zero,
		FinalRate:          r,
		FeeType:            "NONE",
		TransactionFee:     decimal.Zero,
		QuoteExpirySeconds: s.quoteExpiry,
		GeneratedAt:        time.Now().UTC(),
	}, nil
}
