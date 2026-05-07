package user

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mangosteen/internal/db"
	"mangosteen/pkg/cache"
)

func setupUserService(t *testing.T) (*Service, *db.DB) {
	t.Helper()

	database, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema, err := os.ReadFile("../../sql/schema.sql")
	require.NoError(t, err)
	_, err = database.Exec(string(schema))
	require.NoError(t, err)

	var testDB db.DB
	testDB.DB = database
	repo := NewRepository(&testDB)
	valkey := cache.NewValkeyClient("", "", 0)
	svc := NewService(repo, valkey)

	return svc, &testDB
}

func createTestUser(t *testing.T, dbx *db.DB, id, email string, active bool) {
	t.Helper()
	activeInt := int64(0)
	if active {
		activeInt = 1
	}
	err := dbx.Query().CreateUser(context.Background(), db.CreateUserParams{
		ID:           id,
		Email:        email,
		PasswordHash: "$argon2id$v=19$m=65536,t=3,p=4$salt$hash",
		Role:         "user",
		Active:       activeInt,
		CreatedAt:    "2024-01-01T00:00:00Z",
		UpdatedAt:    "2024-01-01T00:00:00Z",
	})
	require.NoError(t, err)
}

func TestUserService_GetByID(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "getbyid@example.com", true)

	t.Run("found user with attributes", func(t *testing.T) {
		user, err := svc.GetByID(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "user-1", user.ID)
		assert.Equal(t, "getbyid@example.com", user.Email)
		assert.Equal(t, "user", user.Role)
		assert.True(t, user.Active)
		assert.NotNil(t, user.Attributes)
	})

	t.Run("user not found", func(t *testing.T) {
		_, err := svc.GetByID(context.Background(), "nonexistent")
		assert.Error(t, err)
	})
}

func TestUserService_ListActiveUsers(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "active1@example.com", true)
	createTestUser(t, testDB, "user-2", "active2@example.com", true)
	createTestUser(t, testDB, "user-3", "inactive@example.com", false)

	t.Run("only active users returned", func(t *testing.T) {
		users, err := svc.ListActiveUsers(context.Background())
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})

	t.Run("users have attributes populated", func(t *testing.T) {
		users, err := svc.ListActiveUsers(context.Background())
		require.NoError(t, err)
		for _, u := range users {
			assert.NotNil(t, u.Attributes)
		}
	})
}

func TestUserService_ListAllUsers(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "all1@example.com", true)
	createTestUser(t, testDB, "user-2", "all2@example.com", false)

	t.Run("all users returned", func(t *testing.T) {
		users, err := svc.ListAllUsers(context.Background())
		require.NoError(t, err)
		assert.Len(t, users, 2)
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "update@example.com", true)

	err := svc.UpdateUser(context.Background(), "user-1", map[string]interface{}{
		"email": "updated@example.com",
	})
	require.NoError(t, err)

	user, err := svc.GetByID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", user.Email)
}

func TestUserService_SoftDelete(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "delete@example.com", true)

	err := svc.SoftDelete(context.Background(), "user-1")
	require.NoError(t, err)

	user, err := svc.GetByID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.False(t, user.Active)
}

func TestUserService_HardDelete(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "destroy@example.com", true)

	err := svc.HardDelete(context.Background(), "user-1")
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), "user-1")
	assert.Error(t, err)
}

func TestUserService_ActivateUser(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "activate@example.com", false)

	err := svc.ActivateUser(context.Background(), "user-1")
	require.NoError(t, err)

	user, err := svc.GetByID(context.Background(), "user-1")
	require.NoError(t, err)
	assert.True(t, user.Active)
}

func TestUserService_Attributes(t *testing.T) {
	svc, testDB := setupUserService(t)
	defer testDB.Close()

	createTestUser(t, testDB, "user-1", "attrs@example.com", true)

	t.Run("set and get attributes", func(t *testing.T) {
		err := svc.SetAttributes(context.Background(), "user-1", map[string]string{
			"department": "engineering",
			"location":   "remote",
		})
		require.NoError(t, err)

		attrs, err := svc.GetAttributes(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, "engineering", attrs["department"])
		assert.Equal(t, "remote", attrs["location"])
	})

	t.Run("overwrite attributes", func(t *testing.T) {
		err := svc.SetAttributes(context.Background(), "user-1", map[string]string{
			"department": "marketing",
		})
		require.NoError(t, err)

		attrs, err := svc.GetAttributes(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Len(t, attrs, 1)
		assert.Equal(t, "marketing", attrs["department"])
	})

	t.Run("delete attribute", func(t *testing.T) {
		err := svc.DeleteAttribute(context.Background(), "user-1", "department")
		require.NoError(t, err)

		attrs, err := svc.GetAttributes(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Empty(t, attrs)
	})

	t.Run("empty attributes for new user", func(t *testing.T) {
		attrs, err := svc.GetAttributes(context.Background(), "user-1")
		require.NoError(t, err)
		assert.Empty(t, attrs)
	})
}
