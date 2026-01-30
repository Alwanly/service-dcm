package usecase

import (
	"context"
	"encoding/json"
	"net/http"

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
	// Determine default poll interval
	defaultInterval := int(uc.Config.PollInterval.Seconds())

	// Create agent with UUID and API token
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
func (uc *UseCase) GetConfigForAgent(agentID string, etag string) wrapper.JSONResult {
	// Look up agent to get poll interval
	agent, err := uc.Repo.GetAgentByID(agentID)
	if err != nil {
		uc.Logger.Error("failed to get agent for config",
			zap.String("agent_id", agentID),
			zap.Error(err),
		)
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get agent", err)
	}

	// Check config changes
	latestETag, configData, err := uc.Repo.GetConfigIfChanged(etag)
	if err != nil {
		uc.Logger.Error("failed to get config",
			zap.Error(err),
			zap.String("agent_id", agentID),
		)
		return wrapper.ResponseFailed(http.StatusInternalServerError, "failed to get configuration", err)
	}

	if latestETag == "" {
		// Not modified
		uc.Logger.Debug("configuration not modified",
			zap.String("agent_id", agentID),
			zap.String("etag", etag),
		)
		return wrapper.ResponseSuccess(http.StatusNotModified, nil)
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

	uc.Logger.Info("configuration returned",
		zap.String("agent_id", agentID),
		zap.String("agent_name", agent.AgentName),
		zap.Int("poll_interval_seconds", *pollInterval),
	)

	return wrapper.ResponseSuccess(http.StatusOK, response)
}
