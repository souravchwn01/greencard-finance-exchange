package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const loggerKey = "logger"

func NewZapLogger(appEnv, logLevel string) (*zap.Logger, error) {
	var cfg zap.Config
	if strings.EqualFold(strings.TrimSpace(appEnv), "production") {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	level, err := parseLogLevel(logLevel)
	if err != nil {
		return nil, err
	}
	cfg.Level = zap.NewAtomicLevelAt(level)

	return cfg.Build()
}

func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Set(loggerKey, logger)

		c.Next()

		logger.Info(
			"http_request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("latency_ms", time.Since(start).Milliseconds()),
			zap.String("request_id", c.GetString(requestIDKey)),
		)
	}
}

func parseLogLevel(logLevel string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(logLevel)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid LOG_LEVEL: %s", logLevel)
	}
}
