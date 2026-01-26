package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"github.com/Alwanly/service-distribute-management/internal/auth"
	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/logger"
	"github.com/Alwanly/service-distribute-management/internal/server"
	"github.com/Alwanly/service-distribute-management/internal/store"
)

func main() {
	// Initialize logger
	log, err := logger.NewLoggerFromEnv("controller")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("starting controller service")

	// Load configuration
	cfg, err := config.LoadControllerConfig()
	if err != nil {
		log.WithError(err).Fatal("failed to load configuration")
	}

	log.Info("configuration loaded",
		logger.String("server_addr", cfg.ServerAddr),
		logger.String("database_path", cfg.DatabasePath),
		logger.Duration("poll_interval", cfg.PollInterval),
	)

	// Initialize authentication
	auth.Initialize(cfg.AdminUsername, cfg.AdminPassword, cfg.AgentUsername, cfg.AgentPassword)
	log.Info("authentication initialized")

	// Initialize database
	db, err := store.NewDB(cfg.DatabasePath)
	if err != nil {
		log.WithError(err).Fatal("failed to initialize database")
	}
	defer db.Close()
	log.Info("database initialized", logger.String("path", cfg.DatabasePath))

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Controller Service",
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: errorHandler(log),
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(loggingMiddleware(log))

	// Create server
	pollIntervalSeconds := int(cfg.PollInterval.Seconds())
	controllerServer := server.NewControllerServer(db, pollIntervalSeconds, "")

	// Setup routes
	controllerServer.SetupRoutes(app)

	// Start server in goroutine
	go func() {
		log.Info("controller service listening", logger.String("addr", cfg.ServerAddr))
		if err := app.Listen(cfg.ServerAddr); err != nil {
			log.WithError(err).Fatal("server error")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down controller service")

	// Graceful shutdown
	if err := app.ShutdownWithTimeout(30 * time.Second); err != nil {
		log.WithError(err).Error("server shutdown error")
	}

	log.Info("controller service stopped")
}

// errorHandler creates a custom Fiber error handler
func errorHandler(log *logger.CanonicalLogger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		log.HTTPError(c.Method(), c.Path(), code, err)

		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
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
