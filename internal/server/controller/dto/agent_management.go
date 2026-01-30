package dto

import "github.com/Alwanly/service-distribute-management/internal/models"

type UpdatePollIntervalRequest struct {
	PollIntervalSeconds *int `json:"poll_interval_seconds"`
}

type RotateTokenResponse struct {
	AgentID  string `json:"agent_id"`
	APIToken string `json:"api_token"`
	Message  string `json:"message"`
}

type ListAgentsResponse struct {
	Agents []models.AgentPublic `json:"agents"`
	Total  int                  `json:"total"`
}
