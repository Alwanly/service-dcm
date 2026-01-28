package repository

import (
	"context"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

// IControllerClient defines the interface for communicating with the controller service
type IControllerClient interface {
	// Register registers the agent with the controller
	Register(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error)
	// GetConfiguration retrieves the latest worker configuration and current ETag
	GetConfiguration(ctx context.Context, agentID string, currentETag string) (*models.WorkerConfiguration, string, error)
}

// IWorkerClient defines the interface for communicating with the worker service
type IWorkerClient interface {
	// SendConfiguration sends the configuration to the worker
	SendConfiguration(ctx context.Context, config *models.WorkerConfiguration) error
}
