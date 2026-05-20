package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	validatorv10 "github.com/go-playground/validator/v10"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/supabase"
	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/transactions"
)

type quotesListResponse struct {
	Success   bool                       `json:"success"`
	Timestamp string                     `json:"timestamp"`
	Data      []transactions.QuoteTransaction `json:"data"`
	Meta      struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}

type quoteDetailResponse struct {
	Success   bool                     `json:"success"`
	Timestamp string                   `json:"timestamp"`
	Data      *transactions.QuoteTransaction `json:"data"`
}

type quotesErrorResponse struct {
	Success   bool              `json:"success"`
	Timestamp string            `json:"timestamp"`
	Error     map[string]string `json:"error"`
}

type QuotesHandler struct {
	transactionService *transactions.Service
	storage            *supabase.Storage
	validate           *validatorv10.Validate
}

func NewQuotesHandler(transactionService *transactions.Service, storage *supabase.Storage, validate *validatorv10.Validate) *QuotesHandler {
	return &QuotesHandler{
		transactionService: transactionService,
		storage:            storage,
		validate:           validate,
	}
}

func (h *QuotesHandler) ListQuotes(c *gin.Context) {
	limit := 20
	offset := 0
	if param := c.Query("limit"); param != "" {
		if parsed, err := strconv.Atoi(param); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	if param := c.Query("offset"); param != "" {
		if parsed, err := strconv.Atoi(param); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	all, err := h.transactionService.List(c.Request.Context(), limit, offset)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to fetch saved quotes.")
		return
	}

	response := quotesListResponse{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      all,
	}
	response.Meta.Limit = limit
	response.Meta.Offset = offset

	c.JSON(http.StatusOK, response)
}

func (h *QuotesHandler) GetQuote(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_QUOTE_ID", "Quote id is invalid.")
		return
	}

	quote, err := h.transactionService.Get(c.Request.Context(), id)
	if err != nil {
		h.respondWithError(c, http.StatusNotFound, "QUOTE_NOT_FOUND", "Saved quote not found.")
		return
	}

	c.JSON(http.StatusOK, quoteDetailResponse{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      quote,
	})
}

type createQuoteRequest struct {
	ProviderID         string `form:"provider_id"`
	ProviderRates       string `form:"provider_rates"`
	SelectedProviderID  string `form:"selected_provider_id"`
	ExchangeFrom       string `form:"exchange_from" validate:"required,iso4217"`
	ExchangeTo         string `form:"exchange_to" validate:"required,iso4217,nefield=ExchangeFrom"`
	SenderCountry      string `form:"sender_country" validate:"required,supported_country"`
	BaseRate           string `form:"base_rate" validate:"required"`
	MarkupPercentage   string `form:"markup_percentage"`
	FinalRate          string `form:"final_rate" validate:"required"`
	FeeType            string `form:"fee_type" validate:"required"`
	TransactionFee     string `form:"transaction_fee"`
	QuoteExpirySeconds string `form:"quote_expiry_seconds" validate:"required"`
	GeneratedAt        string `form:"generated_at"`
	Notes              string `form:"notes"`
}

type updateQuoteRequest struct {
	BaseRate           string `form:"base_rate"`
	MarkupPercentage   string `form:"markup_percentage"`
	FinalRate          string `form:"final_rate"`
	FeeType            string `form:"fee_type"`
	TransactionFee     string `form:"transaction_fee"`
	QuoteExpirySeconds string `form:"quote_expiry_seconds"`
	GeneratedAt        string `form:"generated_at"`
	Notes              string `form:"notes"`
	ProviderRates      string `form:"provider_rates"`
	SelectedProviderID string `form:"selected_provider_id"`
}

