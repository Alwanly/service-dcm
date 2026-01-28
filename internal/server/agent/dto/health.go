package dto

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	AgentID   string `json:"agent_id,omitempty"`
	Timestamp string `json:"timestamp"`
}
