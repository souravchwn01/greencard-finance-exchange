package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/config"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/handler"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/middleware"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/rates"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/service"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/supabase"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/transactions"
	appvalidator "github.com/gfc-app-finance/greencard-mobile/exchange/internal/validator"
	"github.com/gfc-app-finance/greencard-mobile/exchange/routes"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	logger, err := middleware.NewZapLogger(cfg.AppEnv, cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = logger.Sync()
	}()

	validate, err := appvalidator.New(cfg.SupportedCountriesFile)
	if err != nil {
		logger.Fatal("failed to load supported countries", zap.Error(err))
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if pingErr := redisClient.Ping(context.Background()).Err(); pingErr != nil {
		logger.Fatal("failed to connect to redis", zap.Error(pingErr))
	}
	defer func() { _ = redisClient.Close() }()

	pgPool, err := pgxpool.New(context.Background(), cfg.PostgresDSN)
	if err != nil {
		logger.Fatal("failed to init postgres pool", zap.Error(err))
	}
	defer pgPool.Close()

	stream := rates.NewStream(redisClient, cfg.RateStreamName, cfg.RateConsumerGroup)
	if err := stream.EnsureConsumerGroup(context.Background()); err != nil {
		logger.Fatal("failed to ensure redis stream consumer group", zap.Error(err))
	}

	cache := rates.NewCache(redisClient)
	store := rates.NewStore(pgPool)
	rateService := rates.NewService(cache, store)

	exchangeService, err := service.NewExchangeService(rateService, cfg.QuoteExpirySeconds)
	if err != nil {
		logger.Fatal("failed to initialize exchange service", zap.Error(err))
	}

	exchangeHandler := handler.NewExchangeHandler(exchangeService, validate)
	healthHandler := handler.NewHealthHandler(cfg.AppVersion)
	internalRatesHandler := handler.NewInternalRatesHandler(stream)
	countriesHandler := handler.NewCountriesHandler(cfg.SupportedCountriesFile)
	ratesHandler := handler.NewRatesHandler(rateService)

	transactionStore := transactions.NewStore(pgPool)
	transactionService := transactions.NewService(transactionStore)
	supabaseStorage := supabase.NewStorage(cfg.SupabaseURL, cfg.SupabaseStorageBucket, cfg.SupabaseServiceRoleKey)
	quotesHandler := handler.NewQuotesHandler(transactionService, supabaseStorage, validate)

	router := routes.New(
		logger,
		exchangeHandler,
		healthHandler,
		internalRatesHandler,
		countriesHandler,
		ratesHandler,
		quotesHandler,
		cfg.InternalAPIKey,
		cfg.InternalBearerToken,
	)

	address := cfg.AppPort
	if !strings.HasPrefix(address, ":") {
		address = ":" + address
	}

	server := &http.Server{
		Addr:    address,
		Handler: router,
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start async rate processor
	processor := rates.NewProcessor(logger, stream, store, cache, redisClient, rates.WorkerConfig{
		ConsumerName:      cfg.RateConsumerName,
		PoolSize:          cfg.WorkerPoolSize,
		ReadCount:         20,
		ReadBlock:         time.Duration(cfg.StreamReadBlockTime) * time.Millisecond,
		ClaimMinIdle:      time.Duration(cfg.StreamClaimMinIdle) * time.Second,
		ProcessedEventTTL: time.Duration(cfg.ProcessedEventTTL) * time.Second,
	})

	processorErr := make(chan error, 1)
	go func() {
		processorErr <- processor.Run(rootCtx)
	}()

	serverErr := make(chan error, 1)
	go func() {
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			serverErr <- serveErr
			return
		}
		serverErr <- nil
	}()

	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-signalChannel:
		logger.Info("shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			logger.Error("http server failed", zap.Error(err))
		}
	case err := <-processorErr:
		if err != nil {
			logger.Error("rate processor failed", zap.Error(err))
		}
	}

	// Cancel workers then shutdown HTTP server.
	cancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if shutdownErr := server.Shutdown(shutdownCtx); shutdownErr != nil {
		logger.Error("server shutdown failed", zap.Error(shutdownErr))
		os.Exit(1)
	}

	// Wait for processor to exit.
	select {
	case err := <-processorErr:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("rate processor stopped with error", zap.Error(err))
		}
	case <-time.After(10 * time.Second):
		logger.Warn("rate processor shutdown timed out")
	}

	logger.Info("server shut down cleanly")
}
