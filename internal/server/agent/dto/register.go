package dto

// RegisterResponse represents the response when an agent registers
type RegisterResponse struct {
	AgentID             string `json:"agent_id"`
	PollURL             string `json:"poll_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	Message             string `json:"message,omitempty"`
}
