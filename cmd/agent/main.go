package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/internal/server/agent"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/retry"
)

const version = "1.0.0"

func main() {
	// Initialize logger
	log, err := logger.NewLoggerFromEnv("agent")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("starting agent service")

	// Load configuration
	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration")
	}

	log.Info("configuration loaded",
		logger.String("controller_url", cfg.ControllerURL),
		logger.String("worker_url", cfg.WorkerURL),
		logger.Duration("poll_interval", cfg.PollInterval),
	)

	// Create HTTP client for worker communication
	workerClient := &http.Client{
		Timeout: cfg.RequestTimeout,
	}

	// Create controller client with retry configuration
	log.Component("agent")
	controllerRetryCfg := retry.Config{
		MaxRetries:     cfg.RegistrationMaxRetries,
		InitialBackoff: cfg.RegistrationInitialBackoff,
		MaxBackoff:     cfg.RegistrationMaxBackoff,
		Multiplier:     cfg.RegistrationBackoffMultiplier,
		Jitter:         false,
	}

	client := agent.NewControllerClient(cfg.ControllerURL, cfg.AgentUsername, cfg.AgentPassword, cfg.RequestTimeout, log, controllerRetryCfg)

	hostname, _ := os.Hostname()
	regResp, err := client.Register(context.Background(), hostname, version, time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		log.WithError(err).Fatal("failed to register with controller")
	}

	log.WithAgentID(regResp.AgentID).Info("registered with controller",
		logger.Int("poll_interval", regResp.PollIntervalSeconds),
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start configuration poller
	poller := agent.NewPoller(client, cfg.PollInterval, regResp.AgentID, func(config *models.WorkerConfiguration) {
		log.WithConfigVersion(config.Version).Info("received new configuration",
			logger.String("target_url", config.TargetURL),
		)

		// Forward configuration to worker
		if err := sendConfigToWorker(workerClient, cfg.WorkerURL, config, log); err != nil {
			log.WithError(err).Error("failed to send config to worker")
		} else {
			log.Info("configuration forwarded to worker")
		}
	})

	go func() {
		if err := poller.Start(ctx); err != nil && err != context.Canceled {
			log.WithError(err).Error("poller error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down agent service")
	cancel()

	// Give poller time to stop gracefully
	time.Sleep(2 * time.Second)

	log.Info("agent service stopped")
}

// sendConfigToWorker sends configuration to the worker service
func sendConfigToWorker(client *http.Client, workerURL string, config *models.WorkerConfiguration, log *logger.CanonicalLogger) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, workerURL+"/config", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("worker returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
