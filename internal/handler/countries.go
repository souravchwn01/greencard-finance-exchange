package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type CountriesHandler struct {
	countriesFilePath string
}

func NewCountriesHandler(countriesFilePath string) *CountriesHandler {
	return &CountriesHandler{countriesFilePath: countriesFilePath}
}

func (h *CountriesHandler) GetSupportedCountries(c *gin.Context) {
	content, err := os.ReadFile(h.countriesFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"error":     gin.H{"code": "INTERNAL_ERROR", "message": "Failed to load supported countries."},
		})
		return
	}

	var raw json.RawMessage
	if err := json.Unmarshal(content, &raw); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":   false,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"error":     gin.H{"code": "INTERNAL_ERROR", "message": "Failed to parse supported countries."},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"data":      raw,
	})
}