func (h *QuotesHandler) CreateQuote(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "Unable to parse form data.")
		return
	}

	var request createQuoteRequest
	request.ProviderID = c.PostForm("provider_id")
	request.ExchangeFrom = c.PostForm("exchange_from")
	request.ExchangeTo = c.PostForm("exchange_to")
	request.SenderCountry = c.PostForm("sender_country")
	request.BaseRate = c.PostForm("base_rate")
	request.MarkupPercentage = c.PostForm("markup_percentage")
	request.FinalRate = c.PostForm("final_rate")
	request.FeeType = c.PostForm("fee_type")
	request.TransactionFee = c.PostForm("transaction_fee")
	request.QuoteExpirySeconds = c.PostForm("quote_expiry_seconds")
	request.GeneratedAt = c.PostForm("generated_at")
	request.Notes = c.PostForm("notes")
	request.ProviderRates = c.PostForm("provider_rates")
	request.SelectedProviderID = c.PostForm("selected_provider_id")

	if err := h.validate.Struct(request); err != nil {
		h.respondWithError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "Invalid quote payload.")
		return
	}

	baseRate, err := decimal.NewFromString(request.BaseRate)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_BASE_RATE", "base_rate must be a valid decimal string.")
		return
	}
	markupPercentage := decimal.Zero
	if request.MarkupPercentage != "" {
		markupPercentage, err = decimal.NewFromString(request.MarkupPercentage)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_MARKUP_PERCENTAGE", "markup_percentage must be a valid decimal string.")
			return
		}
	}
	finalRate, err := decimal.NewFromString(request.FinalRate)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_FINAL_RATE", "final_rate must be a valid decimal string.")
		return
	}
	transactionFee := decimal.Zero
	if request.TransactionFee != "" {
		transactionFee, err = decimal.NewFromString(request.TransactionFee)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_TRANSACTION_FEE", "transaction_fee must be a valid decimal string.")
			return
		}
	}
	quoteExpirySeconds, err := strconv.Atoi(request.QuoteExpirySeconds)
	if err != nil || quoteExpirySeconds <= 0 {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_QUOTE_EXPIRY", "quote_expiry_seconds must be a positive integer.")
		return
	}
	generatedAt := time.Now().UTC()
	if request.GeneratedAt != "" {
		generatedAt, err = time.Parse(time.RFC3339, request.GeneratedAt)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_GENERATED_AT", "generated_at must use RFC3339 format.")
			return
		}
	}

	quoteID := uuid.New()
	evidenceURL := ""
	if file, err := c.FormFile("evidence_file"); err == nil {
		evidenceURL, err = h.storage.UploadQuoteEvidence(c.Request.Context(), quoteID, file)
		if err != nil {
			h.respondWithError(c, http.StatusInternalServerError, "EVIDENCE_UPLOAD_FAILED", fmt.Sprintf("Failed to upload evidence: %v", err))
			return
		}
	} else if err != nil && err != http.ErrMissingFile {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_EVIDENCE_FILE", "Invalid evidence file upload.")
		return
	}

	createReq := transactions.CreateQuoteTransactionRequest{
		QuoteID:            quoteID,
		ProviderID:         request.ProviderID,
		ProviderRates:      nil,
		SelectedProviderID: request.SelectedProviderID,
		ExchangeFrom:       request.ExchangeFrom,
		ExchangeTo:         request.ExchangeTo,
		SenderCountry:      request.SenderCountry,
		BaseRate:           baseRate,
		MarkupPercentage:   markupPercentage,
		FinalRate:          finalRate,
		FeeType:            request.FeeType,
		TransactionFee:     transactionFee,
		QuoteExpirySeconds: quoteExpirySeconds,
		GeneratedAt:        generatedAt,
		EvidenceURL:        evidenceURL,
		Notes:              request.Notes,
	}

	// parse provider_rates JSON if provided
	if request.ProviderRates != "" {
		var prs []transactions.ProviderRate
		if err := json.Unmarshal([]byte(request.ProviderRates), &prs); err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_PROVIDER_RATES", "provider_rates must be valid JSON array of {provider_id,rate}")
			return
		}
		createReq.ProviderRates = prs
	}

	quote, err := h.transactionService.Create(c.Request.Context(), createReq)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "SAVE_QUOTE_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusCreated, quoteDetailResponse{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      quote,
	})
}

