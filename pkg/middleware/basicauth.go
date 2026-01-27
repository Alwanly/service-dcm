package middleware

import (
	"net/http"
	"strings"

	authentication "github.com/Alwanly/service-distribute-management/pkg/auth"
	"github.com/gofiber/fiber/v2"
)

type IAuthMiddleware interface {
	// Jwt token
	JwtAuth() fiber.Handler

	// Basic Auth
	BasicAuth() fiber.Handler

	// Basic Auth Admin
	BasicAuthAdmin() fiber.Handler
}

type AuthMiddleware struct {
	Basic authentication.IBasicAuthService
}

// mockery:ignore
type AuthConfig func(*AuthOpts)

type AuthUserData struct {
	UserID string `json:"userId"`
}

type AuthOpts struct {
	*authentication.BasicAuthTConfig
}

func SetBasicAuth(basicAuthConfig *authentication.BasicAuthTConfig) AuthConfig {
	return func(o *AuthOpts) {
		o.BasicAuthTConfig = basicAuthConfig
	}
}

func NewAuthMiddleware(opts ...AuthConfig) *AuthMiddleware {
	var o AuthOpts
	for _, opt := range opts {
		opt(&o)
	}

	basicAuth := authentication.NewBasicAuthService(o.BasicAuthTConfig)

	return &AuthMiddleware{
		Basic: basicAuth,
	}
}

func (a *AuthMiddleware) BasicAuth() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// get auth from header
		auth := ctx.Get(fiber.HeaderAuthorization)
		if !strings.Contains(auth, "Basic") {
			return responseUnauthorized(ctx, "Basic", "Invalid auth")
		}

		// decode auth
		username, password := a.Basic.DecodeFromHeader(auth)
		if !a.Basic.Validate(username, password) {
			return responseUnauthorized(ctx, "Basic", "Invalid auth")
		}
		return ctx.Next()
	}
}

func (a *AuthMiddleware) BasicAuthAdmin() fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		// get auth from header
		auth := ctx.Get(fiber.HeaderAuthorization)
		if !strings.Contains(auth, "Basic") {
			return responseUnauthorized(ctx, "Basic", "Invalid auth")
		}

		// decode auth
		username, password := a.Basic.DecodeFromHeader(auth)
		if !a.Basic.ValidateAdmin(username, password) {
			return responseUnauthorized(ctx, "Basic", "Invalid auth")
		}
		return ctx.Next()
	}
}

func responseUnauthorized(c *fiber.Ctx, _ string, message ...string) error {
	c.Set("WWW-Authenticate", "Basic realm=Restricted")
	response := fiber.Map{
		"message": message[0],
	}
	if len(message) > 1 {
		response["statusCode"] = message[1]
	}
	return c.Status(http.StatusUnauthorized).JSON(response)
}
