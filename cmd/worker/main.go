package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/handler"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
)

func main() {
	// Initialize logger
	log, err := logger.NewLoggerFromEnv("worker")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("Starting Worker Service...")

	// Load configuration
	cfg, err := config.LoadWorkerConfig()
	if err != nil {
		log.Fatal("Failed to load configuration")
	}

	log.Info("configuration loaded",
		logger.String("server_addr", cfg.ServerAddr),
		logger.Duration("request_timeout", cfg.RequestTimeout),
	)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Worker Service",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Error("Fiber error handler")
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// Setup middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	// simple logging middleware
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		duration := time.Since(start).Milliseconds()
		log.HTTP(c.Method(), c.Path(), c.Response().StatusCode(), duration)
		return err
	})

	// Create dependency container
	dependencies := deps.App{
		Fiber:  app,
		Logger: log,
	}

	// Initialize handler (creates full dependency chain)
	handler.NewHandler(dependencies, cfg.RequestTimeout)

	log.Info("Worker Service configured",
		logger.String("addr", cfg.ServerAddr),
		logger.Duration("request_timeout", cfg.RequestTimeout),
	)

	// Start server in goroutine
	go func() {
		addr := cfg.ServerAddr
		log.Info("Worker Service starting", logger.String("address", addr))
		if err := app.Listen(addr); err != nil {
			log.Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Worker Service...")

	// Gracefully shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error("Server forced to shutdown")
	}

	log.Info("Worker Service stopped")
}
