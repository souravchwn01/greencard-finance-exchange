package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// InternalAuth enforces either:
// - X-API-Key: <key>
// - Authorization: Bearer <token>
//
// At least one of apiKey or bearerToken must be configured.
func InternalAuth(apiKey, bearerToken string) gin.HandlerFunc {
	apiKey = strings.TrimSpace(apiKey)
	bearerToken = strings.TrimSpace(bearerToken)

	return func(c *gin.Context) {
		if apiKey != "" {
			if strings.TrimSpace(c.GetHeader("X-API-Key")) == apiKey {
				c.Next()
				return
			}
		}

		if bearerToken != "" {
			authz := strings.TrimSpace(c.GetHeader("Authorization"))
			if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
				token := strings.TrimSpace(authz[len("bearer "):])
				if token == bearerToken {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Invalid or missing credentials.",
			},
		})
	}
}

