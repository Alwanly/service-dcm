package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Alwanly/service-distribute-management/internal/config"
	agenthandler "github.com/Alwanly/service-distribute-management/internal/server/agent/handler"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/Alwanly/service-distribute-management/pkg/retry"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/sync/errgroup"
)

func main() {
	log, err := logger.NewLoggerFromEnv("agent")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("starting agent service")

	cfg, err := config.LoadAgentConfig()
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration")
	}

	log.Info("configuration loaded",
		logger.String("controller_url", cfg.ControllerURL),
		logger.String("worker_url", cfg.WorkerURL),
		logger.String("agent_addr", cfg.AgentAddr),
	)

	// Create Fiber app
	app := fiber.New(fiber.Config{DisableStartupMessage: true, ErrorHandler: middleware.ErrorHandler(log)})

	// Initialize repositories
	controllerClient := repository.NewControllerClient(cfg, log)
	workerClient := repository.NewWorkerClient(cfg, log)

	// Initialize usecase
	uc := usecase.NewUseCase(controllerClient, workerClient, cfg, log)

	// Initialize handler
	h := agenthandler.NewHandler(uc, log)
	h.RegisterRoutes(app)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Register agent with controller with retry
	var agentID string
	registerOp := func(ctx context.Context) error {
		id, err := uc.RegisterWithController(ctx)
		if err == nil {
			agentID = id
		}
		return err
	}

	backoffCfg := retry.Config{
		MaxRetries:     cfg.RegistrationMaxRetries,
		InitialBackoff: cfg.RegistrationInitialBackoff,
		MaxBackoff:     cfg.RegistrationMaxBackoff,
		Multiplier:     cfg.RegistrationBackoffMultiplier,
		Jitter:         true,
	}

	if err := retry.WithExponentialBackoff(ctx, backoffCfg, func(c context.Context) error { return registerOp(c) }); err != nil {
		log.WithError(err).Fatal("failed to register with controller after retries")
	}

	log.WithAgentID(agentID).Info("agent registered successfully")

	// Start polling
	if err := uc.StartPolling(ctx, agentID); err != nil {
		log.WithError(err).Fatal("failed to start polling")
	}

	// Use errgroup for managing concurrent goroutines
	g, gCtx := errgroup.WithContext(ctx)

	// Start HTTP server
	g.Go(func() error {
		log.Info("starting HTTP server", logger.String("address", cfg.AgentAddr))
		if err := app.Listen(cfg.AgentAddr); err != nil {
			return fmt.Errorf("failed to start server: %w", err)
		}
		return nil
	})

	// Handle graceful shutdown
	g.Go(func() error {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-sigCh:
			log.Info("received shutdown signal", logger.String("signal", sig.String()))
		case <-gCtx.Done():
			log.Info("context cancelled")
		}

		if err := uc.StopPolling(); err != nil {
			log.WithError(err).Error("error stopping polling")
		}

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := app.ShutdownWithContext(shutdownCtx); err != nil {
			log.WithError(err).Error("error during server shutdown")
		}

		cancel()
		return nil
	})

	if err := g.Wait(); err != nil {
		log.WithError(err).Error("agent service stopped with error")
		os.Exit(1)
	}

	log.Info("agent service stopped gracefully")
}
