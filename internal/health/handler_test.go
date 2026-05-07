package health

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mangosteen/internal/db"
	"mangosteen/pkg/cache"
)

func setupHealthHandler(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema, err := os.ReadFile("../../sql/schema.sql")
	require.NoError(t, err)
	_, err = database.Exec(string(schema))
	require.NoError(t, err)

	valkeyClient := cache.NewValkeyClient("", "", 0)

	var testDB db.DB
	testDB.DB = database
	handler := NewHandler(&testDB, valkeyClient)

	app := fiber.New()
	handler.RegisterRoutes(app)

	return app, database
}

func TestHandler_Live(t *testing.T) {
	app, sqlDB := setupHealthHandler(t)
	defer sqlDB.Close()

	req, _ := http.NewRequest("GET", "/health/live", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "alive", body["status"])
	assert.NotNil(t, body["time"])
}

func TestHandler_Ready(t *testing.T) {
	app, sqlDB := setupHealthHandler(t)
	defer sqlDB.Close()

	t.Run("ready when database is connected", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health/ready", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "ready", body["status"])
		assert.Equal(t, "ok", body["database"])
	})

	t.Run("not ready when database is closed", func(t *testing.T) {
		sqlDB.Close()

		req, _ := http.NewRequest("GET", "/health/ready", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 503, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "not_ready", body["status"])
		assert.Equal(t, "error", body["database"])
	})
}

func TestHandler_Metrics(t *testing.T) {
	app, sqlDB := setupHealthHandler(t)
	defer sqlDB.Close()

	req, _ := http.NewRequest("GET", "/health/metrics", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.NotNil(t, body["goroutines"])
	assert.NotNil(t, body["memory"])
	assert.NotNil(t, body["time"])
}
