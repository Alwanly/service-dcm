package poll

import "time"

// Config holds configuration for the poller
type Config struct {
	// Interval between poll attempts
	Interval time.Duration
	// InitialETag is the starting ETag value (empty string for first poll)
	InitialETag string
}

// DefaultConfig returns a Config with sensible defaults
func DefaultConfig() Config {
	return Config{
		Interval:    5 * time.Second,
		InitialETag: "",
	}
}
