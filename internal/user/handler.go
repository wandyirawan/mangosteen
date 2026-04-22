package user

import (
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(app fiber.Router) {
	users := app.Group("/users")

	users.Get("/me", h.GetMe)
	users.Patch("/me", h.UpdateMe)
	users.Delete("/me", h.DeactivateMe)

	users.Get("/", h.ListUsers)
	users.Get("/all", h.ListAllUsers)
	users.Get("/:id", h.GetUser)
	users.Patch("/:id", h.UpdateUser)
	users.Patch("/:id/role", h.ChangeRole)
	users.Post("/:id/activate", h.ActivateUser)
	users.Delete("/:id", h.DeleteUser)
}

func (h *Handler) GetMe(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	user, err := h.service.GetByID(c.Context(), userID.(string))
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(user)
}

func (h *Handler) UpdateMe(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.service.UpdateUser(c.Context(), userID.(string), updates); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "updated"})
}

func (h *Handler) DeactivateMe(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	if err := h.service.SoftDelete(c.Context(), userID.(string)); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "deactivated"})
}

func (h *Handler) ListUsers(c *fiber.Ctx) error {
	users, err := h.service.ListActiveUsers(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"count": len(users),
	})
}

func (h *Handler) ListAllUsers(c *fiber.Ctx) error {
	users, err := h.service.ListAllUsers(c.Context())
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"users": users,
		"count": len(users),
	})
}

func (h *Handler) GetUser(c *fiber.Ctx) error {
	id := c.Params("id")
	user, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(user)
}

func (h *Handler) UpdateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.service.UpdateUser(c.Context(), id, updates); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "updated"})
}

func (h *Handler) ChangeRole(c *fiber.Ctx) error {
	id := c.Params("id")

	var dto struct {
		Role string `json:"role"`
	}
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if dto.Role != "admin" && dto.Role != "user" {
		return c.Status(400).JSON(fiber.Map{"error": "role must be admin or user"})
	}

	err := h.service.UpdateUser(c.Context(), id, map[string]interface{}{"role": dto.Role})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "role updated"})
}

func (h *Handler) ActivateUser(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.service.ActivateUser(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "user activated"})
}

func (h *Handler) DeleteUser(c *fiber.Ctx) error {
	id := c.Params("id")

	err := h.service.HardDelete(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "user deleted"})
}