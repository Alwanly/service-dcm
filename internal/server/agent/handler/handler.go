package handler

import (
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/Alwanly/service-distribute-management/internal/server/agent/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/agent/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
)

// Handler handles HTTP requests for the agent service
type Handler struct {
	useCase usecase.IUseCase
	logger  *logger.CanonicalLogger
}

// NewHandler creates a new agent handler
func NewHandler(uc usecase.IUseCase, log *logger.CanonicalLogger) *Handler {
	return &Handler{useCase: uc, logger: log}
}

// RegisterRoutes registers all agent routes
func (h *Handler) RegisterRoutes(app *fiber.App) {
	app.Get("/health", h.Health)
	app.Get("/status", h.Status)
}

// Health handles the health check endpoint
func (h *Handler) Health(c *fiber.Ctx) error {
	response := dto.HealthResponse{
		Status:    "ok",
		AgentID:   h.useCase.GetAgentID(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	return c.Status(fiber.StatusOK).JSON(wrapper.ResponseSuccess(fiber.StatusOK, response))
}

// Status handles the status endpoint showing agent information
func (h *Handler) Status(c *fiber.Ctx) error {
	status := h.useCase.GetStatus()
	return c.Status(fiber.StatusOK).JSON(wrapper.ResponseSuccess(fiber.StatusOK, status))
}
