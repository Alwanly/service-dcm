package agent

import (
	"context"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

// ControllerClient is a minimal HTTP client for the controller (stub)
type ControllerClient struct {
	baseURL  string
	username string
	password string
	timeout  time.Duration
}

// NewControllerClient constructs a new client
func NewControllerClient(baseURL, username, password string, timeout time.Duration) *ControllerClient {
	return &ControllerClient{baseURL: baseURL, username: username, password: password, timeout: timeout}
}

// Register registers the agent with the controller. Returns a minimal RegistrationResponse.
func (c *ControllerClient) Register(ctx context.Context, hostname string) (*models.RegistrationResponse, error) {
	// Stubbed response
	return &models.RegistrationResponse{AgentID: "agent-local", PollIntervalSeconds: 5}, nil
}

// GetConfiguration fetches configuration from controller. Returns nil if no config.
func (c *ControllerClient) GetConfiguration(ctx context.Context, currentETag string) (*models.WorkerConfiguration, string, error) {
	// Stubbed: no configuration available
	return nil, "", nil
}
