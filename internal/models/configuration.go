package models

import "time"

type Configuration struct {
	ID       string
	ETag     string
	Metadata string
}

// WorkerConfiguration represents the configuration for a worker instance
type WorkerConfiguration struct {
	Version   int64             `json:"version"`
	TargetURL string            `json:"target_url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timeout   int               `json:"timeout_seconds,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ErrorResponse represents an error response from the API
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// RegistrationResponse represents the response when an agent registers with the controller
type RegistrationResponse struct {
	AgentID             string `json:"agent_id"`
	PollURL             string `json:"poll_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
}
