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

	return &regResp, nil
}
