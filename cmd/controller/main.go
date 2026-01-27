package main

// @title           Service Distribute Management - Controller API
// @version         1.0
// @description     Controller service for distributed configuration management system. Manages agent registration and worker configuration distribution.
// @termsOfService  http://swagger.io/terms/
// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @host      localhost:8080
// @BasePath  /
// @securityDefinitions.basic  BasicAuth

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	_ "github.com/Alwanly/service-distribute-management/docs/controller"
	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/handler"
	authentication "github.com/Alwanly/service-distribute-management/pkg/auth"
	"github.com/Alwanly/service-distribute-management/pkg/database"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	swagger "github.com/gofiber/swagger"
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

	auth := middleware.SetBasicAuth(&authentication.BasicAuthTConfig{
		Username:      cfg.AgentUsername,
		Password:      cfg.AgentPassword,
		AdminUsername: cfg.AdminUsername,
		AdminPassword: cfg.AdminPassword,
	})
	mid := middleware.NewAuthMiddleware(auth)
	log.Info("authentication initialized")

	// Initialize database
	db, err := database.NewSQLiteDB(cfg.DatabasePath)
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

	// Initialize dependencies
	deps := deps.App{
		Fiber:      app,
		Database:   db,
		Logger:     log,
		Middleware: mid,
	}

	// Register handlers
	handler.NewHandler(deps)

	// Swagger documentation route (accessible without authentication)
	app.Get("/swagger/*", swagger.HandlerDefault)

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
