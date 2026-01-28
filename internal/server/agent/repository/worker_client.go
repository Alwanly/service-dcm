package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
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
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
