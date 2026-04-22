package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"mangosteen/internal/auth"
)

type AuthMiddleware struct {
	jwt *auth.JWTManager
}

func NewAuthMiddleware(jwt *auth.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{jwt: jwt}
}

func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(401).JSON(fiber.Map{"error": "invalid authorization format"})
		}

		token := parts[1]
		claims, err := m.jwt.Validate(token)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals("userID", claims["sub"])
		c.Locals("role", claims["role"])

		return c.Next()
	}
}

func (m *AuthMiddleware) RequireAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role == nil || role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "admin access required"})
		}
		return c.Next()
	}
}

func (m *AuthMiddleware) OptionalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return c.Next()
	}
}