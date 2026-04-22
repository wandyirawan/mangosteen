package auth

import "github.com/gofiber/fiber/v2"

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