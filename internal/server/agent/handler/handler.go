package handler

import (
	"context"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/poll"

	"go.uber.org/zap"
)

// Handler handles HTTP requests for the agent service
type Handler struct {
	useCase *usecase.UseCase
	logger  *logger.CanonicalLogger
	cfg     *config.AgentConfig
	poller  poll.Poller
}

// NewHandler creates a new agent handler
func NewHandler(d deps.App, config *config.AgentConfig) *Handler {

	// create central repository and clients
	// Pass in the pubsub subscriber (may be nil) so repository can start Redis listener if available.
	repo := repository.NewRepository(config.ControllerURL, config.WorkerURL, "", "", d.Pub)
	controllerRepo := repository.NewControllerClient(config, d.Logger)
	workerClient := repository.NewWorkerClient(config, d.Logger)

	uc := usecase.NewUseCase(controllerRepo, repo, workerClient, config, d.Logger)
	h := &Handler{
		useCase: uc,
		logger:  d.Logger,
		cfg:     config,
		poller:  d.Poller,
	}

	// registration is performed at startup; do not register periodic register task here

	return h
}

func (h *Handler) RegisterAgent(ctx context.Context) (*models.RegistrationResponse, error) {
	startTime := time.Now().UTC().Format(time.RFC3339)
	return h.useCase.RegisterWithController(ctx, h.cfg.Hostname, startTime)
}

// StartBackgroundServices starts background listeners and pollers for the agent
func (h *Handler) StartBackgroundServices(ctx context.Context) error {
	hbInterval := h.cfg.Heartbeat.Interval
	fbInterval := h.cfg.FallbackPoll.Interval
	return h.useCase.StartBackgroundServices(ctx, hbInterval, fbInterval)
}

// GetConfigure is a poller fetch function that fetches configuration from the controller
// using the usecase and returns an error on failure.
func (h *Handler) GetConfigure(ctx context.Context, log *logger.CanonicalLogger) error {
	cfg, pollInterval, notModified, err := h.useCase.FetchConfiguration(ctx)
	if err != nil {
		return err
	}
	if notModified {
		return nil
	}

	// If controller provided a new poll interval, and it's different, update poller
	if pollInterval != nil {
		_, currentInterval, _ := h.useCase.GetPollInfo()
		if *pollInterval > 0 && *pollInterval != currentInterval {
			agentID, _ := h.useCase.GetAgentID()

			// log intent to update with both old and new values
			h.logger.Info("updating poller interval",
				logger.Int("old_interval", currentInterval),
				logger.Int("new_interval", *pollInterval),
				logger.String("agent_id", agentID),
			)

			// update repository stored interval
			h.useCase.SetStoredPollInterval(*pollInterval)
			// update poller runtime interval
			if err := h.poller.UpdateInterval("get-configure", *pollInterval); err != nil {
				h.logger.WithError(err).Error("failed to update poller interval",
					logger.Int("new_interval", *pollInterval),
					logger.String("agent_id", agentID),
				)
			} else {
				h.logger.Info("updated poller interval",
					logger.Int("new_interval", *pollInterval),
					logger.Int("old_interval", currentInterval),
					logger.String("agent_id", agentID),
				)
			}
		}
	}

	if cfg != nil {
		log.Info("configuration applied", zap.String("etag", cfg.ETag))
	}
	return nil
}
