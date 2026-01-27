package middleware

import (
	"time"

	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

// CanonicalLoggerMiddleware creates a middleware that logs once per request
func CanonicalLoggerMiddleware(log *logger.CanonicalLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Initialize LogContext for this request
		logCtx := logger.NewLogContext()

		// Store in Fiber locals for handler access
		c.Locals("log_context", logCtx)

		// Add LogContext to user context for usecase/repository access
		userCtx := logger.WithLogContext(c.UserContext(), logCtx)
		c.SetUserContext(userCtx)

		// Get request ID from Fiber's requestid middleware
		if reqID := c.Locals("requestid"); reqID != nil {
			if id, ok := reqID.(string); ok {
				logCtx.AddField(zap.String(logger.FieldRequestID, id))
			}
		}

		// Record start time
		start := time.Now()

		// Use defer to ensure logging happens even on panic (after recover middleware)
		defer func() {
			duration := time.Since(start)
			status := c.Response().StatusCode()

			// Build base fields
			fields := []zap.Field{
				zap.String("method", c.Method()),
				zap.String("path", c.Path()),
				zap.Int("status", status),
				zap.Duration("duration", duration),
				zap.Int64("duration_ms", duration.Milliseconds()),
			}

			// Add accumulated fields from handlers/usecases
			fields = append(fields, logCtx.Fields()...)

			// Log based on status code
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
