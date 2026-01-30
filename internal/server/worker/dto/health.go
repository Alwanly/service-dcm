package dto

import "time"

type HealthCheckResponse struct {
	Status      string            `json:"status" example:"healthy"`
	Configured  bool              `json:"configured" example:"true"`
	Version     int64             `json:"version,omitempty" example:"2"`
	TargetURL   string            `json:"target_url,omitempty" example:"https://webhook.site/unique-id"`
	Headers     map[string]string `json:"headers,omitempty" example:"{\"Authorization\":\"Bearer token123\"}"`
	LastUpdated time.Time         `json:"last_updated,omitempty" example:"2026-01-27T12:30:45Z"`
}
