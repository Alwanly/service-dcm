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

	repo := repository.NewRepository(d.Database)

	uc := usecase.NewUseCase(usecase.UseCase{
		Repo:   repo,
		Config: cfg,
		Logger: d.Logger,
	})

	h := &Handler{
		UseCase: uc,
	}

	d.Fiber.Post("/register", d.Middleware.BasicAuth(), h.register)
	d.Fiber.Post("/config", d.Middleware.BasicAuthAdmin(), h.setConfig)
	d.Fiber.Get("/config", d.Middleware.BasicAuth(), h.getConfig)

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
	// Enrich log context
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
	// Enrich log context
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
// @Success      200 {object} dto.GetConfigAgentResponse "Current configuration data"
// @Failure      500 {object} wrapper.JSONResult "Internal server error"
// @Router       /config [get]
// @Security     BasicAuth
func (h *Handler) getConfig(c *fiber.Ctx) error {
	// Enrich log context
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "get_config"))

	req := new(dto.GetConfigAgentRequest)

	// get ETag from header
	headers := c.GetReqHeaders()
	if etag, exists := headers["If-None-Match"]; exists {
		req.ETag = etag[0]
	}

	res := h.UseCase.GetConfig(c.UserContext(), req)
	return c.Status(res.Code).JSON(res.Data)
}
