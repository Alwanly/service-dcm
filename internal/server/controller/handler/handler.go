package handler

import (
	"github.com/Alwanly/service-distribute-management/internal/config"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/Alwanly/service-distribute-management/pkg/validator"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type Handler struct {
	Logger     *logger.CanonicalLogger
	UseCase    *usecase.UseCase
	Config     *config.ControllerConfig
	Middleware *middleware.AuthMiddleware
}

func NewHandler(d deps.App, cfg *config.ControllerConfig) *Handler {

	repo := repository.NewRepository(d.Database, d.Pub)

	uc := usecase.NewUseCase(usecase.UseCase{
		Repo:   repo,
		Config: cfg,
		Logger: d.Logger,
	})

	h := &Handler{
		Logger:     d.Logger,
		UseCase:    uc,
		Config:     cfg,
		Middleware: d.Middleware,
	}

	// Health check endpoint (no auth required)
	d.Fiber.Get("/health", h.health)

	// Public registration endpoint (agents register without Bearer token)
	d.Fiber.Post("/register", d.Middleware.BasicAuth(), h.register)

	// Admin-protected endpoints
	d.Fiber.Post("/config", d.Middleware.BasicAuthAdmin(), h.setConfig)

	// Agent-authenticated endpoint for fetching configuration
	d.Fiber.Get("/config", middleware.AgentTokenAuth(d.Database, d.Logger), h.getConfig)

	// Agent-authenticated endpoint for sending heartbeat
	d.Fiber.Post("/heartbeat", middleware.AgentTokenAuth(d.Database, d.Logger), h.heartbeat)

	// Management endpoints for agents (admin only)
	adminRoutes := d.Fiber.Group("/agents", d.Middleware.BasicAuthAdmin())
	adminRoutes.Put(":id/interval", h.updateAgentInterval)
	adminRoutes.Post(":id/token/rotate", h.rotateAgentToken)
	adminRoutes.Get("", h.listAgents)
	adminRoutes.Get(":id", h.getAgent)
	adminRoutes.Delete(":id", h.deleteAgent)

	return h
}

// register godoc
// @Summary      Register a new agent
// @Description  Register a new agent with the controller service and receive polling configuration
// @Tags         agents
// @Accept       json
// @Produce      json
// @Param        request body dto.RegisterAgentRequest true "Agent registration details"
// @Success      200 {object} dto.RegisterAgentResponse "Successfully registered agent"
// @Failure      400 {object} wrapper.JSONResult "Invalid request body or validation error"
// @Failure      500 {object} wrapper.JSONResult "Internal server error"
// @Router       /register [post]
// @Security     BasicAuth
func (h *Handler) register(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "register_agent"))

	req := new(dto.RegisterAgentRequest)
	if err := c.BodyParser(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	res := h.UseCase.RegisterAgent(c.UserContext(), req)

	return c.Status(res.Code).JSON(res.Data)
}

// setConfig godoc
// @Summary      Set worker configuration
// @Description  Set new configuration for all workers (admin only). Configuration includes target URL, headers, and timeout settings.
// @Tags         configuration
// @Accept       json
// @Produce      json
// @Param        request body dto.SetConfigAgentRequest true "Configuration data"
// @Success      200 {object} wrapper.JSONResult "Configuration set successfully"
// @Failure      400 {object} wrapper.JSONResult "Invalid request body or validation error"
// @Failure      500 {object} wrapper.JSONResult "Internal server error"
// @Router       /config [post]
// @Security     BasicAuth
func (h *Handler) setConfig(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "set_config"))

	req := new(dto.SetConfigAgentRequest)
	if err := c.BodyParser(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	res := h.UseCase.UpdateConfig(c.UserContext(), req)

	return c.Status(res.Code).JSON(res.Data)
}

