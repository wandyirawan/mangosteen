package auth

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mangosteen/config"
	"mangosteen/internal/db"
	"mangosteen/pkg/cache"
)

func setupTestHandler(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema, err := os.ReadFile("../../sql/schema.sql")
	require.NoError(t, err)
	_, err = database.Exec(string(schema))
	require.NoError(t, err)

	privPEM, pubPEM := generateTestRSAKey(t)
	cfg := &config.JWTConfig{
		PrivateKeyPEM: privPEM,
		PublicKeyPEM:  pubPEM,
		Issuer:        "test-issuer",
		AccessTTL:     15,
		RefreshTTL:    7,
	}
	jwtMgr, err := NewJWTManager(cfg)
	require.NoError(t, err)

	var testDB db.DB
	testDB.DB = database
	repo := NewRepository(&testDB)
	valkey := cache.NewValkeyClient("", "", 0)
	svc := NewService(repo, valkey, jwtMgr)

	handler := NewHandler(svc, jwtMgr)

	app := fiber.New()
	handler.RegisterRoutes(app)

	return app, database
}

type jwksResponse struct {
	Keys []jwkResponse `json:"keys"`
}

type jwkResponse struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func TestHandler_OpenIDConfig(t *testing.T) {
	app, sqlDB := setupTestHandler(t)
	defer sqlDB.Close()

	req, _ := http.NewRequest("GET", "/.well-known/openid-configuration", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]string
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "mangosteen", body["issuer"])
	assert.NotEmpty(t, body["jwks_uri"])
}

func TestHandler_JWKS(t *testing.T) {
	app, sqlDB := setupTestHandler(t)
	defer sqlDB.Close()

	req, _ := http.NewRequest("GET", "/.well-known/jwks.json", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var jwks jwksResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&jwks))
	assert.NotEmpty(t, jwks.Keys)
	assert.Equal(t, "RSA", jwks.Keys[0].Kty)
}

func TestHandler_Register(t *testing.T) {
	app, sqlDB := setupTestHandler(t)
	defer sqlDB.Close()

	t.Run("successful registration", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email":"handler@example.com","password":"password123"}`)
		req, _ := http.NewRequest("POST", "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 201, resp.StatusCode)

		var user UserResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&user))
		assert.Equal(t, "handler@example.com", user.Email)
		assert.Equal(t, "user", user.Role)
	})

	t.Run("duplicate email", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email":"handler@example.com","password":"password123"}`)
		req, _ := http.NewRequest("POST", "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})

	t.Run("invalid body", func(t *testing.T) {
		body := bytes.NewBufferString(`not-json`)
		req, _ := http.NewRequest("POST", "/auth/register", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestHandler_Login(t *testing.T) {
	app, sqlDB := setupTestHandler(t)
	defer sqlDB.Close()

	regBody := bytes.NewBufferString(`{"email":"login@example.com","password":"password123"}`)
	req, _ := http.NewRequest("POST", "/auth/register", regBody)
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	require.NoError(t, err)

	t.Run("valid login", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email":"login@example.com","password":"password123"}`)
		req, _ := http.NewRequest("POST", "/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var tokens TokenPair
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&tokens))
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, "Bearer", tokens.Type)
	})

	t.Run("wrong password", func(t *testing.T) {
		body := bytes.NewBufferString(`{"email":"login@example.com","password":"wrongpass"}`)
		req, _ := http.NewRequest("POST", "/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("invalid body", func(t *testing.T) {
		body := bytes.NewBufferString(`bad json`)
		req, _ := http.NewRequest("POST", "/auth/login", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestHandler_Refresh(t *testing.T) {
	app, sqlDB := setupTestHandler(t)
	defer sqlDB.Close()

	regBody := bytes.NewBufferString(`{"email":"refresh@example.com","password":"password123"}`)
	req, _ := http.NewRequest("POST", "/auth/register", regBody)
	req.Header.Set("Content-Type", "application/json")
	_, err := app.Test(req)
	require.NoError(t, err)

	t.Run("invalid refresh token", func(t *testing.T) {
		body := bytes.NewBufferString(`{"refresh_token":"bad.token"}`)
		req, _ := http.NewRequest("POST", "/auth/refresh", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("invalid body", func(t *testing.T) {
		body := bytes.NewBufferString(`bad`)
		req, _ := http.NewRequest("POST", "/auth/refresh", body)
		req.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 400, resp.StatusCode)
	})
}

func TestHandler_Logout(t *testing.T) {
	app, sqlDB := setupTestHandler(t)
	defer sqlDB.Close()

	body := bytes.NewBufferString(`{"email":"logout@example.com","password":"password123"}`)
	req, _ := http.NewRequest("POST", "/auth/register", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)

	resp.Body.Close()

	t.Run("requires auth", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/auth/logout", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})
}
