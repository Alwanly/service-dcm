package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/handler"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/Alwanly/service-distribute-management/pkg/poll"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
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

	// initialize poller
	poller := poll.NewPoller(log)

	// Create Fiber app
	app := fiber.New(fiber.Config{DisableStartupMessage: true, ErrorHandler: middleware.ErrorHandler(log)})

	// Add middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.CanonicalLoggerMiddleware(log))

	// Initialize dependencies
	deps := deps.App{
		Fiber:  app,
		Logger: log,
		Poller: poller,
	}

	// Initialize and register handler
	handler.NewHandler(deps, cfg)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	// Start poller
	g.Go(func() error {
		log.Info("starting poller")
		if err := poller.Start(gCtx); err != nil {
			return fmt.Errorf("poller stopped with error: %w", err)
		}
		return nil
	})

	// Handle graceful shutdown
	g.Go(func() error {
		<-gCtx.Done()

		if err := app.Shutdown(); err != nil {
			log.WithError(err).Error("failed to shutdown fiber app")
			return err
		}

		if err := poller.Stop(); err != nil {
			log.WithError(err).Error("failed to stop poller")
			return err
		}

		return nil
	})

	// listen for OS signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		log.Info("listening for OS signals")
		<-sigChan
		log.Info("shutdown signal received")
		cancel()
	}()

	if err := g.Wait(); err != nil {
		log.WithError(err).Error("agent service stopped with error")
	}

	log.Info("agent service stopped gracefully")
}
