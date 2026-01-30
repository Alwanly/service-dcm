package dto

import "github.com/Alwanly/service-distribute-management/internal/models"

// UpdatePollIntervalRequest updates an agent's polling interval
type UpdatePollIntervalRequest struct {
	PollIntervalSeconds *int `json:"poll_interval_seconds"`
}

// RotateTokenResponse returns the new token after rotation
type RotateTokenResponse struct {
	AgentID  string `json:"agent_id"`
	APIToken string `json:"api_token"`
	Message  string `json:"message"`
}

// ListAgentsResponse returns all registered agents
type ListAgentsResponse struct {
	Agents []models.AgentPublic `json:"agents"`
	Total  int                  `json:"total"`
}
