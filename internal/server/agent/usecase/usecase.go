package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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
	logger     *logger.CanonicalLogger
}

func NewUseCase(ctrl repository.IControllerClient, repo repository.IRepository, worker repository.IWorkerClient, cfg *config.AgentConfig, log *logger.CanonicalLogger) *UseCase {
	return &UseCase{controller: ctrl, repo: repo, worker: worker, cfg: cfg, logger: log}
}

// StartBackgroundServices initializes background listeners and polling (best-effort)
func (uc *UseCase) StartBackgroundServices(ctx context.Context, heartbeatInterval, fallbackInterval time.Duration) error {
	// Start Redis listener for push notifications
	if err := uc.repo.StartRedisListener(ctx, uc.logger); err != nil {
		uc.logger.WithError(err).Error("Failed to start Redis listener")
		// Continue operating in poll-only mode
	}

	// Start heartbeat polling if enabled
	if uc.cfg != nil {
		if uc.cfg.Heartbeat.Enabled && heartbeatInterval > 0 {
			uc.repo.RegisterHeartbeatPolling(ctx, uc.logger, heartbeatInterval)
		}
		if uc.cfg.FallbackPoll.Enabled && fallbackInterval > 0 {
			// Register fallback polling (uses same underlying mechanism)
			uc.repo.RegisterConfigPolling(ctx, uc.logger)
		}
	} else {
		// Fallback: register config polling
		uc.repo.RegisterConfigPolling(ctx, uc.logger)
	}

	return nil
}

// RegisterWithController registers the agent and stores agentID into the repository.
func (uc *UseCase) RegisterWithController(ctx context.Context, hostname, startTime string) (*models.RegistrationResponse, error) {
	var lastErr error
	var savedResp *models.RegistrationResponse
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
		// persist api token if provided
		if resp.APIToken != "" {
			uc.repo.SetAPIToken(resp.APIToken)
		}
		if err := uc.repo.SetPollInfo(resp.PollURL, resp.PollIntervalSeconds); err != nil {
			lastErr = fmt.Errorf("persist poll info: %w", err)
			return lastErr
		}
		savedResp = resp
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
	token := uc.repo.GetAPIToken()
	// prefer saved response if available
	if savedResp != nil {
		return savedResp, nil
	}
	return &models.RegistrationResponse{AgentID: agentID, PollIntervalSeconds: poll, APIToken: token}, nil
}

// GetConfigure is a FetchFunc implementation that polls for configuration updates.
// It wraps FetchConfiguration and uses the provided logger for debugging.
func (uc *UseCase) GetConfigure(ctx context.Context, log *logger.CanonicalLogger) error {
	log.Debug("starting configuration fetch")

	cfg, _, notModified, err := uc.FetchConfiguration(ctx)
	if err != nil {
		log.Error("configuration fetch failed", zap.Error(err))
		return err
	}

	if notModified {
		log.Debug("configuration not modified")
		return nil
	}

	if cfg != nil {
		log.Info("configuration updated",
			zap.String("etag", cfg.ETag))
	}

	return nil
}

// FetchConfiguration fetches configuration using ETag conditional requests.
func (uc *UseCase) FetchConfiguration(ctx context.Context) (*models.Configuration, *int, bool, error) {
	curCfg, _ := uc.repo.GetCurrentConfig()
	var curETag string
	if curCfg != nil {
		curETag = curCfg.ETag
	}

	agentID, _ := uc.repo.GetAgentID()
	pollURL, _, _ := uc.repo.GetPollInfo()

	cfg, newETag, pollInterval, notModified, err := uc.controller.GetConfiguration(ctx, agentID, pollURL, curETag)
	logger.AddToContext(ctx,
		zap.String("agent_id", agentID),
		zap.String("poll_url", pollURL),
		zap.String("if_none_match", curETag),
	)
	if err != nil {
		logger.AddToContext(ctx, zap.Error(err), zap.Bool(logger.FieldSuccess, false))
		return nil, nil, false, err
	}
	if notModified {
		logger.AddToContext(ctx, zap.Bool(logger.FieldSuccess, true), zap.String("result", "not_modified"))
		return nil, nil, true, nil
	}

	if cfg != nil {
		cfg.ETag = newETag
		if err := uc.repo.UpdateConfig(cfg); err != nil {
			return nil, nil, false, fmt.Errorf("update config repository: %w", err)
		}
		// Send configuration to worker with retry wrapper if supported

		// Ensure correlation ID is present in context for downstream worker calls
		corr := logger.GetCorrelationID(ctx)
		if corr == "" {
			corr = uuid.Must(uuid.NewV7()).String()
			ctx = logger.WithCorrelationID(ctx, corr)
		}
		uc.logger.Info("forwarding configuration to worker", zap.String("correlation_id", corr), zap.String("etag", cfg.ETag))

		if wc, ok := uc.worker.(interface {
			SendConfigurationWithRetry(context.Context, *models.Configuration, int) error
		}); ok {
			if err := wc.SendConfigurationWithRetry(ctx, cfg, 5); err != nil {
				return nil, nil, false, fmt.Errorf("send configuration to worker (with retry): %w", err)
			}
		} else {
			if err := uc.worker.SendConfiguration(ctx, cfg); err != nil {
				return nil, nil, false, fmt.Errorf("send configuration to worker: %w", err)
			}
		}
	}

	return cfg, pollInterval, false, nil
}

// GetPollInfo returns the stored poll URL and interval
func (uc *UseCase) GetPollInfo() (string, int, error) {
	return uc.repo.GetPollInfo()
}

// SetStoredPollInterval updates the stored polling interval in the repository
func (uc *UseCase) SetStoredPollInterval(newInterval int) {
	uc.repo.UpdatePollInterval(newInterval)
}

// GetAgentID returns the currently stored agent ID
func (uc *UseCase) GetAgentID() (string, error) {
	return uc.repo.GetAgentID()
}
