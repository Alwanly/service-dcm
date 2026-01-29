package usecase

import (
	"context"
	"fmt"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/retry"
	"go.uber.org/zap"
)

type UseCase struct {
	controller repository.IControllerClient
	repo       repository.IRepository
	worker     repository.IWorkerClient
	cfg        *config.AgentConfig
}

func NewUseCase(ctrl repository.IControllerClient, repo repository.IRepository, worker repository.IWorkerClient, cfg *config.AgentConfig) *UseCase {
	return &UseCase{controller: ctrl, repo: repo, worker: worker, cfg: cfg}
}

// RegisterWithController registers the agent and stores agentID into the repository.
func (uc *UseCase) RegisterWithController(ctx context.Context, hostname, startTime string) (*models.RegistrationResponse, error) {
	var lastErr error
	op := func(ctx context.Context) error {
		resp, err := uc.controller.Register(ctx, hostname, "", startTime)
		if err != nil {
			lastErr = err
			return err
		}
		if resp == nil || resp.AgentID == "" {
			lastErr = fmt.Errorf("invalid registration response")
			return lastErr
		}
		if err := uc.repo.SetAgentID(resp.AgentID); err != nil {
			lastErr = fmt.Errorf("persist agent id: %w", err)
			return lastErr
		}
		if err := uc.repo.SetPollInfo(resp.PollURL, resp.PollIntervalSeconds); err != nil {
			lastErr = fmt.Errorf("persist poll info: %w", err)
			return lastErr
		}
		return nil
	}

	retryCfg := retry.Config{
		MaxRetries:     uc.cfg.RegistrationMaxRetries,
		InitialBackoff: uc.cfg.RegistrationInitialBackoff,
		MaxBackoff:     uc.cfg.RegistrationMaxBackoff,
		Multiplier:     uc.cfg.RegistrationBackoffMultiplier,
		Jitter:         true,
	}

	if err := retry.WithExponentialBackoff(ctx, retryCfg, op); err != nil {
		return nil, fmt.Errorf("register with controller failed after retries: %w", lastErr)
	}

	agentID, _ := uc.repo.GetAgentID()
	_, poll, _ := uc.repo.GetPollInfo()
	return &models.RegistrationResponse{AgentID: agentID, PollIntervalSeconds: poll}, nil
}

// FetchConfiguration fetches configuration using ETag conditional requests.
func (uc *UseCase) FetchConfiguration(ctx context.Context) (*models.Configuration, bool, error) {
	curCfg, _ := uc.repo.GetCurrentConfig()
	var curETag string
	if curCfg != nil {
		curETag = curCfg.ETag
	}

	agentID, _ := uc.repo.GetAgentID()
	pollURL, _, _ := uc.repo.GetPollInfo()

	cfg, newETag, notModified, err := uc.controller.GetConfiguration(ctx, agentID, pollURL, curETag)
	logger.AddToContext(ctx,
		zap.String("agent_id", agentID),
		zap.String("poll_url", pollURL),
		zap.String("if_none_match", curETag),
	)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return nil, false, err
	}
	if notModified {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true), zap.String("result", "not_modified"))
		return nil, true, nil
	}

	if cfg != nil {
		cfg.ETag = newETag
		if err := uc.repo.UpdateConfig(cfg); err != nil {
			return nil, false, fmt.Errorf("update config repository: %w", err)
		}
		if err := uc.worker.SendConfiguration(ctx, cfg); err != nil {
			return nil, false, fmt.Errorf("send configuration to worker: %w", err)
		}
	}

	return cfg, false, nil
}