func (h *QuotesHandler) UpdateQuote(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_QUOTE_ID", "Quote id is invalid.")
		return
	}

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_PAYLOAD", "Unable to parse form data.")
		return
	}

	var request updateQuoteRequest
	request.BaseRate = c.PostForm("base_rate")
	request.MarkupPercentage = c.PostForm("markup_percentage")
	request.FinalRate = c.PostForm("final_rate")
	request.FeeType = c.PostForm("fee_type")
	request.TransactionFee = c.PostForm("transaction_fee")
	request.QuoteExpirySeconds = c.PostForm("quote_expiry_seconds")
	request.GeneratedAt = c.PostForm("generated_at")
	request.Notes = c.PostForm("notes")

	var updateReq transactions.UpdateQuoteTransactionRequest
	if request.BaseRate != "" {
		value, err := decimal.NewFromString(request.BaseRate)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_BASE_RATE", "base_rate must be a valid decimal string.")
			return
		}
		updateReq.BaseRate = &value
	}
	if request.MarkupPercentage != "" {
		value, err := decimal.NewFromString(request.MarkupPercentage)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_MARKUP_PERCENTAGE", "markup_percentage must be a valid decimal string.")
			return
		}
		updateReq.MarkupPercentage = &value
	}
	if request.FinalRate != "" {
		value, err := decimal.NewFromString(request.FinalRate)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_FINAL_RATE", "final_rate must be a valid decimal string.")
			return
		}
		updateReq.FinalRate = &value
	}
	if request.FeeType != "" {
		updateReq.FeeType = &request.FeeType
	}
	if request.TransactionFee != "" {
		value, err := decimal.NewFromString(request.TransactionFee)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_TRANSACTION_FEE", "transaction_fee must be a valid decimal string.")
			return
		}
		updateReq.TransactionFee = &value
	}
	if request.QuoteExpirySeconds != "" {
		value, err := strconv.Atoi(request.QuoteExpirySeconds)
		if err != nil || value <= 0 {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_QUOTE_EXPIRY", "quote_expiry_seconds must be a positive integer.")
			return
		}
		updateReq.QuoteExpirySeconds = &value
	}
	if request.GeneratedAt != "" {
		value, err := time.Parse(time.RFC3339, request.GeneratedAt)
		if err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_GENERATED_AT", "generated_at must use RFC3339 format.")
			return
		}
		updateReq.GeneratedAt = &value
	}
	if request.Notes != "" {
		updateReq.Notes = &request.Notes
	}
	// parse provider rates and selected provider id for update
	if request.ProviderRates != "" {
		var prs []transactions.ProviderRate
		if err := json.Unmarshal([]byte(request.ProviderRates), &prs); err != nil {
			h.respondWithError(c, http.StatusBadRequest, "INVALID_PROVIDER_RATES", "provider_rates must be valid JSON array of {provider_id,rate}")
			return
		}
		updateReq.ProviderRates = &prs
	}
	if request.SelectedProviderID != "" {
		updateReq.SelectedProviderID = &request.SelectedProviderID
	}

	if file, err := c.FormFile("evidence_file"); err == nil {
		evidenceURL, err := h.storage.UploadQuoteEvidence(c.Request.Context(), id, file)
		if err != nil {
			h.respondWithError(c, http.StatusInternalServerError, "EVIDENCE_UPLOAD_FAILED", fmt.Sprintf("Failed to upload evidence: %v", err))
			return
		}
		updateReq.EvidenceURL = &evidenceURL
	} else if err != nil && err != http.ErrMissingFile {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_EVIDENCE_FILE", "Invalid evidence file upload.")
		return
	}

	quote, err := h.transactionService.Update(c.Request.Context(), id, updateReq)
	if err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "UPDATE_QUOTE_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, quoteDetailResponse{
		Success:   true,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Data:      quote,
	})
}

func (h *QuotesHandler) DeleteQuote(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		h.respondWithError(c, http.StatusBadRequest, "INVALID_QUOTE_ID", "Quote id is invalid.")
		return
	}

	if err := h.transactionService.Delete(c.Request.Context(), id); err != nil {
		h.respondWithError(c, http.StatusInternalServerError, "DELETE_QUOTE_FAILED", err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *QuotesHandler) respondWithError(c *gin.Context, status int, code, message string) {
	c.JSON(status, quotesErrorResponse{
		Success:   false,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Error: map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
