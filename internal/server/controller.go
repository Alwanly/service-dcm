package server

import (
	"github.com/gofiber/fiber/v2"

	"github.com/Alwanly/service-distribute-management/internal/store"
)

// ControllerServer handles controller HTTP endpoints (minimal stub)
type ControllerServer struct {
	db                 *store.DB
	pollIntervalSecond int
	extra              string
}

// NewControllerServer creates a new ControllerServer
func NewControllerServer(db *store.DB, pollIntervalSeconds int, extra string) *ControllerServer {
	return &ControllerServer{db: db, pollIntervalSecond: pollIntervalSeconds, extra: extra}
}

// SetupRoutes registers routes for controller (minimal)
func (s *ControllerServer) SetupRoutes(app *fiber.App) {
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy", "service": "controller"})
	})

	// Admin config endpoint (minimal stub)
	app.Post("/admin/config", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
