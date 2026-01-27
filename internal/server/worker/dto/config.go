package dto

import "time"

// ReceiveConfigRequest represents the request body for receiving worker configuration
type ReceiveConfigRequest struct {
	Version   int64             `json:"version" validate:"required" example:"2"`
	TargetURL string            `json:"target_url" validate:"required,url" example:"https://webhook.site/unique-id"`
	Headers   map[string]string `json:"headers,omitempty" example:"{\"Authorization\":\"Bearer token123\",\"X-Custom-Header\":\"value\"}"`
	Timeout   int               `json:"timeout_seconds,omitempty" example:"15"`
}

// ReceiveConfigResponse represents the response after updating worker configuration
type ReceiveConfigResponse struct {
	Success   bool      `json:"success" example:"true"`
	Message   string    `json:"message" example:"Configuration updated successfully"`
	Version   int64     `json:"version" example:"2"`
	UpdatedAt time.Time `json:"updated_at" example:"2026-01-27T12:30:45Z"`
}
