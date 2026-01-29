package logger

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

// Context key type for storing LogContext
type contextKey string

const (
	logContextKey contextKey = "log_context"
)

// Field name constants for consistency
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

// LogContext accumulates log fields throughout request lifecycle
type LogContext struct {
	mu     sync.RWMutex
	fields []zap.Field
}

// NewLogContext creates a new LogContext instance
func NewLogContext() *LogContext {
	return &LogContext{
		fields: make([]zap.Field, 0, 10),
	}
}

// AddField adds a field to the context
func (lc *LogContext) AddField(field zap.Field) {
	if lc == nil {
		return
	}
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.fields = append(lc.fields, field)
}

// AddFields adds multiple fields to the context
func (lc *LogContext) AddFields(fields ...zap.Field) {
	if lc == nil {
		return
	}
	lc.mu.Lock()
	defer lc.mu.Unlock()
	lc.fields = append(lc.fields, fields...)
}

// Fields returns a copy of all accumulated fields
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

// WithLogContext adds LogContext to a context.Context
func WithLogContext(ctx context.Context, lc *LogContext) context.Context {
	return context.WithValue(ctx, logContextKey, lc)
}

// GetLogContext retrieves LogContext from context.Context
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

// AddToContext is a helper that retrieves LogContext and adds fields
func AddToContext(ctx context.Context, fields ...zap.Field) {
	lc := GetLogContext(ctx)
	if lc != nil {
		lc.AddFields(fields...)
	}
}
