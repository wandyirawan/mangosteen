package crown

import (
	"bytes"
	"embed"
	"encoding/json"
	"html/template"
	"log"

	"github.com/gofiber/fiber/v2"
	"mangosteen/internal/auth"
	"mangosteen/internal/user"
)

//go:embed templates/*.gohtml
var templateFS embed.FS

var tmpl *template.Template

func init() {
	var err error
	tmpl = template.New("").Funcs(template.FuncMap{
		"json": func(v interface{}) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"safeHTMLAttr": func(v interface{}) template.HTMLAttr {
			b, _ := json.Marshal(v)
			return template.HTMLAttr(string(b))
		},
	})
	tmpl, err = tmpl.ParseFS(templateFS, "templates/*.gohtml")
	if err != nil {
		log.Fatalf("crown templates parse: %v", err)
	}
}

type Crown struct {
	userService *user.Service
	authService *auth.Service
	jwtManager  *auth.JWTManager
}

func New(userService *user.Service, authService *auth.Service, jwtManager *auth.JWTManager) *Crown {
	return &Crown{
		userService: userService,
		authService: authService,
		jwtManager:  jwtManager,
	}
}

func renderContent(c *fiber.Ctx, contentName string, data fiber.Map) error {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, contentName, data); err != nil {
		return err
	}
	data["Content"] = template.HTML(buf.String())
	c.Set("Content-Type", "text/html; charset=utf-8")
	return tmpl.ExecuteTemplate(c.Response().BodyWriter(), "layout", data)
}

func (cr *Crown) RequireAuth(c *fiber.Ctx) error {
	token := c.Cookies("access_token")
	if token == "" {
		return c.Redirect("/admin/login")
	}
	claims, err := cr.jwtManager.Validate(token)
	if err != nil {
		c.ClearCookie("access_token")
		return c.Redirect("/admin/login")
	}
	if claims["role"] != "admin" {
		return c.Status(403).SendString("Forbidden: admin only")
	}
	c.Locals("userID", claims["sub"])
	c.Locals("email", claims["email"])
	c.Locals("role", claims["role"])
	return c.Next()
}

func (cr *Crown) Dashboard(c *fiber.Ctx) error {
	return c.Redirect("/admin/users")
}

func (cr *Crown) LoginPage(c *fiber.Ctx) error {
	token := c.Cookies("access_token")
	if token != "" {
		if _, err := cr.jwtManager.Validate(token); err == nil {
			return c.Redirect("/admin/users")
		}
	}
	return renderContent(c, "login_content", fiber.Map{
		"PageTitle": "Login",
	})
}

func (cr *Crown) Login(c *fiber.Ctx) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	pair, err := cr.authService.SignIn(c.Context(), auth.LoginDTO{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return renderContent(c, "login_content", fiber.Map{
			"PageTitle": "Login",
			"Error":     "Invalid credentials",
		})
	}

	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    pair.AccessToken,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   pair.ExpiresIn,
	})
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    pair.RefreshToken,
		HTTPOnly: true,
		Secure:   false,
		SameSite: "Lax",
		Path:     "/",
		MaxAge:   7 * 24 * 3600,
	})

	return c.Redirect("/admin/users")
}

func (cr *Crown) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:   "access_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	c.Cookie(&fiber.Cookie{
		Name:   "refresh_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return c.Redirect("/admin/login")
}

func (cr *Crown) UsersList(c *fiber.Ctx) error {
	users, err := cr.userService.ListAllUsers(c.Context())
	if err != nil {
		return c.Status(500).SendString("Failed to load users")
	}

	return renderContent(c, "users_list_content", fiber.Map{
		"PageTitle": "Users",
		"Users":     users,
	})
}

func (cr *Crown) UserDetail(c *fiber.Ctx) error {
	userID := c.Params("id")
	u, err := cr.userService.GetByID(c.Context(), userID)
	if err != nil {
		return c.Status(404).SendString("User not found")
	}

	return renderContent(c, "user_detail_content", fiber.Map{
		"PageTitle":  "User Detail",
		"User":       u,
	})
}
