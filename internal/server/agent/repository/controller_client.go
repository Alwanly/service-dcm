package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/dto"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
)

type controllerClient struct {
	httpClient    *http.Client
	baseURL       string
	username      string
	password      string
	logger        *logger.CanonicalLogger
	currentConfig *StoreData
	mutex         sync.Mutex
}

// NewControllerClient creates a new controller client repository
func NewControllerClient(cfg *config.AgentConfig, log *logger.CanonicalLogger) IControllerClient {
	return &controllerClient{
		httpClient: &http.Client{Timeout: cfg.RequestTimeout},
		baseURL:    cfg.ControllerURL,
		username:   cfg.AgentUsername,
		password:   cfg.AgentPassword,
		logger:     log,
	}
}

func (c *controllerClient) Register(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error) {
	reqBody := map[string]string{
		"hostname":   hostname,
		"version":    version,
		"start_time": startTime,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal registration request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/register", c.baseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.username, c.password)

	// Set GetBody for potential retries
	buf := body
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, string(b))
	}

	var regResp models.RegistrationResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return nil, fmt.Errorf("failed to decode registration response: %w", err)
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.currentConfig == nil {
		c.currentConfig = &StoreData{}
	}
	c.currentConfig.AgentID = regResp.AgentID
	c.currentConfig.PollURL = regResp.PollURL
	c.currentConfig.PollInterval = regResp.PollIntervalSeconds
	c.currentConfig.APIToken = regResp.APIToken

	return &regResp, nil
}

// GetConfiguration fetches configuration from the controller or from a provided pollURL.
// It supports conditional requests via If-None-Match and returns the new ETag when present.
func (c *controllerClient) GetConfiguration(ctx context.Context, agentID, pollURL, ifNoneMatch string) (*models.Configuration, string, *int, bool, error) {
	// determine URL to call

	target := fmt.Sprintf("%s%s", c.baseURL, c.currentConfig.PollURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", nil, false, fmt.Errorf("failed to create get configuration request: %w", err)
	}

	if agentID != "" {
		req.Header.Set("X-Agent-ID", agentID)
	}
	if ifNoneMatch != "" {
		req.Header.Set("If-None-Match", ifNoneMatch)
	}

	// bearer token auth if available
	c.mutex.Lock()
	token := ""
	if c.currentConfig != nil {
		token = c.currentConfig.APIToken
	}
	c.mutex.Unlock()
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", nil, false, fmt.Errorf("get configuration request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return nil, "", nil, true, nil
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, "", nil, false, fmt.Errorf("get configuration returned status %d: %s", resp.StatusCode, string(b))
	}

	var respBody dto.ConfigurationResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, "", nil, false, fmt.Errorf("failed to decode configuration: %w", err)
	}
	cfg := models.Configuration{
		ID:         respBody.ID,
		ETag:       respBody.ETag,
		ConfigData: "",
	}
	configDataBytes, err := json.Marshal(respBody.Config)
	if err != nil {
		return nil, "", nil, false, fmt.Errorf("failed to marshal configuration data: %w", err)
	}
	cfg.ConfigData = string(configDataBytes)

	// Optionally store agentID in local store if provided
	if agentID != "" {
		c.mutex.Lock()
		if c.currentConfig == nil {
			c.currentConfig = &StoreData{}
		}
		c.currentConfig.AgentID = agentID
		c.mutex.Unlock()
	}

	return &cfg, cfg.ETag, respBody.PollIntervalSeconds, false, nil
}
