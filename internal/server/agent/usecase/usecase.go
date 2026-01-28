package usecase

import (
	"context"
	"net/http"

	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
)

type UseCase struct {
	controllerClient repository.IControllerClient
}

func NewUseCase(controllerClient repository.IControllerClient) *UseCase {
	return &UseCase{
		controllerClient: controllerClient,
	}
}

// RegisterWithController registers the agent with the controller
func (uc *UseCase) RegisterWithController(ctx context.Context) wrapper.JSONResult {
	_, err := uc.controllerClient.Register(ctx, "agent-hostname", "1.0.0", "2024-01-01T00:00:00Z")
	if err != nil {
		return wrapper.JSONResult{
			Success: false,
			Message: err.Error(),
		}
	}

	return wrapper.JSONResult{
		Success: true,
		Message: "Agent registered successfully",
	}
}

func (uc *UseCase) FetchConfiguration(ctx context.Context) wrapper.JSONResult {

	return wrapper.ResponseSuccess(http.StatusOK, nil)
}
