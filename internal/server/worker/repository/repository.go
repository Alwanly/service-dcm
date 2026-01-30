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
type Repository struct {
	currentConfig *StorageData
	mutex         sync.RWMutex
}

func NewRepository() IRepository {
	return &Repository{
		currentConfig: nil,
		mutex:         sync.RWMutex{},
	}
}
func (r *Repository) GetCurrentConfig() (*StorageData, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.currentConfig, nil
}
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
