package rates

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Reader is the minimal interface used by the public quote service.
type Reader interface {
	GetLatest(ctx context.Context, pair string) (*CachedRate, error)
	GetAllRates(ctx context.Context) ([]CachedRate, error)
	UpdateRate(ctx context.Context, pair string, rate float64, providerID string) (*CachedRate, error)
}

// Service is the centralized read-path service used by public APIs.
type Service struct {
	cache *Cache
	store *Store
}

func NewService(cache *Cache, store *Store) *Service {
	return &Service{cache: cache, store: store}
}

func (s *Service) GetLatest(ctx context.Context, pair string) (*CachedRate, error) {
	if err := ValidatePair(pair); err != nil {
		return nil, err
	}
	r, err := s.cache.GetLatest(ctx, pair)
	if err != nil {
		return nil, err
	}
	if r.Rate <= 0 {
		return nil, fmt.Errorf("invalid cached rate for %s", NormalizePair(pair))
	}
	return r, nil
}

func (s *Service) GetAllRates(ctx context.Context) ([]CachedRate, error) {
	return s.cache.GetAllLatest(ctx)
}

func (s *Service) UpdateRate(ctx context.Context, pair string, rate float64, providerID string) (*CachedRate, error) {
	pair = NormalizePair(pair)
	if err := ValidatePair(pair); err != nil {
		return nil, err
	}
	if rate <= 0 {
		return nil, fmt.Errorf("rate must be positive")
	}
	if providerID == "" {
		providerID = "admin-manual"
	}

	now := time.Now().UTC()
	eventID := uuid.New().String()

	ev := RateEvent{
		EventID:    eventID,
		ProviderID: providerID,
		Pair:       pair,
		Rate:       rate,
		Timestamp:  now,
	}
	if err := s.store.UpsertRate(ctx, ev); err != nil {
		return nil, err
	}

	cached := CachedRate{
		Pair:       pair,
		Rate:       rate,
		ProviderID: providerID,
		UpdatedAt:  now,
	}
	if err := s.cache.SetLatest(ctx, cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

