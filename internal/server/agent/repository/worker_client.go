package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/dto"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/retry"
	"go.uber.org/zap"
)

type workerClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *logger.CanonicalLogger
}

// NewWorkerClient creates a new worker client repository
func NewWorkerClient(cfg *config.AgentConfig, log *logger.CanonicalLogger) IWorkerClient {
	return &workerClient{
		httpClient: &http.Client{Timeout: cfg.RequestTimeout},
		baseURL:    cfg.WorkerURL,
		logger:     log,
	}
}

// SendConfiguration sends the configuration to the worker
func (w *workerClient) SendConfiguration(ctx context.Context, config *models.Configuration) error {
	url := fmt.Sprintf("%s/config", w.baseURL)

	configData := new(models.ConfigData)
	if config.ConfigData == "" {
		return fmt.Errorf("config data is empty for configuration ID %s", config.ETag)
	}

	if err := json.Unmarshal([]byte(config.ConfigData), configData); err != nil {
		return fmt.Errorf("failed to unmarshal config data for ID %s: %w", config.ETag, err)
	}

	rawRequestBody := dto.SendConfigRequest{
		ID:         config.ID,
		ETag:       config.ETag,
		ConfigData: *configData,
	}
	requestBody, err := json.Marshal(rawRequestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	// propagate correlation id from context if present
	if corr := logger.GetCorrelationID(ctx); corr != "" {
		req.Header.Set("X-Correlation-ID", corr)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("worker returned status %d: %s", resp.StatusCode, string(b))
	}

	return nil
}

// SendConfigurationWithRetry sends configuration to worker with exponential backoff retry
func (w *workerClient) SendConfigurationWithRetry(ctx context.Context, config *models.Configuration, maxRetries int) error {
	// Use closure to track attempts for logging
	attempt := 0
	retryCfg := retry.Config{
		MaxRetries:     maxRetries,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
	}

	op := func(ctx context.Context) error {
		attempt++
		w.logger.Info("attempting to send configuration to worker", zap.Int("attempt", attempt), zap.String("etag", config.ETag))
		err := w.SendConfiguration(ctx, config)
		if err != nil {
			w.logger.WithError(err).Error("failed to send configuration to worker", zap.Int("attempt", attempt), zap.String("etag", config.ETag))
		} else {
			w.logger.Info("configuration sent to worker", zap.Int("attempt", attempt), zap.String("etag", config.ETag))
		}
		return err
	}

	return retry.WithExponentialBackoff(ctx, retryCfg, op)
}
