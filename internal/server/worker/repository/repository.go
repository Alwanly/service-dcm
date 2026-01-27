package repository

import (
	"sync"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

// IRepository defines the interface for worker configuration storage
type IRepository interface {
	GetCurrentConfig() (*models.WorkerConfiguration, error)
	UpdateConfig(config *models.WorkerConfiguration) error
}

// Repository implements in-memory storage for worker configuration
type Repository struct {
	currentConfig *models.WorkerConfiguration
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
func (r *Repository) GetCurrentConfig() (*models.WorkerConfiguration, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if r.currentConfig == nil {
		return nil, nil
	}

	// Return a copy to prevent external modifications
	configCopy := *r.currentConfig
	if r.currentConfig.Headers != nil {
		configCopy.Headers = make(map[string]string)
		for k, v := range r.currentConfig.Headers {
			configCopy.Headers[k] = v
		}
	}

	return &configCopy, nil
}

// UpdateConfig updates the worker configuration
func (r *Repository) UpdateConfig(config *models.WorkerConfiguration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Store a copy to prevent external modifications
	configCopy := *config
	if config.Headers != nil {
		configCopy.Headers = make(map[string]string)
		for k, v := range config.Headers {
			configCopy.Headers[k] = v
		}
	}

	r.currentConfig = &configCopy
	return nil
}
