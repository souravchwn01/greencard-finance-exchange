package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type healthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

type HealthHandler struct {
	appVersion string
}

func NewHealthHandler(appVersion string) *HealthHandler {
	return &HealthHandler{appVersion: appVersion}
}

func (h *HealthHandler) GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, healthResponse{
		Status:    "UP",
		Version:   h.appVersion,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
