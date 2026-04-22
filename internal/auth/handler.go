package auth

import (
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service *Service
	jwt     *JWTManager
}

func NewHandler(service *Service, jwt *JWTManager) *Handler {
	return &Handler{service: service, jwt: jwt}
}

func (h *Handler) RegisterRoutes(app fiber.Router) {
	auth := app.Group("/auth")
	auth.Post("/login", h.Login)
	auth.Post("/register", h.Register)
	auth.Post("/refresh", h.Refresh)
	auth.Post("/logout", h.Logout)

	app.Get("/.well-known/openid-configuration", h.OpenIDConfig)
	app.Get("/.well-known/jwks.json", h.JWKS)
}

func (h *Handler) OpenIDConfig(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"issuer":                 "mangosteen",
		"authorization_endpoint": "/api/auth/login",
		"token_endpoint":        "/api/auth/refresh",
		"jwks_uri":             "/.well-known/jwks.json",
	})
}

func (h *Handler) JWKS(c *fiber.Ctx) error {
	if h.jwt == nil {
		return c.JSON(fiber.Map{"keys": []fiber.Map{}})
	}
	return c.JSON(h.jwt.GetJWKS())
}

func (h *Handler) Login(c *fiber.Ctx) error {
	var dto LoginDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	tokens, err := h.service.SignIn(c.Context(), dto)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(tokens)
}

func (h *Handler) Register(c *fiber.Ctx) error {
	var dto RegisterDTO
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	user, err := h.service.SignUp(c.Context(), dto)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(201).JSON(user)
}

func (h *Handler) Refresh(c *fiber.Ctx) error {
	var dto struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&dto); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid request body"})
	}

	tokens, err := h.service.RefreshToken(c.Context(), dto.RefreshToken)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{"error": "invalid or expired refresh token"})
	}

	return c.JSON(tokens)
}

func (h *Handler) Logout(c *fiber.Ctx) error {
	userID := c.Locals("userID")
	if userID == nil {
		return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
	}

	tokenID := c.Locals("tokenID")
	if tokenID != nil {
		h.service.Logout(c.Context(), userID.(string), tokenID.(string))
	}

	return c.JSON(fiber.Map{"message": "logged out"})
}