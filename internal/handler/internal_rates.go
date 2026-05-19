package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/gfc-app-finance/greencard-mobile/exchange/internal/rates"
)

type InternalRatesHandler struct {
	stream *rates.Stream
}

func NewInternalRatesHandler(stream *rates.Stream) *InternalRatesHandler {
	return &InternalRatesHandler{stream: stream}
}

func (h *InternalRatesHandler) PostRate(c *gin.Context) {
	var req rates.RateEvent
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "INVALID_PAYLOAD", "message": "Invalid JSON payload."}})
		return
	}

	if _, err := uuid.Parse(req.EventID); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": gin.H{"code": "INVALID_EVENT_ID", "message": "event_id must be a valid UUID."}})
		return
	}
	if req.ProviderID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": gin.H{"code": "INVALID_PROVIDER_ID", "message": "provider_id is required."}})
		return
	}
	if err := rates.ValidatePair(req.Pair); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": gin.H{"code": "INVALID_PAIR", "message": err.Error()}})
		return
	}
	if req.Rate <= 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": gin.H{"code": "INVALID_RATE", "message": "rate must be a positive number."}})
		return
	}
	if req.Timestamp.IsZero() {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"success": false, "error": gin.H{"code": "INVALID_TIMESTAMP", "message": "timestamp is required."}})
		return
	}
	req.Timestamp = req.Timestamp.UTC().Truncate(time.Nanosecond)

	if err := h.stream.Enqueue(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "ENQUEUE_FAILED", "message": "Failed to enqueue rate event."}})
		return
	}

	c.Status(http.StatusAccepted)
}

