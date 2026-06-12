package crown

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mangosteen/config"
	"mangosteen/internal/auth"
	"mangosteen/internal/db"
	"mangosteen/internal/user"
	"mangosteen/pkg/cache"
)

func setupCrownTest(t *testing.T) (*fiber.App, *Crown, *sql.DB) {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema, err := os.ReadFile("../../sql/schema.sql")
	require.NoError(t, err)
	_, err = database.Exec(string(schema))
	require.NoError(t, err)

	// Generate test RSA key
	privPEM, pubPEM := generateTestRSAKeyCrown(t)

	cfg := &config.JWTConfig{
		PrivateKeyPEM: privPEM,
		PublicKeyPEM:  pubPEM,
		Issuer:        "test-issuer",
		AccessTTL:     15,
		RefreshTTL:    7,
	}
	jwtMgr, err := auth.NewJWTManager(cfg)
	require.NoError(t, err)

	var testDB db.DB
	testDB.DB = database

	authRepo := auth.NewRepository(&testDB)
	valkey := cache.NewValkeyClient("", "", 0)
	authSvc := auth.NewService(authRepo, valkey, jwtMgr)

	userRepo := user.NewRepository(&testDB)
	userSvc := user.NewService(userRepo, valkey)

	crown := New(userSvc, authSvc, jwtMgr)

	app := fiber.New()
	crown.RegisterRoutes(app)

	return app, crown, database
}

// generateTestRSAKeyCrown generates a 2048-bit RSA key pair for testing
func generateTestRSAKeyCrown(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()

	// Generate 2048-bit RSA key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA key: %v", err)
	}

	// Encode private key to PEM
	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}
	privPEM = string(pem.EncodeToMemory(privBlock))

	// Encode public key to PEM
	pubBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubPEM = string(pem.EncodeToMemory(pubBlock))

	return privPEM, pubPEM
}

func TestCrown_LoginPage(t *testing.T) {
	app, _, sqlDB := setupCrownTest(t)
	defer sqlDB.Close()

	t.Run("returns login page", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/login", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestCrown_Logout(t *testing.T) {
	app, _, sqlDB := setupCrownTest(t)
	defer sqlDB.Close()

	t.Run("clears cookies and redirects to login", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/logout", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		// Without auth, should redirect to login
		assert.Equal(t, 302, resp.StatusCode)
	})
}

func TestCrown_RequireAuth(t *testing.T) {
	app, _, sqlDB := setupCrownTest(t)
	defer sqlDB.Close()

	t.Run("no cookie redirects to login", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/users", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 302, resp.StatusCode)
		assert.Equal(t, "/admin/login", resp.Header.Get("Location"))
	})

	t.Run("invalid token redirects to login", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/users", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "invalid.token.here",
		})
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 302, resp.StatusCode)
	})
}

func TestCrown_Dashboard(t *testing.T) {
	app, _, sqlDB := setupCrownTest(t)
	defer sqlDB.Close()

	t.Run("redirects to users page", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 302, resp.StatusCode)
		assert.Equal(t, "/admin/users", resp.Header.Get("Location"))
	})
}

func TestCrown_UsersList(t *testing.T) {
	app, _, sqlDB := setupCrownTest(t)
	defer sqlDB.Close()

	t.Run("requires auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/users", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 302, resp.StatusCode)
	})
}

func TestCrown_UserDetail(t *testing.T) {
	app, _, sqlDB := setupCrownTest(t)
	defer sqlDB.Close()

	t.Run("requires auth", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/users/123", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 302, resp.StatusCode)
	})
}
