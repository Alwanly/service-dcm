package logger

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type contextKey string

const (
	logContextKey contextKey = "log_context"
)

const (
	correlationKey contextKey = "correlation_id"
)

const (
	FieldRequestID     = "request_id"
	FieldOperation     = "operation"
	FieldAgentID       = "agent_id"
	FieldConfigVersion = "config_version"
	FieldTargetURL     = "target_url"
	FieldProxyStatus   = "proxy_status"
	FieldSuccess       = "success"
	FieldETag          = "etag"

	// Poller-specific field names
	FieldPollName     = "poll_name"
	FieldFetchCount   = "fetch_count"
	FieldSuccessCount = "success_count"
	FieldFailedCount  = "failed_count"
)

type LogContext struct {
	mu     sync.RWMutex
	fields []zap.Field
}

func NewLogContext() *LogContext {
	return &LogContext{
		fields: make([]zap.Field, 0, 10),
	}
}

func (lc *LogContext) AddField(field zap.Field) {
	if lc == nil {
		return
	}
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.fields = append(lc.fields, field)
}

func (lc *LogContext) AddFields(fields ...zap.Field) {
	if lc == nil {
		return
	}
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.fields = append(lc.fields, fields...)
}

func (lc *LogContext) Fields() []zap.Field {
	if lc == nil {
		return nil
	}
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	result := make([]zap.Field, len(lc.fields))
	copy(result, lc.fields)
	return result
}

func WithLogContext(ctx context.Context, lc *LogContext) context.Context {
	return context.WithValue(ctx, logContextKey, lc)
}

func GetLogContext(ctx context.Context) *LogContext {
	if ctx == nil {
		return nil
	}
	lc, ok := ctx.Value(logContextKey).(*LogContext)
	if !ok {
		return nil
	}
	return lc
}

func AddToContext(ctx context.Context, fields ...zap.Field) {
	lc := GetLogContext(ctx)
	if lc != nil {
		lc.AddFields(fields...)
	}
}

func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationKey, id)
}

func GetCorrelationID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(correlationKey).(string); ok {
		return v
	}
	return ""
}
