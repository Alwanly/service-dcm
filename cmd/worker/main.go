package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/logger"
	"github.com/Alwanly/service-distribute-management/internal/server"
)

func main() {
	// Initialize logger
	log, err := logger.NewLoggerFromEnv("worker")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("starting worker service")

	// Load configuration
	cfg, err := config.LoadWorkerConfig()
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration")
	}

	log.Info("configuration loaded",
		logger.String("server_addr", cfg.ServerAddr),
		logger.Duration("request_timeout", cfg.RequestTimeout),
	)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Worker Service",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(loggingMiddleware(log))

	// Create server
	workerServer := server.NewWorkerServer(cfg.RequestTimeout)

	// Setup routes
	workerServer.SetupRoutes(app)

	// Start server in goroutine
	go func() {
		log.Info("worker service listening", logger.String("addr", cfg.ServerAddr))
		if err := app.Listen(cfg.ServerAddr); err != nil {
			log.WithError(err).Fatal("server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down worker service")

	// Graceful shutdown
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.WithError(err).Error("server shutdown error")
	}

	log.Info("worker service stopped")
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(log *logger.CanonicalLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		duration := time.Since(start).Milliseconds()
		log.HTTP(c.Method(), c.Path(), c.Response().StatusCode(), duration)

		return err
	}
}
