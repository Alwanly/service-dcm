package repository

import (
	"encoding/json"
	"sync"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

type StorageData struct {
	Config models.ConfigData
	ETag   string
}
type IRepository interface {
	GetCurrentConfig() (*StorageData, error)
	UpdateConfig(config *models.Configuration) error
}

// Repository implements in-memory storage for worker configuration
type Repository struct {
	currentConfig *StorageData
	mutex         sync.RWMutex
}

// NewRepository creates a new repository instance
func NewRepository() IRepository {
	return &Repository{
		currentConfig: nil,
		mutex:         sync.RWMutex{},
	}
}

// GetCurrentConfig retrieves the current worker configuration
func (r *Repository) GetCurrentConfig() (*StorageData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.currentConfig, nil
}

// UpdateConfig updates the worker configuration
func (r *Repository) UpdateConfig(config *models.Configuration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var configData models.ConfigData
	err := json.Unmarshal([]byte(config.ConfigData), &configData)
	if err != nil {
		return err
	}

	r.currentConfig = &StorageData{
		Config: configData,
		ETag:   config.ETag,
	}

	return nil
}
