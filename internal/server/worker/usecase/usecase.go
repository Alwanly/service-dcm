package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Alwanly/service-distribute-management/internal/models"
	dto "github.com/Alwanly/service-distribute-management/internal/server/worker/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/repository"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
	"go.uber.org/zap"
)

// UseCaseInterface defines the business logic interface for worker operations
type UseCaseInterface interface {
	ReceiveConfig(ctx context.Context, req *dto.ReceiveConfigRequest) wrapper.JSONResult
	HitRequest(ctx context.Context) wrapper.JSONResult
	GetCurrentConfig() *models.ConfigData
}

// UseCase implements the business logic for worker operations
type UseCase struct {
	repo       repository.IRepository
	httpClient *http.Client
}

// NewUseCase creates a new UseCase instance
func NewUseCase(repo repository.IRepository, timeout time.Duration) UseCaseInterface {
	return &UseCase{
		repo: repo,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// ReceiveConfig handles configuration updates from the agent
func (uc *UseCase) ReceiveConfig(ctx context.Context, req *dto.ReceiveConfigRequest) wrapper.JSONResult {

	configData, err := json.Marshal(req.ConfigData)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err))
		return wrapper.ResponseSuccess(http.StatusConflict, "Failed validate configData")
	}

	// Create worker configuration model
	config := &models.Configuration{
		ID:         req.ID,
		ETag:       req.ETag,
		ConfigData: string(configData),
	}

	// Update configuration in repository
	if err := uc.repo.UpdateConfig(config); err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.JSONResult{
			Code:    fiber.StatusInternalServerError,
			Success: false,
			Message: "Failed to update configuration",
			Data:    nil,
		}
	}

	logger.AddToContext(ctx,
		zap.Bool(logger.FieldSuccess, true),
		zap.String(logger.FieldETag, req.ETag),
	)

	return wrapper.ResponseSuccess(http.StatusOK, nil)
}

// ProxyRequest forwards a request to the configured target URL
func (uc *UseCase) HitRequest(ctx context.Context) wrapper.JSONResult {
	// Get current configuration
	data, err := uc.repo.GetCurrentConfig()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get configuration", nil)
	}

	if data == nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, false), zap.String(logger.FieldProxyStatus, "no_config"))
		return wrapper.ResponseFailed(http.StatusBadRequest, "no configuration available", nil)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, data.Config.URL, nil)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to create request", nil)
	}
	// Set proxy if configured
	if data.Config.Proxy != "" {
		proxyURL, err := http.ProxyFromEnvironment(req)
		if err != nil {
			logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
			return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to set proxy", nil)
		}
		req.URL = proxyURL
	}
	// Perform HTTP request
	resp, err := uc.httpClient.Do(req)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to perform request", nil)
	}
	defer resp.Body.Close()
	logger.AddToContext(ctx,
		zap.Bool(logger.FieldSuccess, true),
		zap.String(logger.FieldTargetURL, data.Config.URL),
	)

	response := &dto.HitResponse{
		ETag: data.ETag,
		URL:  data.Config.URL,
		Data: resp.Body,
	}
	return wrapper.ResponseSuccess(http.StatusOK, response)
}

// GetCurrentConfig returns the current configuration data (if any)
func (uc *UseCase) GetCurrentConfig() *models.ConfigData {
	data, err := uc.repo.GetCurrentConfig()
	if err != nil || data == nil {
		return nil
	}
	return &data.Config
}
