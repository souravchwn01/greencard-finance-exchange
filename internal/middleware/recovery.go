package middleware

import (
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apperrors "github.com/gfc-app-finance/greencard-mobile/exchange/internal/errors"
)

type recoveryErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type recoveryErrorResponse struct {
	Success   bool              `json:"success"`
	Timestamp string            `json:"timestamp"`
	Error     recoveryErrorBody `json:"error"`
}

func Recovery(baseLogger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				requestID := c.GetString(requestIDKey)

				logger := baseLogger
				if contextLogger, ok := c.Get(loggerKey); ok {
					if typedLogger, ok := contextLogger.(*zap.Logger); ok {
						logger = typedLogger
					}
				}

				logger.Error(
					"panic_recovered",
					zap.Any("panic", recovered),
					zap.String("request_id", requestID),
					zap.ByteString("stack", debug.Stack()),
				)

				c.AbortWithStatusJSON(http.StatusInternalServerError, recoveryErrorResponse{
					Success:   false,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Error: recoveryErrorBody{
						Code:    apperrors.ErrInternal.Code,
						Message: apperrors.ErrInternal.Message,
					},
				})
			}
		}()

		c.Next()
	}
}
