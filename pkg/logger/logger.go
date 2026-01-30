package logger

import (
	"os"

	"go.uber.org/zap"
)

type CanonicalLogger struct {
	l *zap.Logger
}

// NewLoggerFromEnv creates a new logger based on the LOG_FORMAT environment variable.
// Supported LOG_FORMAT values:
//   - "console" or "development": Human-readable console output with colored levels, ISO8601 timestamps
//   - "json" or "production" (default): Structured JSON output for production environments
//
// The logger automatically skips one caller frame to report the actual calling code
// instead of the wrapper function location.
func NewLoggerFromEnv(component string) (*CanonicalLogger, error) {
	// Read LOG_FORMAT environment variable with default to "production"
	logFormat := os.Getenv("LOG_FORMAT")
	if logFormat == "" {
		logFormat = "production"
	}

	// Select configuration based on environment
	var cfg zap.Config
	if logFormat == "console" || logFormat == "development" {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}

	// Build logger with AddCallerSkip(1) to skip the wrapper frame
	// This ensures the caller field shows the actual calling code, not the wrapper
	zapLogger, err := cfg.Build(
		zap.AddCallerSkip(1),
		zap.Fields(zap.String("component", component)),
	)
	if err != nil {
		return nil, err
	}

	return &CanonicalLogger{
		l: zapLogger,
	}, nil
}

func (c *CanonicalLogger) Sync() {
	_ = c.l.Sync()
}

func (c *CanonicalLogger) Info(msg string, fields ...zap.Field) {
	c.l.Info(msg, fields...)
}

func (c *CanonicalLogger) Debug(msg string, fields ...zap.Field) {
	c.l.Debug(msg, fields...)
}

func (c *CanonicalLogger) Error(msg string, fields ...zap.Field) {
	c.l.Error(msg, fields...)
}

func (c *CanonicalLogger) Fatal(msg string, fields ...zap.Field) {
	c.l.Fatal(msg, fields...)
}

func (c *CanonicalLogger) WithError(err error) *CanonicalLogger {
	return &CanonicalLogger{l: c.l.With(zap.Error(err))}
}

func (c *CanonicalLogger) WithAgentID(id string) *CanonicalLogger {
	return &CanonicalLogger{l: c.l.With(zap.String("agent_id", id))}
}

func (c *CanonicalLogger) WithConfigVersion(v string) *CanonicalLogger {
	return &CanonicalLogger{l: c.l.With(zap.String("config_version", v))}
}

func (c *CanonicalLogger) Component(name string) *CanonicalLogger {
	return &CanonicalLogger{l: c.l.With(zap.String("component", name))}
}

func (c *CanonicalLogger) HTTP(method, path string, status int, durationMs int64) {
	c.l.Info("http_request", zap.String("method", method), zap.String("path", path), zap.Int("status", status), zap.Int64("duration_ms", durationMs))
}

func (c *CanonicalLogger) HTTPError(method, path string, status int, err error) {
	c.l.Error("http_error", zap.String("method", method), zap.String("path", path), zap.Int("status", status), zap.Error(err))
}
