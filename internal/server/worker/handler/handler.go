package handler

import (
	"time"

	"github.com/Alwanly/service-distribute-management/internal/server/worker/dto"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/repository"
	"github.com/Alwanly/service-distribute-management/internal/server/worker/usecase"
	"github.com/Alwanly/service-distribute-management/pkg/deps"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
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

// healthCheck godoc
// @Summary      Worker health check
// @Description  Check worker service health status and current configuration state
// @Tags         health
// @Accept       json
// @Produce      json
// @Success      200 {object} dto.HealthCheckResponse "Worker is healthy"
// @Router       /health [get]
func (h *Handler) healthCheck(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "health_check"))
	res := h.UseCase.GetHealthStatus(c.UserContext())
	return c.Status(res.Code).JSON(res.Data)
}

// receiveConfig godoc
// @Summary      Receive configuration update
// @Description  Receive and apply new configuration from the agent service. Configuration includes target URL, headers, and timeout.
// @Tags         configuration
// @Accept       json
// @Produce      json
// @Param        request body dto.ReceiveConfigRequest true "Configuration data"
// @Success      200 {object} dto.ReceiveConfigResponse "Configuration updated successfully"
// @Failure      400 {object} map[string]string "Invalid request body or validation error"
// @Router       /config [post]
func (h *Handler) receiveConfig(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "receive_config"))

	req := new(dto.ReceiveConfigRequest)
	if err := c.BodyParser(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.ValidateStruct(req); err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		errs := validator.TranslateError(err)
		return c.Status(fiber.StatusBadRequest).JSON(errs)
	}

	res := h.UseCase.ReceiveConfig(c.UserContext(), req)
	return c.Status(res.Code).JSON(res.Data)
}

// hit godoc
// @Summary      Proxy request to target URL
// @Description  Forward incoming request to the configured target URL with configured headers. Returns proxied response.
// @Tags         proxy
// @Accept       */*
// @Produce      */*
// @Param        body body string false "Request body to forward"
// @Success      200 {object} map[string]interface{} "Proxied response from target URL"
// @Failure      503 {object} map[string]string "Worker not configured"
// @Failure      502 {object} map[string]string "Proxy request failed"
// @Router       /hit [post]
func (h *Handler) hit(c *fiber.Ctx) error {
	logger.AddToContext(c.UserContext(), logger.String(logger.FieldOperation, "proxy_request"))

	body := c.Body()

	headers := make(map[string][]string)
	c.Request().Header.VisitAll(func(k, v []byte) {
		key := string(k)
		val := string(v)
		headers[key] = append(headers[key], val)
	})

	respBody, status, err := h.UseCase.ProxyRequest(c.UserContext(), body, headers)
	if err != nil {
		logger.AddToContext(c.UserContext(), zap.Error(err))
		return c.Status(status).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(status).Send(respBody)
}
