package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/poll"
)

type useCase struct {
	controllerClient repository.IControllerClient
	workerClient     repository.IWorkerClient
	config           *config.AgentConfig
	logger           *logger.CanonicalLogger

	agentID      string
	agentIDMutex sync.RWMutex

	poller   poll.Poller
	stopCh   chan struct{}
	isActive bool
	mu       sync.RWMutex
}

// NewUseCase creates a new agent usecase
func NewUseCase(
	controllerClient repository.IControllerClient,
	workerClient repository.IWorkerClient,
	cfg *config.AgentConfig,
	log *logger.CanonicalLogger,
) IUseCase {
	return &useCase{
		controllerClient: controllerClient,
		workerClient:     workerClient,
		config:           cfg,
		logger:           log,
		stopCh:           make(chan struct{}),
	}
}

// RegisterWithController registers the agent with the controller
func (uc *useCase) RegisterWithController(ctx context.Context) (string, error) {
	uc.logger.Info("attempting to register with controller")

	resp, err := uc.controllerClient.Register(ctx, "", "", "")
	if err != nil {
		return "", fmt.Errorf("failed to register: %w", err)
	}

	uc.agentIDMutex.Lock()
	uc.agentID = resp.AgentID
	uc.agentIDMutex.Unlock()

	uc.logger.Info("successfully registered with controller")

	return resp.AgentID, nil
}

// StartPolling starts the configuration polling process
func (uc *useCase) StartPolling(ctx context.Context, agentID string) error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if uc.isActive {
		return fmt.Errorf("polling is already active")
	}

	// Create fetch function for poller
	fetchFunc := func(ctx context.Context, currentETag string) (interface{}, string, error) {
		config, newETag, err := uc.controllerClient.GetConfiguration(ctx, agentID, currentETag)
		if err != nil {
			return nil, currentETag, err
		}
		if config == nil {
			return nil, currentETag, nil
		}
		return config, newETag, nil
	}

	// Create poller
	pollConfig := poll.Config{
		Interval:    uc.config.PollInterval,
		InitialETag: "",
	}
	uc.poller = poll.NewPoller(fetchFunc, pollConfig, uc.logger)

	updateCh, err := uc.poller.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start poller: %w", err)
	}

	uc.isActive = true
	uc.logger.Info("polling started successfully")

	go uc.handleConfigUpdates(ctx, updateCh)

	return nil
}

// StopPolling stops the configuration polling process
func (uc *useCase) StopPolling() error {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if !uc.isActive {
		return fmt.Errorf("polling is not active")
	}

	if uc.poller != nil {
		if err := uc.poller.Stop(); err != nil {
			return fmt.Errorf("failed to stop poller: %w", err)
		}
	}

	close(uc.stopCh)
	uc.isActive = false
	uc.logger.Info("polling stopped successfully")

	return nil
}

// GetAgentID returns the current agent ID
func (uc *useCase) GetAgentID() string {
	uc.agentIDMutex.RLock()
	defer uc.agentIDMutex.RUnlock()
	return uc.agentID
}

// GetStatus returns the agent status information
func (uc *useCase) GetStatus() map[string]interface{} {
	uc.mu.RLock()
	defer uc.mu.RUnlock()

	return map[string]interface{}{
		"agent_id":  uc.GetAgentID(),
		"is_active": uc.isActive,
	}
}

// handleConfigUpdates processes configuration updates from the poller
func (uc *useCase) handleConfigUpdates(ctx context.Context, updateCh <-chan poll.ConfigUpdateMessage) {
	for {
		select {
		case <-ctx.Done():
			uc.logger.Info("stopping config update handler due to context cancellation")
			return
		case <-uc.stopCh:
			uc.logger.Info("stopping config update handler")
			return
		case msg, ok := <-updateCh:
			if !ok {
				uc.logger.Info("update channel closed")
				return
			}
			uc.processConfigUpdate(ctx, msg)
		}
	}
}

// processConfigUpdate processes a single configuration update
func (uc *useCase) processConfigUpdate(ctx context.Context, msg poll.ConfigUpdateMessage) {
	config, ok := msg.Config.(*models.WorkerConfiguration)
	if !ok {
		uc.logger.Error("received config update with unexpected type")
		return
	}

	uc.logger.WithConfigVersion(config.Version).Info("processing configuration update")

	if err := uc.forwardConfigToWorker(ctx, config); err != nil {
		uc.logger.WithError(err).Error("failed to forward configuration to worker")
		return
	}

	uc.logger.Info("configuration update processed successfully")
}

// forwardConfigToWorker sends the configuration to the worker
func (uc *useCase) forwardConfigToWorker(ctx context.Context, config *models.WorkerConfiguration) error {
	if err := uc.workerClient.SendConfiguration(ctx, config); err != nil {
		return fmt.Errorf("failed to send config to worker: %w", err)
	}
	return nil
}
