// Package service tests exchange quote calculations and error behavior.
package service

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/gfc-app-finance/greencard-mobile/exchange/internal/errors"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/rates"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/domain"
	"github.com/shopspring/decimal"
)

type stubRateReader struct {
	rates map[string]rates.CachedRate
}

func (s *stubRateReader) GetLatest(_ context.Context, pair string) (*rates.CachedRate, error) {
	pair = rates.NormalizePair(pair)
	r, ok := s.rates[pair]
	if !ok {
		return nil, rates.ErrRateNotFound
	}
	out := r
	return &out, nil
}

func (s *stubRateReader) GetAllRates(_ context.Context) ([]rates.CachedRate, error) {
	var result []rates.CachedRate
	for _, r := range s.rates {
		result = append(result, r)
	}
	return result, nil
}

func (s *stubRateReader) UpdateRate(_ context.Context, pair string, rate float64, providerID string) (*rates.CachedRate, error) {
	pair = rates.NormalizePair(pair)
	cr := rates.CachedRate{Pair: pair, Rate: rate, ProviderID: providerID}
	s.rates[pair] = cr
	return &cr, nil
}

func TestExchangeServiceGetQuote(t *testing.T) {
	t.Parallel()

	reader := &stubRateReader{
		rates: map[string]rates.CachedRate{
			"NGN_CAD": {Pair: "NGN_CAD", Rate: 0.00089, ProviderID: "p1"},
			"USD_NGN": {Pair: "USD_NGN", Rate: 1580.00, ProviderID: "p1"},
		},
	}

	svc, err := NewExchangeService(reader, 30)
	if err != nil {
		t.Fatalf("NewExchangeService() returned unexpected error: %v", err)
	}

	tests := []struct {
		name              string
		from              string
		to                string
		expectedRate      decimal.Decimal
		expectErr         error
	}{
		{
			name:         "NGN to CAD uses cached provider rate",
			from:              "NGN",
			to:                "CAD",
			expectedRate:      decimal.RequireFromString("0.00089"),
		},
		{
			name:         "USD to NGN uses cached provider rate",
			from:              "USD",
			to:                "NGN",
			expectedRate:      decimal.RequireFromString("1580.00"),
		},
		{
			name:      "Unsupported pair returns error",
			from:      "NGN",
			to:        "GBP",
			expectErr: apperrors.ErrUnsupportedPair,
		},
	}

	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			quote, quoteErr := svc.GetQuote(context.Background(), domain.QuoteRequest{
				From:          testCase.from,
				To:            testCase.to,
				SenderCountry: "Nigeria",
			})

			if testCase.expectErr != nil {
				if !errors.Is(quoteErr, testCase.expectErr) {
					t.Fatalf("expected error %v, got %v", testCase.expectErr, quoteErr)
				}
				return
			}

			if quoteErr != nil {
				t.Fatalf("GetQuote() returned unexpected error: %v", quoteErr)
			}

			if !quote.FinalRate.Equal(testCase.expectedRate) {
				t.Fatalf("unexpected final rate: got %s, want %s", quote.FinalRate.String(), testCase.expectedRate.String())
			}
			if !quote.BaseRate.Equal(testCase.expectedRate) {
				t.Fatalf("unexpected base rate: got %s, want %s", quote.BaseRate.String(), testCase.expectedRate.String())
			}
			if !quote.MarkupPercentage.Equal(decimal.Zero) {
				t.Fatalf("expected zero markup, got %s", quote.MarkupPercentage.String())
			}
			if quote.FeeType != "NONE" {
				t.Fatalf("expected fee_type NONE, got %s", quote.FeeType)
			}
			if !quote.TransactionFee.Equal(decimal.Zero) {
				t.Fatalf("expected zero fee, got %s", quote.TransactionFee.String())
			}
		})
	}
}
