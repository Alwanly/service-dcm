package handler

import (
	"github.com/Alwanly/service-distribute-management/internal/server/worker/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"time"
)

type Handler struct {
	Logger  *logger.CanonicalLogger
	UseCase usecase.UseCaseInterface
}

func NewHandler(d deps.App, timeout time.Duration) *Handler {
	repo := repository.NewRepository()
	uc := usecase.NewUseCase(repo, timeout)

	h := &Handler{
		UseCase: uc,
		Logger:  d.Logger,
	}

	// register routes on fiber app
	d.Fiber.Get("/health", h.healthCheck)
	d.Fiber.Post("/config", h.receiveConfig)
	d.Fiber.Post("/hit", h.hit)

	return h
}

func (h *Handler) healthCheck(c *fiber.Ctx) error {
	res := h.UseCase.GetHealthStatus(c.Context())
	return c.Status(res.Code).JSON(res.Data)
}

func (h *Handler) receiveConfig(c *fiber.Ctx) error {
	req := new(dto.ReceiveConfigRequest)
	if err := c.BodyParser(req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		errs := validator.TranslateError(err)
		return c.Status(fiber.StatusBadRequest).JSON(errs)
	}

	res := h.UseCase.ReceiveConfig(c.Context(), req)
	return c.Status(res.Code).JSON(res.Data)
}

func (h *Handler) hit(c *fiber.Ctx) error {
	body := c.Body()

	headers := make(map[string][]string)
	c.Request().Header.VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		headers[key] = append(headers[key], val)
	})

	respBody, status, err := h.UseCase.ProxyRequest(c.Context(), body, headers)
	if err != nil {
		h.Logger.Error("proxy request failed", zap.Error(err))
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(status).Send(respBody)
}
