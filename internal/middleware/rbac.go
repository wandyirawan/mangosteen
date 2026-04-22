package middleware

import "github.com/gofiber/fiber/v2"

func RequireRole(roles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userRole := c.Locals("role").(string)

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
