package deps

import (
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

type App struct {
	Fiber      *fiber.App
	Logger     *logger.CanonicalLogger
	Database   *gorm.DB
	Middleware *middleware.AuthMiddleware
}
