package rates

import "time"

// RateEvent is the provider-driven payload received by the internal ingest API.
type RateEvent struct {
	EventID    string    `json:"event_id"`
	ProviderID string    `json:"provider_id"`
	Pair       string    `json:"pair"`
	Rate       float64   `json:"rate"`
	Timestamp  time.Time `json:"timestamp"`
}

// CachedRate is stored in Redis (latest per pair).
type CachedRate struct {
	Pair       string    `json:"pair"`
	Rate       float64   `json:"rate"`
	ProviderID string    `json:"provider_id"`
	UpdatedAt  time.Time `json:"updated_at"`
}

