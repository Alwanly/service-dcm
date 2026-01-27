package logger

import (
	"go.uber.org/zap"
)

// CanonicalLogger is a small wrapper around zap.Logger for this project
type CanonicalLogger struct {
	l *zap.Logger
}

// NewLoggerFromEnv creates a new logger. For simplicity, use zap.NewProduction.
func NewLoggerFromEnv(component string) (*CanonicalLogger, error) {
	cfg := zap.NewProductionConfig()
	l, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return &CanonicalLogger{l: l}, nil
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

func (c *CanonicalLogger) WithConfigVersion(v int64) *CanonicalLogger {
	return &CanonicalLogger{l: c.l.With(zap.Int64("config_version", v))}
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
