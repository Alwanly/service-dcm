package dto

import "github.com/Alwanly/service-distribute-management/internal/models"

type SendConfigRequest struct {
	ID         int64             `json:"id" example:"1"`
	ETag       string            `json:"etag" example:"v1.0.0"`
	ConfigData models.ConfigData `json:"config_data"`
}
