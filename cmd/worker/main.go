package main

// @title           Service Distribute Management - Worker API
// @version         1.0
// @description     Worker service for distributed configuration management system. Receives configuration from agents and proxies requests to target URLs.
// @termsOfService  http://swagger.io/terms/
// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @host      localhost:8082
// @BasePath  /

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	_ "github.com/Alwanly/service-distribute-management/docs/worker"
	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/handler"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	swagger "github.com/gofiber/swagger"
)

func main() {
	log, err := logger.NewLoggerFromEnv("worker")
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	log.Info("Starting Worker Service...")

	cfg, err := config.LoadWorkerConfig()
	if err != nil {
		log.Fatal("Failed to load configuration")
	}

	log.Info("configuration loaded",
		logger.String("server_addr", cfg.ServerAddr),
		logger.Duration("request_timeout", cfg.RequestTimeout),
	)

	app := fiber.New(fiber.Config{
		AppName:               "Worker Service",
		DisableStartupMessage: true,
		ErrorHandler:          middleware.ErrorHandler(log),
	})

	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(middleware.CanonicalLoggerMiddleware(log))

	dependencies := deps.App{
		Fiber:  app,
		Logger: log,
	}

	handler.NewHandler(dependencies, cfg.RequestTimeout)

	app.Get("/swagger/*", swagger.HandlerDefault)

	log.Info("Worker Service configured",
		logger.String("addr", cfg.ServerAddr),
		logger.Duration("request_timeout", cfg.RequestTimeout),
	)

	go func() {
		addr := cfg.ServerAddr
		log.Info("Worker Service starting", logger.String("address", addr))
		if err := app.Listen(addr); err != nil {
			log.Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down Worker Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Error("Server forced to shutdown")
	}

	log.Info("Worker Service stopped")
}
