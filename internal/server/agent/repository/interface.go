package repository

import (
	"context"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

// IControllerClient defines the interface for communicating with the controller service
type IControllerClient interface {
	// Register registers the agent with the controller
	Register(ctx context.Context, hostname, version, startTime string) (*models.RegistrationResponse, error)
}

// IWorkerClient defines the interface for communicating with the worker service
type IWorkerClient interface {
	// SendConfiguration sends the configuration to the worker
	SendConfiguration(ctx context.Context, config *models.Configuration) error
}

type IRepository interface {
	// SetAgentID sets the agent ID
	SetAgentID(agentID string) error
	// GetCurrentConfig retrieves the current worker configuration
	GetCurrentConfig() (*models.Configuration, error)
	// UpdateConfig updates the worker configuration
	UpdateConfig(config *models.Configuration) error
}
