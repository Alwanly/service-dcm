package poll

import (
	"context"
	"fmt"
	"time"

	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"go.uber.org/zap"
)

// poller implements the Poller interface
type poller struct {
	logger     *logger.CanonicalLogger
	stopCh     chan struct{}
	fetchFuncs map[string]MetaFunc
}

// NewPoller creates a new Poller instance
func NewPoller(log *logger.CanonicalLogger) Poller {
	return &poller{
		logger:     log,
		stopCh:     make(chan struct{}),
		fetchFuncs: make(map[string]MetaFunc),
	}
}

// Start begins polling and returns a channel for config updates
func (p *poller) Start(ctx context.Context) error {
	go p.poll(ctx)
	return nil
}

// Stop gracefully stops the poller
func (p *poller) Stop() error {
	close(p.stopCh)
	return nil
}

// poll performs the polling loop
func (p *poller) poll(ctx context.Context) {
	tickers := make(map[string]*time.Ticker)
	for name, meta := range p.fetchFuncs {
		interval := time.Duration(meta.PollIntervalSeconds) * time.Second
		tickers[name] = time.NewTicker(interval)
		p.logger.Info("started polling", zap.String("name", name), zap.Duration("interval", interval))
	}

	for {
		select {
		case <-p.stopCh:
			p.logger.Info("stopping poller")
			for _, ticker := range tickers {
				ticker.Stop()
			}
			return
		default:
			for name, ticker := range tickers {
				select {
				case <-ticker.C:
					p.logger.Debug("polling for configuration", zap.String("name", name))
					p.performPoll(ctx)
				default:
				}
			}
			time.Sleep(100 * time.Millisecond) // Prevent tight loop
		}
	}
}

// performPoll executes a single poll operation with canonical logging
func (p *poller) performPoll(ctx context.Context) {
	// Create LogContext for this poll cycle
	logCtx := logger.NewLogContext()

	// Add LogContext to context for propagation to FetchFuncs
	ctx = logger.WithLogContext(ctx, logCtx)

	// Add initial poll cycle metadata
	logCtx.AddFields(
		zap.String(logger.FieldOperation, "poll_cycle"),
		zap.Time("poll_start_time", time.Now()),
	)

	start := time.Now()

	var fetchCount, successCount, failedCount int
	var errors []string

	defer func() {
		duration := time.Since(start)

		// Build final summary fields
		fields := []zap.Field{
			zap.Duration("duration", duration),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Int(logger.FieldFetchCount, fetchCount),
			zap.Int(logger.FieldSuccessCount, successCount),
			zap.Int(logger.FieldFailedCount, failedCount),
		}

		// Add error details if any failures occurred
		if len(errors) > 0 {
			fields = append(fields, zap.Strings("errors", errors))
		}

		// Add all accumulated fields from FetchFuncs
		fields = append(fields, logCtx.Fields()...)

		// Single canonical log output
		if failedCount > 0 {
			p.logger.Error("poll_cycle_completed", fields...)
		} else {
			p.logger.Info("poll_cycle_completed", fields...)
		}
	}()

	for name, meta := range p.fetchFuncs {
		fetchCount++

		// Add fetch-specific context for this iteration
		logger.AddToContext(ctx, zap.String(logger.FieldPollName, name))

		err := meta.FetchFunc(ctx, p.logger)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))

			// Add failure context
			logger.AddToContext(ctx,
				zap.Bool(logger.FieldSuccess, false),
				zap.Error(err),
			)
			continue
		}
		successCount++

		// Add success context
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true))
	}
}

// RegisterFetchFunc registers a fetch function with its polling configuration
func (p *poller) RegisterFetchFunc(name string, fetchFunc FetchFunc, config PollerConfig) {
	if name == "" || fetchFunc == nil {
		p.logger.Error("invalid fetch function registration")
		return
	}
	if _, exists := p.fetchFuncs[name]; exists {
		panic("name already existing")
	}
	p.fetchFuncs[name] = MetaFunc{
		FetchFunc:    fetchFunc,
		PollerConfig: config,
	}
	p.logger.Info("fetch function registered", zap.String("name", name), zap.Int("poll_interval_seconds", config.PollIntervalSeconds))
}
