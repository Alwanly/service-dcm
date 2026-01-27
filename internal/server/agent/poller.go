package agent

import (
	"context"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
)

// Poller handles periodic polling of the controller
type Poller struct {
	client         *ControllerClient
	interval       time.Duration
	currentETag    string
	agentID        string
	onConfigChange func(*models.WorkerConfiguration)
	logger         *logger.CanonicalLogger
}

// NewPoller creates a new poller
func NewPoller(client *ControllerClient, interval time.Duration, agentID string, onConfigChange func(*models.WorkerConfiguration)) *Poller {
	log, _ := logger.NewLoggerFromEnv("agent")

	return &Poller{
		client:         client,
		interval:       interval,
		agentID:        agentID,
		onConfigChange: onConfigChange,
		logger:         log.Component("poller"),
	}
}

// Start begins the polling loop
func (p *Poller) Start(ctx context.Context) error {
	p.logger.Info("starting configuration polling",
		logger.Duration("interval", p.interval),
	)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Do initial poll
	if err := p.poll(ctx); err != nil {
		p.logger.WithError(err).Error("initial poll failed")
	}

	// Continue polling
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("stopping poller")
			return ctx.Err()
		case <-ticker.C:
			if err := p.poll(ctx); err != nil {
				p.logger.WithError(err).Error("poll failed")
			}
		}
	}
}

// poll fetches configuration from controller
func (p *Poller) poll(ctx context.Context) error {
	config, newETag, err := p.client.GetConfiguration(ctx, p.agentID, p.currentETag)
	if err != nil {
		return err
	}

	// No configuration exists yet
	if config == nil && newETag == "" {
		p.logger.Debug("no configuration available yet")
		return nil
	}

	// Configuration unchanged
	if config == nil && newETag == p.currentETag {
		p.logger.Debug("configuration unchanged",
			logger.String("etag", p.currentETag),
		)
		return nil
	}

	// Configuration changed
	if config != nil {
		p.logger.Info("configuration changed",
			logger.String("old_etag", p.currentETag),
			logger.String("new_etag", newETag),
			logger.Int64("version", config.Version),
		)

		p.currentETag = newETag

		if p.onConfigChange != nil {
			p.onConfigChange(config)
		}
	}

	return nil
}
