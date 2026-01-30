package middleware

import (
	"net/http"
	"strings"

	"github.com/Alwanly/service-distribute-management/internal/models"
	"github.com/Alwanly/service-distribute-management/pkg/logger"
	"github.com/Alwanly/service-distribute-management/pkg/wrapper"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const AgentIDContextKey = "agent_id"

func AgentTokenAuth(db *gorm.DB, log *logger.CanonicalLogger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get(fiber.HeaderAuthorization)
		if authHeader == "" {
			log.Debug("missing authorization header",
				zap.String("path", c.Path()),
				zap.String("ip", c.IP()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(wrapper.ResponseFailed(http.StatusUnauthorized, "missing authorization header", nil))
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Debug("malformed authorization header",
				zap.String("path", c.Path()),
				zap.String("header", authHeader),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(wrapper.ResponseFailed(http.StatusUnauthorized, "malformed authorization header", nil))
		}

		token := parts[1]
		if token == "" {
			log.Debug("empty bearer token",
				zap.String("path", c.Path()),
			)
			return c.Status(fiber.StatusUnauthorized).JSON(wrapper.ResponseFailed(http.StatusUnauthorized, "empty bearer token", nil))
		}

		var agent models.AgentConfig
		if err := db.Where("api_token = ?", token).First(&agent).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				log.Debug("invalid api token",
					zap.String("path", c.Path()),
					zap.String("ip", c.IP()),
				)
				return c.Status(fiber.StatusUnauthorized).JSON(wrapper.ResponseFailed(http.StatusUnauthorized, "invalid api token", nil))
			}

			log.Error("database error during token lookup",
				zap.Error(err),
				zap.String("path", c.Path()),
			)
			return c.Status(fiber.StatusInternalServerError).JSON(wrapper.ResponseFailed(http.StatusInternalServerError, "authentication failed", nil))
		}

		c.Locals(AgentIDContextKey, agent.ID)

		log.Debug("agent authenticated",
			zap.String("agent_id", agent.ID),
			zap.String("agent_name", agent.AgentName),
			zap.String("path", c.Path()),
		)

		return c.Next()
	}
}
