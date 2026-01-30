package dto

type ConfigurationRequest struct {
	ID   string `json:"id"`
	Etag string `json:"etag"`
}

type ConfigurationResponse struct {
	ID                  int64       `json:"id" example:"config-123"`
	ETag                string      `json:"etag" example:"1"`
	Config              interface{} `json:"config"`
	PollIntervalSeconds *int        `json:"poll_interval_seconds,omitempty"` // Optional: allows dynamic updates
}
