package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/dto"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/retry"
)

type ControllerClient struct {
	baseURL     string
	username    string
	password    string
	httpClient  *http.Client
	log         *logger.CanonicalLogger
	retryConfig retry.Config
}

func NewControllerClient(baseURL, username, password string, timeout time.Duration, log *logger.CanonicalLogger, retryConfig retry.Config) *ControllerClient {
	return &ControllerClient{
		baseURL:     baseURL,
		username:    username,
		password:    password,
		httpClient:  &http.Client{Timeout: timeout},
		log:         log,
		retryConfig: retryConfig,
	}
}

func (c *ControllerClient) Register(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error) {
	var result *models.RegistrationResponse
	var attempts int

	operation := func(ctx context.Context) error {
		attempts++

		resp, err := c.attemptRegistration(ctx, hostname, version, startTime)
		if err != nil {
			c.log.Info("registration attempt failed",
				logger.Int("attempt", attempts),
				logger.Int("max_retries", c.retryConfig.MaxRetries),
				logger.String("error", err.Error()),
			)
			return err
		}

		result = resp
		return nil
	}

	err := retry.WithExponentialBackoff(ctx, c.retryConfig, operation)
	if err != nil {
		c.log.WithError(err).Error("registration failed after all retries",
			logger.Int("total_attempts", attempts),
		)
		return nil, err
	}

	if attempts > 1 {
		c.log.Info("registration successful after retries",
			logger.String("agent_id", result.AgentID),
			logger.Int("attempts", attempts),
		)
	} else {
		c.log.Info("registration successful",
			logger.String("agent_id", result.AgentID),
		)
	}

	return result, nil
}

func (c *ControllerClient) attemptRegistration(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error) {
	reqData := dto.RegisterAgentRequest{
		Hostname:  hostname,
		Version:   version,
		StartTime: startTime,
	}

	body, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/register", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)

	// Set GetBody for retry support
	buf := body
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), nil
	}

	c.log.Debug("sending registration request",
		logger.String("url", c.baseURL+"/register"),
		logger.String("hostname", hostname),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response dto.RegisterAgentResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &models.RegistrationResponse{
		AgentID:             response.AgentID,
		PollURL:             response.PollURL,
		PollIntervalSeconds: response.PollIntervalSeconds,
	}, nil
}

func (c *ControllerClient) GetConfiguration(ctx context.Context, agentID, etag string) (*models.WorkerConfiguration, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/config", nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("X-Agent-ID", agentID)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	c.log.Debug("fetching configuration",
		logger.String("agent_id", agentID),
		logger.String("etag", etag),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		c.log.Debug("configuration not modified")
		return nil, etag, nil
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("fetch configuration failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var config models.WorkerConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, "", fmt.Errorf("failed to decode configuration: %w", err)
	}

	newETag := resp.Header.Get("ETag")

	c.log.Info("received new configuration",
		logger.String("agent_id", agentID),
		logger.Int64("version", config.Version),
		logger.String("etag", newETag),
	)

	return &config, newETag, nil
}
