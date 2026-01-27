package dto

type RegisterAgentRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
}

type RegisterAgentResponse struct {
	AgentID             string `json:"agent_id"`
	PollURL             string `json:"poll_url"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
}
