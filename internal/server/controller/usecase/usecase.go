package usecase

import (
	"context"
	"net/http"

	"github.com/Alwanly/service-distribute-management/internal/server/controller/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/repository"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
)

type UseCase struct {
	Repository *repository.Repository
}

type UseCaseInterface interface {
	RegisterAgent(ctx context.Context, req *dto.RegisterAgentRequest) wrapper.JSONResult
	SetConfigAgent(ctx context.Context, req *dto.SetConfigAgentRequest) wrapper.JSONResult
	GetConfigAgent(ctx context.Context) wrapper.JSONResult
}

func NewUseCase(repo *repository.Repository) *UseCase {
	return &UseCase{
		Repository: repo,
	}
}

func (uc *UseCase) RegisterAgent(ctx context.Context, req *dto.RegisterAgentRequest) wrapper.JSONResult {

	return wrapper.ResponseSuccess(http.StatusOK, dto.RegisterAgentResponse{})
}

func (uc *UseCase) SetConfigAgent(ctx context.Context, req *dto.SetConfigAgentRequest) wrapper.JSONResult {
	// Implement the logic to set configuration for an agent
	return wrapper.ResponseSuccess(http.StatusOK, dto.RegisterAgentResponse{})
}

func (uc *UseCase) GetConfigAgent(ctx context.Context) wrapper.JSONResult {
	// Implement the logic to get configuration for an agent
	return wrapper.ResponseSuccess(http.StatusOK, dto.RegisterAgentResponse{})
}
