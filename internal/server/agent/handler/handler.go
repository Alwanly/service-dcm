package handler

import (
	"context"
	"errors"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/poll"
)

// Handler handles HTTP requests for the agent service
type Handler struct {
	useCase *usecase.UseCase
	logger  *logger.CanonicalLogger
}

// NewHandler creates a new agent handler
func NewHandler(d deps.App, config *config.AgentConfig) *Handler {

	controllerRepo := repository.NewControllerClient(config, d.Logger)

	uc := usecase.NewUseCase(controllerRepo)
	h := &Handler{
		useCase: uc,
		logger:  d.Logger,
	}

	d.Poller.RegisterFetchFunc("register-agent", h.RegisterAgent, poll.PollerConfig{
		PollIntervalSeconds: 50,
	})
	return h
}

func (h *Handler) RegisterAgent(ctx context.Context) error {
	res := h.useCase.RegisterWithController(ctx)

	if !res.Success {
		return errors.New(res.Message)
	}
	return nil
}
