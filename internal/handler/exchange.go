package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	validatorv10 "github.com/go-playground/validator/v10"

	apperrors "github.com/gfc-app-finance/greencard-mobile/exchange/internal/errors"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/service"
)

type exchangeSuccessResponse struct {
	Success   bool                   `json:"success"`
	Timestamp string                 `json:"timestamp"`
	Data      *service.QuoteResponse `json:"data"`
}

type exchangeErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type exchangeErrorResponse struct {
	Success   bool              `json:"success"`
	Timestamp string            `json:"timestamp"`
	Error     exchangeErrorBody `json:"error"`
}

type ExchangeHandler struct {
	exchangeService service.ExchangeService
	validate        *validatorv10.Validate
}

func NewExchangeHandler(exchangeService service.ExchangeService, validate *validatorv10.Validate) *ExchangeHandler {
	return &ExchangeHandler{
		exchangeService: exchangeService,
		validate:        validate,
	}
}

func (h *ExchangeHandler) GetExchangeRate(c *gin.Context) {
	var request service.QuoteRequest
	if err := c.ShouldBindQuery(&request); err != nil {
		h.respondWithError(c, apperrors.ErrMissingParameter)
		return
	}

	if err := h.validate.Struct(request); err != nil {
		h.respondWithError(c, mapValidationError(err))
		return
	}

	quote, err := h.exchangeService.GetQuote(c.Request.Context(), request)
	if err != nil {
		var domainErr *apperrors.DomainError
		if errors.As(err, &domainErr) {
			h.respondWithError(c, domainErr)
			return
		}
		h.respondWithError(c, apperrors.ErrInternal)
		return
	}

	c.JSON(http.StatusOK, exchangeSuccessResponse{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      quote,
	})
}

func mapValidationError(err error) *apperrors.DomainError {
	var validationErrors validatorv10.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return apperrors.ErrInternal
	}

	for _, fieldErr := range validationErrors {
		switch fieldErr.Tag() {
		case "iso4217":
			return apperrors.ErrInvalidCurrency
		case "nefield":
			return apperrors.ErrSameCurrency
		case "supported_country":
			return apperrors.ErrInvalidCountry
		case "required":
			return apperrors.ErrMissingParameter
		}
	}

	return apperrors.ErrInternal
}

func (h *ExchangeHandler) respondWithError(c *gin.Context, domainErr *apperrors.DomainError) {
	c.JSON(domainErr.HTTPStatus, exchangeErrorResponse{
		Success:   false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error: exchangeErrorBody{
			Code:    domainErr.Code,
			Message: domainErr.Message,
		},
	})
}
