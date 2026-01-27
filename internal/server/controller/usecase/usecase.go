package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/repository"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
	"github.com/google/uuid"
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
	agentID, err := uuid.NewV7()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to generate agent ID", err)
	}

	startup_time := time.Now()
	status := "active"

	err = uc.Repo.RegisterAgent(agentID.String(), startup_time, status)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to register agent", err)
	}

	// Add success context
	logger.AddToContext(ctx,
		zap.String(logger.FieldAgentID, agentID.String()),
		zap.Bool(logger.FieldSuccess, true),
	)

	pollURL := fmt.Sprintf("http://%s/config", uc.Config.ServerAddr)
	pollInterval := int(uc.Config.PollInterval.Seconds())

	return wrapper.ResponseSuccess(http.StatusOK, dto.RegisterAgentResponse{
		AgentID:             agentID.String(),
		PollURL:             pollURL,
		PollIntervalSeconds: pollInterval,
	})
}

func (uc *UseCase) UpdateConfig(ctx context.Context, req *dto.SetConfigAgentRequest) wrapper.JSONResult {
	config, err := json.Marshal(req.Config)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to marshal config data", err)
	}

	err = uc.Repo.UpdateConfig(string(config))
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to update config", err)
	}

	logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true))
	return wrapper.ResponseSuccess(http.StatusOK, "Config updated successfully")
}

func (uc *UseCase) GetConfig(ctx context.Context) wrapper.JSONResult {
	etag, err := uc.Repo.GetConfigETag()
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to get config", err)
	}

	configData, err := uc.Repo.GetConfig(etag)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to get config if changed", err)
	}

	if configData == "" {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true), zap.String("result", "not_modified"))
		return wrapper.ResponseSuccess(http.StatusNotModified, nil)
	}

	var config interface{}
	err = json.Unmarshal([]byte(configData), &config)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return wrapper.ResponseFailed(http.StatusInternalServerError, "Failed to unmarshal config data", err)
	}

	logger.AddToContext(ctx,
		zap.String(logger.FieldETag, etag),
		zap.Bool(logger.FieldSuccess, true),
	)

	response := dto.GetConfigAgentResponse{
		ETag:   etag,
		Config: config,
	}
	return wrapper.ResponseSuccess(http.StatusOK, response)
}
