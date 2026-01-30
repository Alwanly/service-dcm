package usecase

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/repository"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
	"go.uber.org/zap"
)

type UseCase struct {
	Repo   *repository.Repository
	Config *config.ControllerConfig
	Logger *logger.CanonicalLogger
}

func NewUseCase(uc UseCase) *UseCase {
	return &UseCase{
		Repo:   uc.Repo,
		Config: uc.Config,
		Logger: uc.Logger,
	}
}

func (uc *UseCase) RegisterAgent(ctx context.Context, req *dto.RegisterAgentRequest) wrapper.JSONResult {
	defaultInterval := int(uc.Config.PollInterval.Seconds())
	agent, err := uc.Repo.CreateAgent(req.Hostname, &defaultInterval)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to create agent", err)
	}

	uc.Logger.Info("agent registered successfully",
		zap.String("agent_id", agent.ID),
		zap.String("agent_name", agent.AgentName),
		zap.Int("poll_interval_seconds", defaultInterval),
	)

	response := dto.RegisterAgentResponse{
		AgentID:             agent.ID,
		AgentName:           agent.AgentName,
		APIToken:            agent.APIToken,
		PollURL:             "/config",
		PollIntervalSeconds: defaultInterval,
	}

	return wrapper.ResponseSuccess(http.StatusOK, response)
}

func (uc *UseCase) UpdateConfig(ctx context.Context, req *dto.SetConfigAgentRequest) wrapper.JSONResult {
	correlationID := uuid.New().String()

	logger.AddToContext(ctx, zap.String("correlation_id", correlationID))

	config, err := json.Marshal(req)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to marshal config data", err)
	}

	err = uc.Repo.UpdateConfig(ctx, string(config))
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to update config", err)
	}

	// Publish notification to Redis (best-effort) with correlation ID
	if etag, gerr := uc.Repo.GetConfigETag(ctx); gerr == nil {
		if perr := uc.Repo.PublishConfigUpdate("", etag, correlationID); perr != nil {
			uc.Logger.WithError(perr).Error("failed to publish config update", zap.String("correlation_id", correlationID))
		} else {
			uc.Logger.Info("config update published", zap.String("correlation_id", correlationID), zap.String("etag", etag))
		}
	} else {
		uc.Logger.WithError(gerr).Error("failed to get config ETag after update", zap.String("correlation_id", correlationID))
	}

	logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true))
	return wrapper.ResponseSuccess(http.StatusOK, "Config updated successfully")
}

func (uc *UseCase) GetConfig(ctx context.Context, req *dto.GetConfigAgentRequest) wrapper.JSONResult {
	etag, err := uc.Repo.GetConfigETag(ctx)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to get config", err)
	}

	if etag == req.ETag {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true), zap.String("result", "not_modified"))
		return wrapper.ResponseSuccess(http.StatusNotModified, nil)
	}

	configData, err := uc.Repo.GetConfig(ctx, etag)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to get config if changed", err)
	}

	if configData == nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true), zap.String("result", "not_modified"))
		return wrapper.ResponseSuccess(http.StatusNotModified, nil)
	}

	logger.AddToContext(ctx,
		zap.String(logger.FieldETag, etag),
		zap.Bool(logger.FieldSuccess, true),
	)

	response := dto.GetConfigAgentResponse{
		ETag:   etag,
		Config: configData,
	}
	return wrapper.ResponseSuccess(http.StatusOK, response)
}

