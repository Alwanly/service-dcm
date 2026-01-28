package dto

import "time"

// ConfigResponse represents a configuration fetched from the controller
type ConfigResponse struct {
	Version   int64             `json:"version"`
	TargetURL string            `json:"target_url"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timeout   int               `json:"timeout_seconds,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
	ETag      string            `json:"etag"`
}
