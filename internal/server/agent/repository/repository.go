package repository

import (
	"sync"

	"github.com/Alwanly/service-distribute-management/internal/models"
)

type StoreData struct {
	Config       *models.Configuration
	ETag         string
	AgentID      string
	PollURL      string
	PollInterval int
	APIToken     string
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

// GetAgentID returns the stored agent ID
func (r *Repository) GetAgentID() (string, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		return "", nil
	}
	return r.currentConfig.AgentID, nil
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

// SetPollInfo sets the poll URL and interval
func (r *Repository) SetPollInfo(pollURL string, pollInterval int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		r.currentConfig = &StoreData{}
	}
	r.currentConfig.PollURL = pollURL
	r.currentConfig.PollInterval = pollInterval
	return nil
}

// GetPollInfo retrieves the poll URL and interval
func (r *Repository) GetPollInfo() (string, int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		return "", 0, nil
	}
	return r.currentConfig.PollURL, r.currentConfig.PollInterval, nil
}

// SetAPIToken stores the API token for future requests
func (r *Repository) SetAPIToken(token string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		r.currentConfig = &StoreData{}
	}
	r.currentConfig.APIToken = token
}

// GetAPIToken returns the stored API token
func (r *Repository) GetAPIToken() string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		return ""
	}
	return r.currentConfig.APIToken
}

// UpdatePollInterval updates the stored polling interval
func (r *Repository) UpdatePollInterval(newInterval int) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		r.currentConfig = &StoreData{}
	}
	r.currentConfig.PollInterval = newInterval
}

// SetConfig stores configuration and its ETag
func (r *Repository) SetConfig(config *models.Configuration, etag string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		r.currentConfig = &StoreData{}
	}
	r.currentConfig.Config = config
	r.currentConfig.ETag = etag
}

// GetConfig retrieves stored configuration and ETag
func (r *Repository) GetConfig() (*models.Configuration, string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.currentConfig == nil {
		return nil, ""
	}
	return r.currentConfig.Config, r.currentConfig.ETag
}
