package middleware

import (
	"time"

	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

func CanonicalLoggerMiddleware(log *logger.CanonicalLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		logCtx := logger.NewLogContext()
		c.Locals("log_context", logCtx)
		userCtx := logger.WithLogContext(c.UserContext(), logCtx)
		c.SetUserContext(userCtx)
		if reqID := c.Locals("requestid"); reqID != nil {
			if id, ok := reqID.(string); ok {
				logCtx.AddField(zap.String(logger.FieldRequestID, id))
			}
		}
		start := time.Now()

		// Use defer to ensure logging happens even on panic (after recover middleware)
		defer func() {
			duration := time.Since(start)
			status := c.Response().StatusCode()
			fields := []zap.Field{
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", status),
				zap.Duration("duration", duration),
				zap.Int64("duration_ms", duration.Milliseconds()),
			}
			fields = append(fields, logCtx.Fields()...)
			if status >= 500 {
				log.Error("http_request", fields...)
			} else if status >= 400 {
				log.Info("http_request_client_error", fields...)
			} else {
				log.Info("http_request", fields...)
			}
		}()

		// Continue to next middleware/handler
		return c.Next()
	}
}
