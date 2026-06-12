package admin

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
	"mangosteen/pkg/queue"
)

// MockUploadWorker is a test stub for UploadWorkerInterface
type MockUploadWorker struct {
	GetStatsFunc         func() (map[queue.Status]int64, error)
	RetryFailedPermanentFunc func() error
}

func (m *MockUploadWorker) GetStats() (map[queue.Status]int64, error) {
	if m.GetStatsFunc != nil {
		return m.GetStatsFunc()
	}
	return map[queue.Status]int64{}, nil
}

func (m *MockUploadWorker) RetryFailedPermanent() error {
	if m.RetryFailedPermanentFunc != nil {
		return m.RetryFailedPermanentFunc()
	}
	return nil
}

func setupAdminTest(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema, err := os.ReadFile("../../sql/schema.sql")
	require.NoError(t, err)
	_, err = database.Exec(string(schema))
	require.NoError(t, err)

	var testDB db.DB
	testDB.DB = database

	mockWorker := &MockUploadWorker{}
	handler := NewHandler(mockWorker, &testDB)

	app := fiber.New()

	// Create auth middleware mocks
	requireAuth := func(c *fiber.Ctx) error {
		// Check if user is authenticated (simplified for testing)
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(401).JSON(fiber.Map{"error": "unauthorized"})
		}
		// Set role from header for testing
		role := c.Get("X-User-Role")
		if role != "" {
			c.Locals("role", role)
		}
		return c.Next()
	}

	requireAdmin := func(c *fiber.Ctx) error {
		role := c.Locals("role")
		if role == nil || role != "admin" {
			return c.Status(403).JSON(fiber.Map{"error": "forbidden"})
		}
		return c.Next()
	}

	handler.RegisterRoutes(app, requireAuth, requireAdmin)

	return app, database
}

func TestHandler_Info(t *testing.T) {
	app, sqlDB := setupAdminTest(t)
	defer sqlDB.Close()

	t.Run("returns service info", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/info", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-User-Role", "admin")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "mangosteen", body["service"])
		assert.Equal(t, "1.0.0", body["version"])
	})
}

func TestHandler_GetStats(t *testing.T) {
	app, sqlDB := setupAdminTest(t)
	defer sqlDB.Close()

	t.Run("returns stats successfully", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/logs/stats", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-User-Role", "admin")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.NotNil(t, body["stats"])
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/logs/stats", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("non-admin returns 403", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/admin/logs/stats", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-User-Role", "user")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})
}

func TestHandler_RetryUploads(t *testing.T) {
	app, sqlDB := setupAdminTest(t)
	defer sqlDB.Close()

	t.Run("retry succeeds", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/logs/retry", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-User-Role", "admin")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		var body map[string]interface{}
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
		assert.Equal(t, "Retry triggered for failed uploads", body["message"])
	})

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/logs/retry", nil)
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 401, resp.StatusCode)
	})

	t.Run("non-admin returns 403", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/admin/logs/retry", nil)
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("X-User-Role", "user")
		resp, err := app.Test(req)
		require.NoError(t, err)
		assert.Equal(t, 403, resp.StatusCode)
	})
}
