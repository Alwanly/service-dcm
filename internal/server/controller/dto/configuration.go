package dto

// SetConfigAgentRequest represents the request to set worker configuration
type SetConfigAgentRequest struct {
	URl   string `json:"url" example:"http://example.com/api" validate:"required,url"`
	Proxy string `json:"proxy" example:"http://proxy.example.com:8080" validate:"omitempty,url"`
}

// SetConfigAgentResponse represents the response after setting configuration
type SetConfigAgentResponse struct {
	Success bool `json:"success" example:"true"`
}

// GetConfigAgentRequest represents the request to get worker configuration
type GetConfigAgentRequest struct {
	ETag string `json:"etag" example:"1"`
}

// GetConfigAgentResponse represents the response when retrieving configuration
type GetConfigAgentResponse struct {
	ID     int64       `json:"id" example:"config-123"`
	ETag   string      `json:"etag" example:"1"`
	Config interface{} `json:"config" swaggertype:"object"`
}
