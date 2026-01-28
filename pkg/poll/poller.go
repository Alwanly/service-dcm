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
	fetchFunc   FetchFunc
	interval    time.Duration
	currentETag string
	logger      *logger.CanonicalLogger
	stopCh      chan struct{}
	updateCh    chan ConfigUpdateMessage
}

// NewPoller creates a new Poller instance
func NewPoller(fetchFunc FetchFunc, config Config, log *logger.CanonicalLogger) Poller {
	return &poller{
		fetchFunc:   fetchFunc,
		interval:    config.Interval,
		currentETag: config.InitialETag,
		logger:      log,
		stopCh:      make(chan struct{}),
		updateCh:    make(chan ConfigUpdateMessage, 1),
	}
}

// Start begins polling and returns a channel for config updates
func (p *poller) Start(ctx context.Context) (<-chan ConfigUpdateMessage, error) {
	if p.fetchFunc == nil {
		return nil, fmt.Errorf("fetch function cannot be nil")
	}

	go p.poll(ctx)
	return p.updateCh, nil
}

// Stop gracefully stops the poller
func (p *poller) Stop() error {
	close(p.stopCh)
	close(p.updateCh)
	return nil
}

// poll performs the polling loop
func (p *poller) poll(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Perform initial poll immediately
	p.performPoll(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("poller stopped due to context cancellation")
			return
		case <-p.stopCh:
			p.logger.Info("poller stopped")
			return
		case <-ticker.C:
			p.performPoll(ctx)
		}
	}
}

// performPoll executes a single poll operation
func (p *poller) performPoll(ctx context.Context) {
	config, newETag, err := p.fetchFunc(ctx, p.currentETag)
	if err != nil {
		p.logger.WithError(err).Error("failed to fetch configuration")
		return
	}

	// No update if ETag hasn't changed (fetchFunc should return same ETag)
	if newETag == p.currentETag {
		p.logger.Debug("no configuration update available")
		return
	}

	p.logger.Info("configuration update detected", zap.String("old_etag", p.currentETag), zap.String("new_etag", newETag))

	p.currentETag = newETag

	// Send update to channel (non-blocking)
	select {
	case p.updateCh <- ConfigUpdateMessage{Config: config, ETag: newETag}:
		p.logger.Debug("configuration update sent to channel")
	default:
		p.logger.Info("configuration update channel full, skipping")
	}
}
