package dto

// SetConfigAgentRequest represents the request to set worker configuration
type SetConfigAgentRequest struct {
	Config interface{} `json:"config" swaggertype:"object"`
}

// SetConfigAgentResponse represents the response after setting configuration
type SetConfigAgentResponse struct {
	Success bool `json:"success" example:"true"`
}

// GetConfigAgentResponse represents the response when retrieving configuration
type GetConfigAgentResponse struct {
	Config interface{} `json:"config" swaggertype:"object"`
}
