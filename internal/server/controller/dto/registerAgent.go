package dto

type RegisterAgentRequest struct {
	Hostname  string `json:"hostname" validate:"required"`
	StartTime string `json:"start_time" validate:"required"`
}

type RegisterAgentResponse struct {
	AgentID             string `json:"agent_id"`              // UUID
	AgentName           string `json:"agent_name"`            // Hostname
	APIToken            string `json:"api_token"`             // Bearer token for authentication
	PollURL             string `json:"poll_url"`              // Endpoint to poll for configuration
	PollIntervalSeconds int    `json:"poll_interval_seconds"` // Polling interval
}
