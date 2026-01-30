package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Alwanly/service-distribute-management/pkg/pubsub"
)

type Repository struct {
	DB  *gorm.DB
	Pub pubsub.Publisher
}

func NewRepository(db *gorm.DB, publisher pubsub.Publisher) *Repository {
	return &Repository{DB: db, Pub: publisher}
}

type IRepository interface {
	RegisterAgent(ctx context.Context, data *models.Agent) error
	UpdateConfig(ctx context.Context, config string) error
	GetConfigETag(ctx context.Context) (string, error)
	GetConfig(ctx context.Context, config string) (models.ConfigData, error)
	GetConfigIfChanged(currentETag string) (string, models.ConfigData, error)
	PublishConfigUpdate(agentID string, etag string, correlationID string) error
	UpdateAgentHeartbeat(agentID string, configVersion string) (*models.Agent, error)
	GetLatestConfigVersionForAgent(agentID string) (string, error)
}

func (r *Repository) RegisterAgent(ctx context.Context, data *models.Agent) error {
	result := r.DB.WithContext(ctx).Create(data)
	return result.Error
}

// CreateAgent creates a new agent with UUID and API token
func (r *Repository) CreateAgent(agentName string, pollIntervalSeconds *int) (*models.AgentConfig, error) {
	// Generate UUID v7 for agent ID
	agentID := uuid.Must(uuid.NewV7()).String()

	// Generate secure random API token (32 bytes = 64 hex chars)
	apiToken, err := generateSecureToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate api token: %w", err)
	}

	agent := &models.AgentConfig{
		ID:                  agentID,
		AgentName:           agentName,
		APIToken:            apiToken,
		PollIntervalSeconds: pollIntervalSeconds,
	}

	if err := r.DB.Create(agent).Error; err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	return agent, nil
}

// GetAgentByID retrieves an agent by UUID
func (r *Repository) GetAgentByID(agentID string) (*models.AgentConfig, error) {
	var agent models.AgentConfig
	if err := r.DB.Where("id = ?", agentID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent not found: %s", agentID)
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return &agent, nil
}

// GetAgentByToken retrieves an agent by API token
func (r *Repository) GetAgentByToken(apiToken string) (*models.AgentConfig, error) {
	var agent models.AgentConfig
	if err := r.DB.Where("api_token = ?", apiToken).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent not found")
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return &agent, nil
}

// UpdateAgentPollInterval updates the polling interval for an agent
func (r *Repository) UpdateAgentPollInterval(agentID string, intervalSeconds *int) error {
	result := r.DB.Model(&models.AgentConfig{}).
		Where("id = ?", agentID).
		Update("poll_interval_seconds", intervalSeconds)

	if result.Error != nil {
		return fmt.Errorf("failed to update poll interval: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	return nil
}

// RotateAgentToken generates a new API token for an agent
func (r *Repository) RotateAgentToken(agentID string) (string, error) {
	newToken, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate new token: %w", err)
	}

	result := r.DB.Model(&models.AgentConfig{}).
		Where("id = ?", agentID).
		Update("api_token", newToken)

	if result.Error != nil {
		return "", fmt.Errorf("failed to rotate token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return "", fmt.Errorf("agent not found: %s", agentID)
	}

	return newToken, nil
}

// ListAgents retrieves all registered agents
func (r *Repository) ListAgents() ([]models.AgentPublic, error) {
	var agents []models.AgentConfig
	if err := r.DB.Order("created_at DESC").Find(&agents).Error; err != nil {
		return nil, fmt.Errorf("failed to list agents: %w", err)
	}

	public := make([]models.AgentPublic, len(agents))
	for i, a := range agents {
		public[i] = a.ToPublic()
	}
	return public, nil
}

// DeleteAgent removes an agent by ID
func (r *Repository) DeleteAgent(agentID string) error {
	result := r.DB.Delete(&models.AgentConfig{}, "id = ?", agentID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete agent: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	return nil
}

// generateSecureToken creates a cryptographically secure random token
func generateSecureToken(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
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
		// create default configuration when none exists
		defaultConfig := "{}"
		etag = generateETag(defaultConfig)
		if createErr := r.DB.WithContext(ctx).Create(&models.Configuration{
			ETag:       etag,
			ConfigData: defaultConfig,
		}).Error; createErr != nil {
			return "", createErr
		}
		return etag, nil
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

// PublishConfigUpdate publishes a configuration change notification to Redis (if configured)
func (r *Repository) PublishConfigUpdate(agentID string, etag string, correlationID string) error {
	if r.Pub == nil {
		// Redis not configured; nothing to do
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	notification := map[string]string{
		"agent_id":       agentID,
		"etag":           etag,
		"correlation_id": correlationID,
	}

	payload, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("failed to marshal config update notification: %w", err)
	}

	channel := "config-updates"
	if err := r.Pub.Publish(ctx, channel, string(payload)); err != nil {
		return fmt.Errorf("failed to publish config update: %w", err)
	}

	return nil
}

// UpdateAgentHeartbeat updates the agent's last heartbeat timestamp and last config version
func (r *Repository) UpdateAgentHeartbeat(agentID string, configVersion string) (*models.Agent, error) {
	var agent models.Agent
	now := time.Now().UTC()

	result := r.DB.Model(&models.Agent{}).
		Where("agent_id = ?", agentID).
		Save(map[string]interface{}{
			"agent_id":            agentID,
			"last_heartbeat":      now,
			"last_config_version": configVersion,
		})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to update agent heartbeat: %w", result.Error)
	}

	if err := r.DB.Where("agent_id = ?", agentID).First(&agent).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve agent after heartbeat update: %w", err)
	}
	return &agent, nil
}

// GetLatestConfigVersionForAgent returns the latest configuration ETag (global) for now
func (r *Repository) GetLatestConfigVersionForAgent(agentID string) (string, error) {
	// For now return the global latest configuration ETag
	etag, err := r.GetConfigETag(context.Background())
	if err != nil {
		return "", err
	}
	return etag, nil
}
