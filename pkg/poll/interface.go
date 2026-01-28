package poll

import (
	"context"
)

// ConfigUpdateMessage represents a configuration update notification
type ConfigUpdateMessage struct {
	Config interface{}
	ETag   string
}

type PollerConfig struct {
	PollIntervalSeconds int
}

type MetaFunc struct {
	FetchFunc
	PollerConfig
}

// Poller defines the interface for configuration polling
type Poller interface {
	// Start begins polling and sends updates to the returned channel
	Start(ctx context.Context) error
	// Stop gracefully stops the poller
	Stop() error
	// RegisterFetchFunc and config retrieval function
	RegisterFetchFunc(name string, fetchFunc FetchFunc, config PollerConfig)
}

// FetchFunc is a function that fetches the latest configuration
// Returns the config, current ETag, and any error
type FetchFunc func(ctx context.Context) error
