package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

// AuthMiddleware provides mock Bearer token authentication.
// Any non-empty Bearer token is considered valid.
// Public paths (health, swagger) bypass authentication.
func AuthMiddleware() fiber.Handler {
	publicPrefixes := []string{"/health", "/swagger"}

	return func(c fiber.Ctx) error {
		path := c.Path()

		// Skip auth for public paths
		for _, prefix := range publicPrefixes {
			if strings.HasPrefix(path, prefix) {
				return c.Next()
			}
		}

		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing Authorization header",
			})
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid Authorization header format, expected 'Bearer <token>'",
			})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "empty bearer token",
			})
		}

		// Mock validation: accept any non-empty token
		// In production, validate JWT or call an auth service here
		c.Locals("auth_token", token)

		return c.Next()
	}
}
