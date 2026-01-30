package repository

import (
	"context"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
)

// IControllerClient defines the interface for communicating with the controller service
type IControllerClient interface {
	// Register registers the agent with the controller
	Register(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error)
	// GetConfiguration fetches the configuration from the controller using the provided poll URL.
	// Returns: configuration, new ETag, optional poll interval (nil if not provided), notModified flag, error
	GetConfiguration(ctx context.Context, agentID, pollURL, ifNoneMatch string) (*models.Configuration, string, *int, bool, error)
}

// IWorkerClient defines the interface for communicating with the worker service
type IWorkerClient interface {
	// SendConfiguration sends the configuration to the worker
	SendConfiguration(ctx context.Context, config *models.Configuration) error
	// SendConfigurationWithRetry sends the configuration to the worker with retry/backoff
	SendConfigurationWithRetry(ctx context.Context, config *models.Configuration, maxRetries int) error
}

type IRepository interface {
	// SetAgentID sets the agent ID
	SetAgentID(agentID string) error
	// GetAgentID returns the currently stored agent ID
	GetAgentID() (string, error)
	// GetCurrentConfig retrieves the current worker configuration
	GetCurrentConfig() (*models.Configuration, error)
	// UpdateConfig updates the worker configuration
	UpdateConfig(config *models.Configuration) error
	// SetPollInfo sets the poll URL and interval
	SetPollInfo(pollURL string, pollInterval int) error
	// GetPollInfo retrieves the poll URL and interval
	GetPollInfo() (string, int, error)
	// SetAPIToken stores the API token for authentication
	SetAPIToken(token string)
	// GetAPIToken retrieves the stored API token
	GetAPIToken() string
	// UpdatePollInterval updates the stored polling interval
	UpdatePollInterval(newInterval int)
	// SetConfig stores configuration and ETag
	SetConfig(config *models.Configuration, etag string)
	// GetConfig retrieves stored configuration and ETag
	GetConfig() (*models.Configuration, string)
	// StartRedisListener starts a background Redis subscription listener
	StartRedisListener(ctx context.Context, logger *logger.CanonicalLogger) error
	// RegisterConfigPolling registers fallback polling mechanism for configuration
	RegisterConfigPolling(ctx context.Context, logger *logger.CanonicalLogger)
	// RegisterHeartbeatPolling starts periodic heartbeat to controller
	RegisterHeartbeatPolling(ctx context.Context, logger *logger.CanonicalLogger, interval time.Duration)
}
