package dto

// RegisterAgentRequest represents the agent registration request
type RegisterAgentRequest struct {
}

// RegisterAgentResponse represents the agent registration response
type RegisterAgentResponse struct {
	AgentID             string `json:"agent_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	PollURL             string `json:"poll_url" example:"http://localhost:8080/config"`
	PollIntervalSeconds int    `json:"poll_interval_seconds" example:"5"`
}
