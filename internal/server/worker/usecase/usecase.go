package usecase

import (
	"bytes"
	"context"
	"fmt"
	"io"
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
	GetHealthStatus(ctx context.Context) wrapper.JSONResult
	ProxyRequest(ctx context.Context, body []byte, headers map[string][]string) ([]byte, int, error)
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
	// Create worker configuration model
	config := &models.WorkerConfiguration{
		Version:   req.Version,
		TargetURL: req.TargetURL,
		Headers:   req.Headers,
		Timeout:   req.Timeout,
		UpdatedAt: time.Now(),
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

	// Create response
	response := &dto.ReceiveConfigResponse{
		Success:   true,
		Message:   "Configuration updated successfully",
		Version:   config.Version,
		UpdatedAt: config.UpdatedAt,
	}

	logger.AddToContext(ctx,
		zap.Int64(logger.FieldConfigVersion, config.Version),
		zap.String(logger.FieldTargetURL, config.TargetURL),
		zap.Bool(logger.FieldSuccess, true),
	)

	return wrapper.JSONResult{
		Code:    fiber.StatusOK,
		Success: true,
		Message: "Configuration updated successfully",
		Data:    response,
	}
}

// GetHealthStatus returns the current health and configuration status
func (uc *UseCase) GetHealthStatus(ctx context.Context) wrapper.JSONResult {
	config, err := uc.repo.GetCurrentConfig()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.JSONResult{
			Code:    fiber.StatusInternalServerError,
			Success: false,
			Message: "Failed to retrieve configuration",
			Data:    nil,
		}
	}

	response := &dto.HealthCheckResponse{
		Status:     "healthy",
		Configured: config != nil,
	}

	if config != nil {
		response.Version = config.Version
		response.TargetURL = config.TargetURL
		response.Headers = config.Headers
		response.LastUpdated = config.UpdatedAt
		logger.AddToContext(ctx,
			zap.Bool("configured", true),
			zap.Int64(logger.FieldConfigVersion, config.Version),
		)
	} else {
		logger.AddToContext(ctx, zap.Bool("configured", false))
	}

	return wrapper.JSONResult{
		Code:    fiber.StatusOK,
		Success: true,
		Message: "Worker is healthy",
		Data:    response,
	}
}

// ProxyRequest forwards a request to the configured target URL
func (uc *UseCase) ProxyRequest(ctx context.Context, body []byte, headers map[string][]string) ([]byte, int, error) {
	// Get current configuration
	config, err := uc.repo.GetCurrentConfig()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return nil, fiber.StatusInternalServerError, fmt.Errorf("failed to get configuration: %w", err)
	}

	if config == nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, false), zap.String(logger.FieldProxyStatus, "no_config"))
		return nil, fiber.StatusServiceUnavailable, fmt.Errorf("worker not configured")
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, config.TargetURL, bytes.NewReader(body))
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false), zap.String(logger.FieldProxyStatus, "create_request_failed"))
		return nil, fiber.StatusInternalServerError, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers from original request
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Add configured headers (these override original headers if there's a conflict)
	if config.Headers != nil {
		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}
	}

	// Execute request
	resp, err := uc.httpClient.Do(req)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false), zap.String(logger.FieldProxyStatus, "request_failed"), zap.String(logger.FieldTargetURL, config.TargetURL), zap.Int64(logger.FieldConfigVersion, config.Version))
		return nil, fiber.StatusBadGateway, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false), zap.String(logger.FieldProxyStatus, "read_response_failed"), zap.String(logger.FieldTargetURL, config.TargetURL), zap.Int64(logger.FieldConfigVersion, config.Version))
		return nil, fiber.StatusBadGateway, fmt.Errorf("failed to read response: %w", err)
	}

	// Success - add contextual fields
	logger.AddToContext(ctx,
		zap.String(logger.FieldTargetURL, config.TargetURL),
		zap.Int64(logger.FieldConfigVersion, config.Version),
		zap.String(logger.FieldProxyStatus, resp.Status),
		zap.Bool(logger.FieldSuccess, true),
	)

	return respBody, resp.StatusCode, nil
}
