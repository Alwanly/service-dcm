package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"gorm.io/gorm"
)

type Repository struct {
	DB *gorm.DB
}

type IRepository interface {
	RegisterAgent(ctx context.Context, data *models.Agent) error
	UpdateConfig(ctx context.Context, config string) error
	GetConfigETag(ctx context.Context) (string, error)
	GetConfig(ctx context.Context, config string) (models.ConfigData, error)
	GetConfigIfChanged(currentETag string) (string, models.ConfigData, error)
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{DB: db}
}

func (r *Repository) RegisterAgent(ctx context.Context, data *models.Agent) error {
	result := r.DB.WithContext(ctx).Create(data)
	return result.Error
}

func generateETag(config string) string {
	// Simple ETag generation using length and timestamp
	return fmt.Sprintf("%x-%d", len(config), time.Now().UnixNano())
}

func (r *Repository) UpdateConfig(ctx context.Context, config string) error {
	etag := generateETag(config)
	result := r.DB.WithContext(ctx).Create(&models.Configuration{
		ETag:       etag,
		ConfigData: config,
	})

	return result.Error
}

func (r *Repository) GetConfigETag(ctx context.Context) (string, error) {
	var etag string
	err := r.DB.WithContext(ctx).Raw("SELECT etag FROM configurations ORDER BY created_at DESC LIMIT 1").Scan(&etag).Error
	if err == gorm.ErrRecordNotFound {
		return "", nil
	}
	return etag, err
}

func (r *Repository) GetConfig(ctx context.Context, config string) (*models.ConfigData, error) {
	var rawConfigData string
	var configData *models.ConfigData

	err := r.DB.WithContext(ctx).Raw("SELECT config_data FROM configurations WHERE etag = ? LIMIT 1", config).Scan(&rawConfigData).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	err = json.Unmarshal([]byte(rawConfigData), &configData)
	if err != nil {
		return nil, err
	}

	return configData, nil
}

func (r *Repository) GetConfigIfChanged(currentETag string) (string, models.ConfigData, error) {
	var etag string
	var rawConfigData string
	var configData models.ConfigData

	err := r.DB.Raw("SELECT etag, config_data FROM configurations ORDER BY created_at DESC LIMIT 1").Scan(&struct {
		ETag       *string
		ConfigData *string
	}{
		ETag:       &etag,
		ConfigData: &rawConfigData,
	}).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", models.ConfigData{}, nil
		}
		return "", models.ConfigData{}, err
	}

	if etag == currentETag {
		return "", models.ConfigData{}, nil
	}

	err = json.Unmarshal([]byte(rawConfigData), &configData)
	if err != nil {
		return "", models.ConfigData{}, err
	}

	return etag, configData, nil
}
