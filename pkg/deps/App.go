package deps

import (
	"database/sql"

	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/gofiber/fiber/v2"
)

type App struct {
	Middleware *middleware.AuthMiddleware
	Logger     *logger.CanonicalLogger
	Database   *sql.DB
	Fiber      *fiber.App
}
