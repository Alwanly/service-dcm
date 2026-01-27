package deps

import (
	"database/sql"

	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/middleware"
	"github.com/gofiber/fiber/v2"
)

type App struct {
	Fiber      *fiber.App
	Logger     *logger.CanonicalLogger
	Database   *sql.DB
	Middleware *middleware.AuthMiddleware
}
