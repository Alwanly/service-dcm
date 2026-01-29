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
)

// Handler handles HTTP requests for the agent service
type Handler struct {
	useCase *usecase.UseCase
	logger  *logger.CanonicalLogger
	cfg     *config.AgentConfig
}

// NewHandler creates a new agent handler
func NewHandler(d deps.App, config *config.AgentConfig) *Handler {

	// create central repository and clients
	repo := repository.NewRepository()
	controllerRepo := repository.NewControllerClient(config, d.Logger)
	workerClient := repository.NewWorkerClient(config, d.Logger)

	uc := usecase.NewUseCase(controllerRepo, repo, workerClient, config)
	h := &Handler{
		useCase: uc,
		logger:  d.Logger,
		cfg:     config,
	}

	// registration is performed at startup; do not register periodic register task here

	return h
}

func (h *Handler) RegisterAgent(ctx context.Context) (*models.RegistrationResponse, error) {
	startTime := time.Now().UTC().Format(time.RFC3339)
	return h.useCase.RegisterWithController(ctx, h.cfg.Hostname, startTime)
}

// GetConfigure is a poller fetch function that fetches configuration from the controller
// using the usecase and returns an error on failure.
func (h *Handler) GetConfigure(ctx context.Context) error {
	cfg, notModified, err := h.useCase.FetchConfiguration(ctx)
	if err != nil {
		h.logger.WithError(err).Error("fetch configuration error")
		return err
	}
	if notModified {
		h.logger.Info("configuration not modified")
		return nil
	}
	if cfg != nil {
		h.logger.WithConfigVersion(cfg.ID).Info("configuration updated")
	}
	return nil
}
