package dto

// ConfigUpdateNotification represents a message published to Redis when config changes
type ConfigUpdateNotification struct {
	AgentID string `json:"agent_id"`
	ETag    string `json:"etag"`
}
