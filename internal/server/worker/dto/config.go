package dto

import "time"

// ReceiveConfigRequest represents the request body for receiving worker configuration
type ReceiveConfigRequest struct {
	Version   int64             `json:"version" validate:"required"`
	TargetURL string            `json:"target_url" validate:"required,url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timeout   int               `json:"timeout_seconds,omitempty"`
}

// ReceiveConfigResponse represents the response after updating worker configuration
type ReceiveConfigResponse struct {
	Success   bool      `json:"success"`
	Message   string    `json:"message"`
	Version   int64     `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
