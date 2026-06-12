package middleware

import (
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireRole(t *testing.T) {
	t.Run("no role in context returns 403", func(t *testing.T) {
		app := fiber.New()
		app.Use(RequireRole("admin"))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("matching role passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "admin")
			return c.Next()
		})
		app.Use(RequireRole("admin"))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("non-matching role returns 403", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "user")
			return c.Next()
		})
		app.Use(RequireRole("admin"))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("admin accessing admin-only passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "admin")
			return c.Next()
		})
		app.Use(RequireAdmin())
		app.Get("/admin/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "admin success"})
		})

		req, _ := http.NewRequest("GET", "/admin/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("multiple allowed roles - any match passes", func(t *testing.T) {
		app := fiber.New()

		// Test with "user" role
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "user")
			return c.Next()
		})
		app.Use(RequireRole("admin", "user"))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("multiple allowed roles - admin match passes", func(t *testing.T) {
		app := fiber.New()

		// Test with "admin" role
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "admin")
			return c.Next()
		})
		app.Use(RequireRole("admin", "user"))
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestRequireAdmin(t *testing.T) {
	t.Run("non-admin role returns 403", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "user")
			return c.Next()
		})
		app.Use(RequireAdmin())
		app.Get("/admin/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/admin/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})

	t.Run("admin role passes", func(t *testing.T) {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error {
			c.Locals("role", "admin")
			return c.Next()
		})
		app.Use(RequireAdmin())
		app.Get("/admin/test", func(c *fiber.Ctx) error {
			return c.JSON(fiber.Map{"message": "success"})
		})

		req, _ := http.NewRequest("GET", "/admin/test", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}
