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
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"

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
		panic(err)
	}
	log.Info("database initialized", logger.String("path", cfg.DatabasePath))

	// Migrate database schema
	if err := database.RunMigrations(db); err != nil {
		log.WithError(err).Fatal("failed to migrate database")
		panic(err)
	}
	log.Info("database migrations applied successfully")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Controller Service",
		DisableStartupMessage: true,
		ErrorHandler:          middleware.ErrorHandler(log),
	})

	// Add middleware
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.CanonicalLoggerMiddleware(log))

	// Initialize dependencies
	deps := deps.App{
		Fiber:      app,
		Database:   db,
		Logger:     log,
		Middleware: mid,
	}

	// Register handlers
	handler.NewHandler(deps, cfg)

	// Swagger documentation route (accessible without authentication)
	app.Get("/swagger/*", swagger.HandlerDefault)

	ctx, cancel := context.WithCancel(context.Background())
	gErr, gCtx := errgroup.WithContext(ctx)

	// Start server in goroutine
	gErr.Go(func() error {
		log.Info("controller service is running", logger.String("address", cfg.ServerAddr))
		if err := app.Listen(cfg.ServerAddr); err != nil {
			cancel()
			return err
		}
		return nil
	})

	// Shutdown goroutine
	gErr.Go(func() error {
		<-gCtx.Done()

		if err := app.Shutdown(); err != nil {
			log.WithError(err).Error("failed to shutdown fiber app")
			return err
		}

		conn, err := db.DB()
		if err != nil {
			log.WithError(err).Error("failed to get database connection")
			return err
		}
		if err := conn.Close(); err != nil {
			log.WithError(err).Error("failed to close database")
			return err
		}

		return nil
	})

	// Listen for OS signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
		log.Info("listening for shutdown signals")
		<-sigChan
		log.Info("shutdown signal received")
		cancel()
	}()

	if err := gErr.Wait(); err != nil {
		log.WithError(err).Fatal("controller service encountered an error")
	}

	log.Info("controller service stopped gracefully")
}
