package worker

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
)

// WorkerServer handles HTTP endpoints for the Worker service
type WorkerServer struct {
	currentConfig *models.WorkerConfiguration
	httpClient    *http.Client
	logger        *logger.CanonicalLogger
	configMutex   sync.RWMutex
}

// NewWorkerServer creates a new WorkerServer
func NewWorkerServer(requestTimeout time.Duration) *WorkerServer {
	log, _ := logger.NewLoggerFromEnv("worker")

	return &WorkerServer{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		logger: log.Component("http-server"),
	}
}

// SetupRoutes registers all routes on the Fiber app
func (w *WorkerServer) SetupRoutes(app *fiber.App) {
	app.Get("/health", w.healthCheckHandler)
	app.Post("/hit", w.hitHandler)
	app.Post("/config", w.receiveConfigHandler)
}

// GetCurrentConfig returns the current configuration (thread-safe)
func (w *WorkerServer) GetCurrentConfig() *models.WorkerConfiguration {
	w.configMutex.RLock()
	defer w.configMutex.RUnlock()
	return w.currentConfig
}

// UpdateConfig updates the current configuration (thread-safe)
func (w *WorkerServer) UpdateConfig(config *models.WorkerConfiguration) {
	w.configMutex.Lock()
	defer w.configMutex.Unlock()
	w.currentConfig = config

	w.logger.WithConfigVersion(config.Version).Info("configuration updated",
		logger.String("target_url", config.TargetURL),
	)
}

// healthCheckHandler returns service health status
func (w *WorkerServer) healthCheckHandler(c *fiber.Ctx) error {
	config := w.GetCurrentConfig()

	hasConfig := config != nil
	var targetURL string
	var configVersion int64

	if hasConfig {
		targetURL = config.TargetURL
		configVersion = config.Version
	}

	return c.JSON(fiber.Map{
		"status":         "healthy",
		"service":        "worker",
		"has_config":     hasConfig,
		"target_url":     targetURL,
		"config_version": configVersion,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}

// receiveConfigHandler receives configuration updates from Agent
func (w *WorkerServer) receiveConfigHandler(c *fiber.Ctx) error {
	var config models.WorkerConfiguration
	if err := c.BodyParser(&config); err != nil {
		w.logger.WithError(err).Error("failed to parse configuration")
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Failed to parse configuration",
		})
	}

	w.UpdateConfig(&config)

	return c.JSON(fiber.Map{
		"status":  "configuration updated",
		"version": config.Version,
	})
}

// hitHandler proxies requests to the configured target URL
func (w *WorkerServer) hitHandler(c *fiber.Ctx) error {
	config := w.GetCurrentConfig()

	if config == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(models.ErrorResponse{
			Error:   "no_configuration",
			Message: "No configuration available",
		})
	}

	// Create proxy request
	req, err := http.NewRequestWithContext(
		c.Context(),
		http.MethodPost,
		config.TargetURL,
		bytes.NewReader(c.Body()),
	)
	if err != nil {
		w.logger.WithError(err).Error("failed to create proxy request")
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "proxy_error",
			Message: "Failed to create proxy request",
		})
	}

	// Copy headers from configuration
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Copy content type from original request
	if contentType := c.Get("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Execute request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		w.logger.WithError(err).Error("proxy request failed")
		return c.Status(fiber.StatusBadGateway).JSON(models.ErrorResponse{
			Error:   "proxy_error",
			Message: "Proxy request failed",
		})
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.logger.WithError(err).Error("failed to read proxy response")
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error:   "proxy_error",
			Message: "Failed to read proxy response",
		})
	}

	w.logger.Info("proxy request successful",
		logger.String("target_url", config.TargetURL),
		logger.Int("status", resp.StatusCode),
	)

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			c.Set(key, value)
		}
	}

	return c.Status(resp.StatusCode).Send(body)
}