// GetConfigForAgent returns configuration for authenticated agent with poll interval
func (uc *UseCase) GetConfigForAgent(ctx context.Context, agentID string, etag string) wrapper.JSONResult {
	// Look up agent to get poll interval
	agent, err := uc.Repo.GetAgentByID(agentID)
	if err != nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, false), zap.Error(err))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get agent", err)
	}

	// Get current configuration
	latestETag, err := uc.Repo.GetConfigETag(ctx)
	if err != nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, false), zap.Error(err))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get configuration ETag", err)
	}

	// If ETag matches, return 304 Not Modified
	if latestETag == etag {
		// Not modified
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true), zap.String("result", "not_modified"))
		return wrapper.ResponseSuccess(http.StatusNotModified, nil)
	}

	// Get configuration data
	configData, err := uc.Repo.GetConfig(ctx, latestETag)
	if err != nil {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, false), zap.Error(err))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get configuration data", err)
	}

	// Determine poll interval (agent-specific or global default)
	var pollInterval *int
	if agent.PollIntervalSeconds != nil {
		pollInterval = agent.PollIntervalSeconds
	} else {
		defaultInterval := int(uc.Config.PollInterval.Seconds())
		pollInterval = &defaultInterval
	}

	response := dto.GetConfigAgentResponse{
		ID:                  1, // Placeholder config ID
		ETag:                latestETag,
		Config:              configData,
		PollIntervalSeconds: pollInterval,
	}

	logger.AddToContext(ctx,
		zap.String(logger.FieldETag, latestETag),
		zap.Bool(logger.FieldSuccess, true),
	)

	return wrapper.ResponseSuccess(http.StatusOK, response)
}

// UpdateAgentPollInterval updates the polling interval for a specific agent
func (uc *UseCase) UpdateAgentPollInterval(agentID string, intervalSeconds *int) error {
	if err := uc.Repo.UpdateAgentPollInterval(agentID, intervalSeconds); err != nil {
		uc.Logger.Error("failed to update agent poll interval", zap.Error(err), zap.String("agent_id", agentID))
		return err
	}
	uc.Logger.Info("agent poll interval updated", zap.String("agent_id", agentID))
	return nil
}

// RotateAgentToken generates a new API token for an agent and returns it
func (uc *UseCase) RotateAgentToken(ctx context.Context, agentID string) wrapper.JSONResult {
	newToken, err := uc.Repo.RotateAgentToken(agentID)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to rotate token", err)
	}

	response := dto.RotateTokenResponse{
		AgentID:  agentID,
		APIToken: newToken,
		Message:  "token rotated",
	}
	logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true))
	return wrapper.ResponseSuccess(http.StatusOK, response)
}

// GetAgent retrieves details for a specific agent
func (uc *UseCase) GetAgent(ctx context.Context, agentID string) wrapper.JSONResult {
	agent, err := uc.Repo.GetAgentByID(agentID)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get agent", err)
	}
	logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true))
	return wrapper.ResponseSuccess(http.StatusOK, agent.ToPublic())
}

// HandleHeartbeat processes an agent heartbeat and returns latest config version info
func (uc *UseCase) HandleHeartbeat(agentID string, req *dto.HeartbeatRequest) (*dto.HeartbeatResponse, error) {
	// Update heartbeat timestamp in DB
	agent, err := uc.Repo.UpdateAgentHeartbeat(agentID, req.ConfigVersion)
	if err != nil {
		uc.Logger.Error("failed to update agent heartbeat", zap.Error(err), zap.String("agent_id", agentID))
		return nil, err
	}

	// Get latest config version for agent
	latest, err := uc.Repo.GetLatestConfigVersionForAgent(agentID)
	if err != nil {
		uc.Logger.Error("failed to get latest config version", zap.Error(err), zap.String("agent_id", agentID))
		return nil, err
	}

	resp := &dto.HeartbeatResponse{
		LatestConfigVersion: latest,
		ReceivedAt:          time.Now().UTC(),
	}

	uc.Logger.Info("heartbeat processed", zap.String("agent_id", agentID), zap.String("latest_config", latest))
	_ = agent
	return resp, nil
}

// ListAgents returns all registered agents
func (uc *UseCase) ListAgents(ctx context.Context) wrapper.JSONResult {
	agents, err := uc.Repo.ListAgents()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to list agents", err)
	}
	response := dto.ListAgentsResponse{
		Agents: agents,
		Total:  len(agents),
	}
	logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true))
	return wrapper.ResponseSuccess(http.StatusOK, response)
}

// DeleteAgent removes an agent by ID
func (uc *UseCase) DeleteAgent(ctx context.Context, agentID string) error {
	if err := uc.Repo.DeleteAgent(agentID); err != nil {
		uc.Logger.Error("failed to delete agent", zap.Error(err), zap.String("agent_id", agentID))
		return err
	}
	uc.Logger.Info("agent deleted", zap.String("agent_id", agentID))
	return nil
}
