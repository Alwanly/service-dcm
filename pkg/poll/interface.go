package poll

import (
	"context"
)

// ConfigUpdateMessage represents a configuration update notification
type ConfigUpdateMessage struct {
	Config interface{}
	ETag   string
}

// Poller defines the interface for configuration polling
type Poller interface {
	// Start begins polling and sends updates to the returned channel
	Start(ctx context.Context) (<-chan ConfigUpdateMessage, error)
	// Stop gracefully stops the poller
	Stop() error
}

// FetchFunc is a function that fetches the latest configuration
// Returns the config, current ETag, and any error
type FetchFunc func(ctx context.Context, currentETag string) (config interface{}, newETag string, err error)
