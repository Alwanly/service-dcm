package models

type WorkerConfiguration struct {
	Version   int64             `json:"version"`
	TargetURL string            `json:"target_url"`
	Headers   map[string]string `json:"headers,omitempty"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type RegistrationResponse struct {
	AgentID             string `json:"agent_id"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
}
