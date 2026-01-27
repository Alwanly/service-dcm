package dto

type SetConfigAgentRequest struct {
	Config interface{} `json:"config"`
}

type SetConfigAgentResponse struct {
	Success bool `json:"success"`
}

type GetConfigAgentResponse struct {
	Config interface{} `json:"config"`
}
