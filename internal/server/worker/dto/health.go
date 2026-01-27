package dto

import "time"

// HealthCheckResponse represents the response from the health check endpoint
type HealthCheckResponse struct {
	Status      string            `json:"status"`
	Configured  bool              `json:"configured"`
	Version     int64             `json:"version,omitempty"`
	TargetURL   string            `json:"target_url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	LastUpdated time.Time         `json:"last_updated,omitempty"`
}
