package middleware

import (
	"context"
	"log/slog"
	"time"

	"order-playground/apps/gateway-api/internal/domain"

	"github.com/gin-gonic/gin"
)

func RequestContext(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := time.Now().UTC().Format("20060102150405.000000000")
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

func Logger(logger *slog.Logger, audit requestLogWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		user := ""
		if claims, ok := UserFromContext(c.Request.Context()); ok {
			user = claims.Username
		}
		requestID, _ := c.Get("request_id")
		entry := domain.RequestLog{
			RequestID: requestID.(string),
			Path:      c.Request.URL.Path,
			Method:    c.Request.Method,
			Status:    c.Writer.Status(),
			LatencyMS: time.Since(start).Milliseconds(),
			User:      user,
			CreatedAt: time.Now().UTC(),
		}

		logger.Info("request completed",
			"request_id", entry.RequestID,
			"method", entry.Method,
			"path", entry.Path,
			"status", entry.Status,
			"latency_ms", entry.LatencyMS,
			"user", entry.User,
		)

		if audit != nil {
			if err := audit.WriteRequestLog(c.Request.Context(), entry); err != nil {
				logger.Warn("failed to write request log", "error", err)
			}
		}
	}
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		c.AbortWithStatusJSON(500, gin.H{"error": "internal server error"})
	})
}

type requestLogWriter interface {
	WriteRequestLog(context.Context, domain.RequestLog) error
}
