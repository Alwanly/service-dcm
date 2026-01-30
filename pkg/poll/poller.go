package poll

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Alwanly/service-distribute-management/pkg/logger"

	"go.uber.org/zap"
)

// poller implements the Poller interface
type poller struct {
	logger    *logger.CanonicalLogger
	fetchMeta map[string]pollMeta
	tickers   map[string]*time.Ticker
	stopChans map[string]chan struct{}
	mu        sync.RWMutex
	started   bool
}

type pollMeta struct {
	FetchFunc           FetchFunc
	PollIntervalSeconds int
}

// NewPoller creates a new Poller instance
func NewPoller(logger *logger.CanonicalLogger) Poller {
	return &poller{
		logger:    logger,
		fetchMeta: make(map[string]pollMeta),
		tickers:   make(map[string]*time.Ticker),
		stopChans: make(map[string]chan struct{}),
	}
}

// RegisterFetchFunc registers a periodic fetch function with configuration
func (p *poller) RegisterFetchFunc(name string, fetchFunc FetchFunc, config PollerConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.fetchMeta[name] = pollMeta{
		FetchFunc:           fetchFunc,
		PollIntervalSeconds: config.PollIntervalSeconds,
	}

	p.logger.Info("registered fetch function",
		zap.String("name", name),
		zap.Int("poll_interval_seconds", config.PollIntervalSeconds),
	)
}

// Start begins polling for all registered fetch functions
func (p *poller) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.started {
		p.mu.Unlock()
		return fmt.Errorf("poller already started")
	}
	p.started = true

	// Create tickers and start polling goroutines
	for name, meta := range p.fetchMeta {
		interval := time.Duration(meta.PollIntervalSeconds) * time.Second
		p.tickers[name] = time.NewTicker(interval)
		p.stopChans[name] = make(chan struct{})

		go p.pollLoop(ctx, name, meta.FetchFunc, p.tickers[name], p.stopChans[name])
	}
	p.mu.Unlock()

	p.logger.Info("poller started", zap.Int("fetch_functions", len(p.fetchMeta)))
	return nil
}

// pollLoop runs the periodic polling for a single fetch function
func (p *poller) pollLoop(ctx context.Context, name string, fetchFunc FetchFunc, ticker *time.Ticker, stopChan chan struct{}) {
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("poll loop stopped due to context cancellation", zap.String("name", name))
			return
		case <-stopChan:
			p.logger.Info("poll loop stopped", zap.String("name", name))
			return
		case <-ticker.C:
			// create a logger instance with extra fields
			pollLogger := p.logger.WithAgentID(name)

			if err := fetchFunc(ctx, pollLogger); err != nil {
				p.logger.Error("fetch function failed", zap.String("poll_name", name), zap.Error(err))
			}
		}
	}
}

// UpdateInterval dynamically updates the polling interval for a registered fetch function
func (p *poller) UpdateInterval(name string, newIntervalSeconds int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Validate interval
	if newIntervalSeconds <= 0 {
		return fmt.Errorf("invalid interval: must be positive, got %d", newIntervalSeconds)
	}

	// Check if fetch function exists
	meta, exists := p.fetchMeta[name]
	if !exists {
		return fmt.Errorf("fetch function %q not registered", name)
	}

	// If interval hasn't changed, do nothing
	if meta.PollIntervalSeconds == newIntervalSeconds {
		p.logger.Debug("poll interval unchanged, skipping update",
			zap.String("name", name),
			zap.Int("interval_seconds", newIntervalSeconds),
		)
		return nil
	}

	// Update metadata
	meta.PollIntervalSeconds = newIntervalSeconds
	p.fetchMeta[name] = meta

	// If poller is running, restart the ticker
	if p.started {
		// Stop old ticker
		if ticker, ok := p.tickers[name]; ok {
			ticker.Stop()
		}

		// Create new ticker with updated interval
		newInterval := time.Duration(newIntervalSeconds) * time.Second
		p.tickers[name] = time.NewTicker(newInterval)

		// Signal old goroutine to stop
		if stopChan, ok := p.stopChans[name]; ok {
			close(stopChan)
		}

		// Create new stop channel and start new goroutine
		p.stopChans[name] = make(chan struct{})

		// Get fresh context from parent (reconstruct if needed)
		ctx := context.Background()
		go p.pollLoop(ctx, name, meta.FetchFunc, p.tickers[name], p.stopChans[name])

		p.logger.Info("poll interval updated",
			zap.String("name", name),
			zap.Int("new_interval_seconds", newIntervalSeconds),
		)
	}

	return nil
}

// Stop halts all polling operations
func (p *poller) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.started {
		return fmt.Errorf("poller not started")
	}

	// Stop all tickers and signal goroutines to stop
	for name, ticker := range p.tickers {
		ticker.Stop()
		if stopChan, ok := p.stopChans[name]; ok {
			close(stopChan)
		}
	}

	p.started = false
	p.logger.Info("poller stopped")
	return nil
}
