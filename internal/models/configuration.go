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
