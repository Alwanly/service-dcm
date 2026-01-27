package handler

import (
	"github.com/Alwanly/service-distribute-management/internal/server/controller/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/controller/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/Alwanly/service-distribute-management/pkg/validator"
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	Logger     *logger.CanonicalLogger
	UseCase    usecase.UseCaseInterface
	Middleware *middleware.AuthMiddleware
}

func NewHandler(d deps.App) *Handler {

	repo := repository.NewRepository(d.Database)
	uc := usecase.NewUseCase(repo)

	h := &Handler{
		UseCase: uc,
	}

	e := d.Fiber.Group("controller")

	e.Post("/register", d.Middleware.BasicAuth(), h.register)
	e.Post("/config", d.Middleware.BasicAuthAdmin(), h.setConfig)
	e.Get("/config", d.Middleware.BasicAuth(), h.getConfig)

	return h
}

func (h *Handler) register(c *fiber.Ctx) error {

	req := new(dto.RegisterAgentRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	res := h.UseCase.RegisterAgent(c.Context(), req)

	return c.Status(res.Code).JSON(res.Data)
}

func (h *Handler) setConfig(c *fiber.Ctx) error {

	req := new(dto.SetConfigAgentRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	res := h.UseCase.SetConfigAgent(c.Context(), req)

	return c.Status(res.Code).JSON(res.Data)
}

func (h *Handler) getConfig(c *fiber.Ctx) error {

	res := h.UseCase.GetConfigAgent(c.Context())
	return c.Status(res.Code).JSON(res.Data)
}
