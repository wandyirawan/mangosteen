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

func (h *Handler) RegisterRoutes(app fiber.Router) {}

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

func (h *Handler) RegisterAdminRoutes(app fiber.Router) {
	app.Get("/", h.ListUsers)
	app.Get("/all", h.ListAllUsers)
	app.Get("/:id", h.GetUser)
	app.Patch("/:id", h.UpdateUser)
	app.Patch("/:id/role", h.ChangeRole)
	app.Post("/:id/activate", h.ActivateUser)
	app.Delete("/:id", h.DeleteUser)
	app.Get("/:id/attributes", h.GetUserAttributes)
	app.Put("/:id/attributes", h.SetUserAttributes)
	app.Delete("/:id/attributes/:key", h.DeleteUserAttribute)
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

func (h *Handler) GetMyAttributes(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	attrs, err := h.service.GetAttributes(c.Context(), userID.(string))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"attributes": attrs})
}

func (h *Handler) SetMyAttributes(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	var dto SetAttributesDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.service.SetAttributes(c.Context(), userID.(string), dto.Attributes); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "attributes updated"})
}

func (h *Handler) DeleteMyAttribute(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	key := c.Params("key")
	if err := h.service.DeleteAttribute(c.Context(), userID.(string), key); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "attribute deleted"})
}

func (h *Handler) GetUserAttributes(c *fiber.Ctx) error {
	id := c.Params("id")
	attrs, err := h.service.GetAttributes(c.Context(), id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"attributes": attrs})
}

func (h *Handler) SetUserAttributes(c *fiber.Ctx) error {
	id := c.Params("id")

	var dto SetAttributesDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	if err := h.service.SetAttributes(c.Context(), id, dto.Attributes); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "attributes updated"})
}

func (h *Handler) DeleteUserAttribute(c *fiber.Ctx) error {
	id := c.Params("id")
	key := c.Params("key")

	if err := h.service.DeleteAttribute(c.Context(), id, key); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "attribute deleted"})
}