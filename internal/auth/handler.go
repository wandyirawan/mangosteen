package auth

import "github.com/gofiber/fiber/v2"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(app fiber.Router) {
	auth := app.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/register", h.Register)
	auth.Post("/refresh", h.Refresh)
	auth.Post("/logout", h.Logout)
}

func (h *Handler) Login(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) Register(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}
