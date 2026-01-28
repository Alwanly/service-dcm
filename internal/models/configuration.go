package models

import "time"

type Configuration struct {
	ID         int64     `gorm:"primaryKey;autoIncrement;column:id"`
	ETag       string    `gorm:"column:etag"`
	ConfigData string    `gorm:"column:config_data"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Configuration) TableName() string {
	return "configurations"
}

type ConfigData struct {
	URL   string `json:"url"`
	Proxy string `json:"proxy"`
}

// WorkerConfiguration represents the configuration sent to a worker instance
type WorkerConfiguration struct {
	Version   int64             `json:"version"`
	TargetURL string            `json:"target_url"`
	Headers   map[string]string `json:"headers,omitempty"`
	UpdatedAt time.Time         `json:"updated_at"`
}
