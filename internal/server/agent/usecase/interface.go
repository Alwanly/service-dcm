package usecase

import (
	"context"
)

// IUseCase defines the business logic interface for the agent service
type IUseCase interface {
	// RegisterWithController registers the agent with the controller
	RegisterWithController(ctx context.Context) (agentID string, err error)
	// StartPolling starts the configuration polling process
	StartPolling(ctx context.Context, agentID string) error
	// StopPolling stops the configuration polling process
	StopPolling() error
	// GetAgentID returns the current agent ID
	GetAgentID() string
	// GetStatus returns the agent status information
	GetStatus() map[string]interface{}
}
