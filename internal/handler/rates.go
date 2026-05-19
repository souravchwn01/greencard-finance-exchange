package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/rates"
)

type RatesHandler struct {
	rateService rates.Reader
}

func NewRatesHandler(rateService rates.Reader) *RatesHandler {
	return &RatesHandler{rateService: rateService}
}

func (h *RatesHandler) GetAllRates(c *gin.Context) {
	all, err := h.rateService.GetAllRates(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"error":     gin.H{"code": "INTERNAL_ERROR", "message": "Failed to fetch rates."},
		})
		return
	}

	if all == nil {
		all = []rates.CachedRate{}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      all,
	})
}

type updateRateRequest struct {
	Rate       float64 `json:"rate" binding:"required"`
	ProviderID string  `json:"provider_id"`
}

func (h *RatesHandler) UpdateRate(c *gin.Context) {
	pair := c.Param("pair")
	if pair == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"error":     gin.H{"code": "MISSING_PARAMETER", "message": "pair path parameter is required."},
		})
		return
	}

	var req updateRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":   false,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"error":     gin.H{"code": "INVALID_PAYLOAD", "message": "Invalid JSON payload."},
		})
		return
	}

	updated, err := h.rateService.UpdateRate(c.Request.Context(), pair, req.Rate, req.ProviderID)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success":   false,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"error":     gin.H{"code": "INVALID_RATE", "message": err.Error()},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      updated,
	})
}
