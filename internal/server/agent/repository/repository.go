package repository

import (
	"sync"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

type StoreData struct {
	Config  *models.Configuration
	ETag    string
	AgentID string
}

type Repository struct {
	currentConfig *StoreData
	mutex         sync.Mutex
}

// NewRepository creates a new repository instance
func NewRepository() IRepository {
	return &Repository{
		currentConfig: nil,
		mutex:         sync.Mutex{},
	}
}

// SetAgentID sets the agent ID
func (r *Repository) SetAgentID(agentID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		r.currentConfig = &StoreData{}
	}
	r.currentConfig.AgentID = agentID
	return nil
}

// GetCurrentConfig retrieves the current worker configuration
func (r *Repository) GetCurrentConfig() (*models.Configuration, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		return nil, nil
	}
	return r.currentConfig.Config, nil
}

// UpdateConfig updates the worker configuration
func (r *Repository) UpdateConfig(config *models.Configuration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.currentConfig = &StoreData{
		Config: config,
		ETag:   config.ETag,
	}
	return nil
}
