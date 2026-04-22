package user

import "github.com/gofiber/fiber/v2"

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(app fiber.Router) {
	users := app.Group("/users")

	// Public/Self
	users.Get("/me", h.GetMe)
	users.Patch("/me", h.UpdateMe)
	users.Delete("/me", h.DeactivateMe)

	// Admin only
	users.Get("/", h.ListUsers)
	users.Get("/all", h.ListAllUsers)
	users.Get("/:id", h.GetUser)
	users.Patch("/:id", h.UpdateUser)
	users.Put("/:id/role", h.ChangeRole)
	users.Post("/:id/activate", h.ActivateUser)
	users.Delete("/:id", h.DeleteUser)
	users.Get("/:id/sessions", h.GetUserSessions)
	users.Delete("/:id/sessions", h.ForceLogout)
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) UpdateMe(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) DeactivateMe(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) ListAllUsers(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) ChangeRole(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) ActivateUser(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) GetUserSessions(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}

func (h *Handler) ForceLogout(c *fiber.Ctx) error {
	return c.Status(501).JSON(fiber.Map{"message": "not implemented"})
}
