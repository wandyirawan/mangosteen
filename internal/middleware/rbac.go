package middleware

import "github.com/gofiber/fiber/v2"

func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		roleVal := c.Locals("role")
		if roleVal == nil {
			return c.Status(403).JSON(fiber.Map{"error": "insufficient permissions"})
		}

		userRole, ok := roleVal.(string)
		if !ok {
			return c.Status(403).JSON(fiber.Map{"error": "insufficient permissions"})
		}

		for _, role := range roles {
			if userRole == role {
				return c.Next()
			}
		}

		return c.Status(403).JSON(fiber.Map{"error": "insufficient permissions"})
	}
}

func RequireAdmin() fiber.Handler {
	return RequireRole("admin")
}

func RequireOwnerOrAdmin() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// TODO: check if user is accessing own resource or is admin
		return c.Next()
	}
}
