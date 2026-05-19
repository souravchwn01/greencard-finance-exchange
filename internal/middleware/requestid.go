package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	requestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

type contextRequestIDKey struct{}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(requestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Writer.Header().Set(requestIDHeader, requestID)
		c.Set(requestIDKey, requestID)

		ctx := context.WithValue(c.Request.Context(), contextRequestIDKey{}, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}
