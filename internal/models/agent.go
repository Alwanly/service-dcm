package models

import "time"

// Legacy Agent model (kept for existing controller logic)
type Agent struct {
	AgentID           string     `gorm:"primaryKey;column:agent_id" json:"agent_id"`
	Status            string     `gorm:"column:status" json:"status"`
	LastSeen          time.Time  `gorm:"column:last_seen" json:"last_seen"`
	LastHeartbeat     *time.Time `gorm:"index" json:"last_heartbeat"`
	LastConfigVersion string     `gorm:"column:last_config_version" json:"last_config_version"`
	CreatedAt         time.Time  `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (Agent) TableName() string {
	return "agents"
}

// RegistrationResponse represents the response when an agent successfully registers with the controller
type RegistrationResponse struct {
	AgentID             string `json:"agent_id"`
	PollURL             string `json:"poll_url,omitempty"`
	PollIntervalSeconds int    `json:"poll_interval_seconds"`
	APIToken            string `json:"api_token,omitempty"`
}

// New AgentConfig model for per-agent authentication and configuration
type AgentConfig struct {
	ID                  string    `gorm:"column:id;primaryKey" json:"id"`
	AgentName           string    `gorm:"column:agent_name;not null" json:"agent_name"`
	APIToken            string    `gorm:"column:api_token;not null;uniqueIndex" json:"-"` // Never expose in JSON
	PollIntervalSeconds *int      `gorm:"column:poll_interval_seconds" json:"poll_interval_seconds,omitempty"`
	CreatedAt           time.Time `gorm:"column:created_at;not null;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"column:updated_at;not null;autoUpdateTime" json:"updated_at"`
}

// TableName specifies the table name for GORM
func (AgentConfig) TableName() string {
	return "agent_configs"
}

// AgentPublic is the public-facing agent model without sensitive fields
type AgentPublic struct {
	ID                  string    `json:"id"`
	AgentName           string    `json:"agent_name"`
	PollIntervalSeconds *int      `json:"poll_interval_seconds,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ToPublic converts AgentConfig to AgentPublic (excludes APIToken)
func (a *AgentConfig) ToPublic() AgentPublic {
	return AgentPublic{
		ID:                  a.ID,
		AgentName:           a.AgentName,
		PollIntervalSeconds: a.PollIntervalSeconds,
		CreatedAt:           a.CreatedAt,
		UpdatedAt:           a.UpdatedAt,
	}
}
