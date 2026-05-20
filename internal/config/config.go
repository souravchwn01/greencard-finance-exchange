package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	AppPort                string
	AppEnv                 string
	AppVersion             string
	LogLevel               string
	QuoteExpirySeconds     int
	DefaultMarkupPct       string
	SupportedCountriesFile string

	// Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// PostgreSQL
	PostgresDSN string

	// Supabase storage for evidence upload
	SupabaseURL          string
	SupabaseServiceRoleKey string
	SupabaseStorageBucket string

	// Internal provider ingest auth (either API key or bearer token must be set)
	InternalAPIKey      string
	InternalBearerToken string

	// Rates processing pipeline
	RateStreamName      string
	RateConsumerGroup   string
	RateConsumerName    string
	WorkerPoolSize      int
	ProcessedEventTTL   int // seconds
	StreamClaimMinIdle  int // seconds
	StreamReadBlockTime int // milliseconds
}

func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigFile(".env")
	v.SetConfigType("env")
	v.AutomaticEnv()

	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("QUOTE_EXPIRY_SECONDS", 30)
	v.SetDefault("SUPPORTED_COUNTRIES_FILE", "data/supported_countries.json")

	// Defaults for new architecture
	v.SetDefault("REDIS_ADDR", "localhost:6379")
	v.SetDefault("REDIS_DB", 0)

	v.SetDefault("RATE_STREAM_NAME", "rate_stream")
	v.SetDefault("RATE_CONSUMER_GROUP", "rate_processors")
	v.SetDefault("RATE_CONSUMER_NAME", "exchange-api")
	v.SetDefault("RATE_WORKER_POOL_SIZE", 4)
	v.SetDefault("PROCESSED_EVENT_TTL_SECONDS", 86400)       // 24h
	v.SetDefault("STREAM_CLAIM_MIN_IDLE_SECONDS", 30)        // 30s
	v.SetDefault("STREAM_READ_BLOCK_TIME_MS", 2000)          // 2s
	v.SetDefault("POSTGRES_DSN", "")                         // required for worker
	v.SetDefault("SUPABASE_URL", "")                        // required for quote evidence upload
	v.SetDefault("SUPABASE_SERVICE_ROLE_KEY", "")          // required for quote evidence upload
	v.SetDefault("SUPABASE_STORAGE_BUCKET", "")             // required for quote evidence upload
	v.SetDefault("INTERNAL_API_KEY", "")                     // at least one required
	v.SetDefault("INTERNAL_BEARER_TOKEN", "")                // at least one required

	if err := v.ReadInConfig(); err != nil {
		_, isNotFoundValue := err.(viper.ConfigFileNotFoundError)
		_, isNotFoundPointer := err.(*viper.ConfigFileNotFoundError)
		if !isNotFoundValue && !isNotFoundPointer && !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to read .env file: %w", err)
		}
	}

	required := []string{"APP_PORT", "APP_ENV", "APP_VERSION"}
	for _, key := range required {
		if strings.TrimSpace(v.GetString(key)) == "" {
			return nil, fmt.Errorf("missing required environment variable: %s", key)
		}
	}

	internalAPIKey := strings.TrimSpace(v.GetString("INTERNAL_API_KEY"))
	internalBearer := strings.TrimSpace(v.GetString("INTERNAL_BEARER_TOKEN"))
	if internalAPIKey == "" && internalBearer == "" {
		return nil, fmt.Errorf("missing required auth config: set INTERNAL_API_KEY and/or INTERNAL_BEARER_TOKEN")
	}

	if strings.TrimSpace(v.GetString("POSTGRES_DSN")) == "" {
		return nil, fmt.Errorf("missing required environment variable: POSTGRES_DSN")
	}
	if strings.TrimSpace(v.GetString("SUPABASE_URL")) == "" {
		return nil, fmt.Errorf("missing required environment variable: SUPABASE_URL")
	}
	if strings.TrimSpace(v.GetString("SUPABASE_SERVICE_ROLE_KEY")) == "" {
		return nil, fmt.Errorf("missing required environment variable: SUPABASE_SERVICE_ROLE_KEY")
	}
	if strings.TrimSpace(v.GetString("SUPABASE_STORAGE_BUCKET")) == "" {
		return nil, fmt.Errorf("missing required environment variable: SUPABASE_STORAGE_BUCKET")
	}

	return &Config{
		AppPort:                v.GetString("APP_PORT"),
		AppEnv:                 v.GetString("APP_ENV"),
		AppVersion:             v.GetString("APP_VERSION"),
		LogLevel:               v.GetString("LOG_LEVEL"),
		QuoteExpirySeconds:     v.GetInt("QUOTE_EXPIRY_SECONDS"),
		DefaultMarkupPct:       v.GetString("DEFAULT_MARKUP_PERCENTAGE"),
		SupportedCountriesFile: v.GetString("SUPPORTED_COUNTRIES_FILE"),

		RedisAddr:     v.GetString("REDIS_ADDR"),
		RedisPassword: v.GetString("REDIS_PASSWORD"),
		RedisDB:       v.GetInt("REDIS_DB"),

		PostgresDSN: strings.TrimSpace(v.GetString("POSTGRES_DSN")),
		SupabaseURL:          strings.TrimSpace(v.GetString("SUPABASE_URL")),
		SupabaseServiceRoleKey: strings.TrimSpace(v.GetString("SUPABASE_SERVICE_ROLE_KEY")),
		SupabaseStorageBucket: strings.TrimSpace(v.GetString("SUPABASE_STORAGE_BUCKET")),

		InternalAPIKey:      internalAPIKey,
		InternalBearerToken: internalBearer,

		RateStreamName:      v.GetString("RATE_STREAM_NAME"),
		RateConsumerGroup:   v.GetString("RATE_CONSUMER_GROUP"),
		RateConsumerName:    v.GetString("RATE_CONSUMER_NAME"),
		WorkerPoolSize:      v.GetInt("RATE_WORKER_POOL_SIZE"),
		ProcessedEventTTL:   v.GetInt("PROCESSED_EVENT_TTL_SECONDS"),
		StreamClaimMinIdle:  v.GetInt("STREAM_CLAIM_MIN_IDLE_SECONDS"),
		StreamReadBlockTime: v.GetInt("STREAM_READ_BLOCK_TIME_MS"),
	}, nil
}
