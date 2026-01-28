package models

import "time"

type Agent struct {
	AgentID   string    `gorm:"primaryKey;column:agent_id"`
	Status    string    `gorm:"column:status"`
	LastSeen  time.Time `gorm:"column:last_seen"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Agent) TableName() string {
	return "agents"
}
