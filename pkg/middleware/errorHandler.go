package middleware

import (
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(log *logger.CanonicalLogger) fiber.ErrorHandler {
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