// getConfig godoc
// @Summary      Get current worker configuration
// @Description  Retrieve the current configuration that will be distributed to workers
// @Tags         configuration
// @Accept       json
// @Produce      json
// @Param        If-None-Match header string false "ETag for conditional requests"
// @Param        agent_id header string true "Agent ID injected by authentication middleware"
// @Param        Authorization header string true "Bearer token for agent authentication"
// @Success      200 {object} dto.GetConfigAgentResponse "Current configuration data"
// @Failure      500 {object} wrapper.JSONResult "Internal server error"
// @Router       /config [get]
func (h *Handler) getConfig(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "get_config"))

	agentID, ok := c.Locals(middleware.AgentIDContextKey).(string)
	if !ok || agentID == "" {
		h.Logger.Error("agent_id not found in context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "authentication context error"})
	}

	// Get If-None-Match header for ETag comparison
	etag := c.Get("If-None-Match")

	// Get configuration for this agent
	res := h.UseCase.GetConfigForAgent(c.UserContext(), agentID, etag)

	// Handle 304 Not Modified
	if res.Code == fiber.StatusNotModified {
		return c.SendStatus(fiber.StatusNotModified)
	}

	// Set ETag header if present
	if data, ok := res.Data.(dto.GetConfigAgentResponse); ok {
		if data.ETag != "" {
			c.Set("ETag", data.ETag)
		}
	}

	return c.Status(res.Code).JSON(res.Data)
}

// updateAgentInterval handles updating an agent's polling interval
func (h *Handler) updateAgentInterval(c *fiber.Ctx) error {
	agentID := c.Params("id")
	req := new(dto.UpdatePollIntervalRequest)
	if err := c.BodyParser(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.UseCase.UpdateAgentPollInterval(agentID, req.PollIntervalSeconds); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	res := wrapper.ResponseSuccess(fiber.StatusOK, "poll interval updated")
	return c.Status(res.Code).JSON(res.Data)
}

// rotateAgentToken handles rotating an agent's API token
func (h *Handler) rotateAgentToken(c *fiber.Ctx) error {
	agentID := c.Params("id")
	res := h.UseCase.RotateAgentToken(c.UserContext(), agentID)
	return c.Status(res.Code).JSON(res.Data)
}

// getAgent handles retrieving a specific agent
func (h *Handler) getAgent(c *fiber.Ctx) error {
	agentID := c.Params("id")
	res := h.UseCase.GetAgent(c.UserContext(), agentID)
	return c.Status(res.Code).JSON(res.Data)
}

// listAgents handles listing all agents
func (h *Handler) listAgents(c *fiber.Ctx) error {
	res := h.UseCase.ListAgents(c.UserContext())
	return c.Status(res.Code).JSON(res.Data)
}

// deleteAgent handles deleting an agent
func (h *Handler) deleteAgent(c *fiber.Ctx) error {
	agentID := c.Params("id")
	if err := h.UseCase.DeleteAgent(c.UserContext(), agentID); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	res := wrapper.ResponseSuccess(fiber.StatusOK, "agent deleted")
	return c.Status(res.Code).JSON(res.Data)
}

// health godoc
// @Summary     Health check
// @Description Get controller health status (unauthenticated)
// @Tags        health
// @Accept      json
// @Produce     json
// @Success     200 {object} map[string]string
// @Router      /health [get]
func (h *Handler) health(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "health_check"))

	return c.JSON(fiber.Map{"status": "healthy"})
}

// heartbeat godoc
// @Summary      Agent heartbeat
// @Description  Receive periodic heartbeat from agent (authenticated)
// @Tags         agents
// @Accept       json
// @Produce      json
// @Param        request body dto.HeartbeatRequest true "Heartbeat payload"
// @Param        Authorization header string true "Bearer token for agent authentication"
// @Success      200 {object} wrapper.JSONResult "Heartbeat processed"
// @Failure      400 {object} wrapper.JSONResult "Invalid request body"
// @Failure      500 {object} wrapper.JSONResult "Internal server error"
// @Router       /heartbeat [post]
// @Security     ApiKeyAuth
func (h *Handler) heartbeat(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "agent_heartbeat"))

	// Extract agent ID from auth middleware
	agentID, ok := c.Locals(middleware.AgentIDContextKey).(string)
	if !ok || agentID == "" {
		h.Logger.Error("agent_id not found in context for heartbeat")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "authentication context error"})
	}

	req := new(dto.HeartbeatRequest)
	if err := c.BodyParser(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	resp, err := h.UseCase.HandleHeartbeat(agentID, req)
	if err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to process heartbeat"})
	}

	res := wrapper.ResponseSuccess(fiber.StatusOK, resp)
	return c.Status(res.Code).JSON(res.Data)
}
