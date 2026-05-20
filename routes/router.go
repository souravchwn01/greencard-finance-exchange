package routes

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/handler"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/middleware"
)

func New(
	logger *zap.Logger,
	exchangeHandler *handler.ExchangeHandler,
	healthHandler *handler.HealthHandler,
	internalRatesHandler *handler.InternalRatesHandler,
	countriesHandler *handler.CountriesHandler,
	ratesHandler *handler.RatesHandler,
	quotesHandler *handler.QuotesHandler,
	internalAPIKey string,
	internalBearerToken string,
) *gin.Engine {
	router := gin.New()

	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.Recovery(logger))

	router.GET("/health", healthHandler.GetHealth)
	router.GET("/api/v1/exchange-rate", exchangeHandler.GetExchangeRate)
	router.GET("/api/v1/supported-countries", countriesHandler.GetSupportedCountries)
	router.GET("/api/v1/rates", ratesHandler.GetAllRates)
	router.GET("/api/v1/quotes", quotesHandler.ListQuotes)
	router.GET("/api/v1/quotes/:id", quotesHandler.GetQuote)
	router.POST("/api/v1/quotes", quotesHandler.CreateQuote)
	router.PUT("/api/v1/quotes/:id", quotesHandler.UpdateQuote)
	router.DELETE("/api/v1/quotes/:id", quotesHandler.DeleteQuote)

	internal := router.Group("/internal/v1")
	internal.Use(middleware.InternalAuth(internalAPIKey, internalBearerToken))
	internal.POST("/rates", internalRatesHandler.PostRate)
	internal.PUT("/rates/:pair", ratesHandler.UpdateRate)

	handler.RegisterDocsRoutes(router)

	return router
}
