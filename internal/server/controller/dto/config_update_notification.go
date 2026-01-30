package dto

type ConfigUpdateNotification struct {
	AgentID string `json:"agent_id"`
	ETag    string `json:"etag"`
}
