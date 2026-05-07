package auth

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mangosteen/config"
	"mangosteen/internal/db"
	"mangosteen/pkg/cache"
)

func setupTestService(t *testing.T) (*Service, *sql.DB) {
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

	return svc, database
}

func newTestRepo(rawDB *sql.DB) *Repository {
	var testDB db.DB
	testDB.DB = rawDB
	return NewRepository(&testDB)
}

func TestSignUp(t *testing.T) {
	svc, sqlDB := setupTestService(t)
	defer sqlDB.Close()

	t.Run("successful registration", func(t *testing.T) {
		user, err := svc.SignUp(context.Background(), RegisterDTO{
			Email:    "test@example.com",
			Password: "password123",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, user.ID)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "user", user.Role)
		assert.NotEmpty(t, user.CreatedAt)
	})

	t.Run("duplicate email rejected", func(t *testing.T) {
		_, err := svc.SignUp(context.Background(), RegisterDTO{
			Email:    "test@example.com",
			Password: "anotherpass",
		})
		assert.ErrorIs(t, err, ErrEmailExists)
	})

	t.Run("password is hashed", func(t *testing.T) {
		user, err := svc.SignUp(context.Background(), RegisterDTO{
			Email:    "hash@example.com",
			Password: "securepass",
		})
		require.NoError(t, err)

		repo := newTestRepo(sqlDB)
		dbUser, err := repo.FindByID(context.Background(), user.ID)
		require.NoError(t, err)
		assert.Contains(t, dbUser.PasswordHash, "$argon2id$")
	})

	t.Run("multiple registrations", func(t *testing.T) {
		u1, err := svc.SignUp(context.Background(), RegisterDTO{
			Email:    "multi1@example.com",
			Password: "password",
		})
		require.NoError(t, err)

		u2, err := svc.SignUp(context.Background(), RegisterDTO{
			Email:    "multi2@example.com",
			Password: "password",
		})
		require.NoError(t, err)

		assert.NotEqual(t, u1.ID, u2.ID)
		assert.NotEqual(t, u1.Email, u2.Email)
	})
}

func TestSignIn(t *testing.T) {
	svc, sqlDB := setupTestService(t)
	defer sqlDB.Close()

	_, err := svc.SignUp(context.Background(), RegisterDTO{
		Email:    "login@example.com",
		Password: "correctpass",
	})
	require.NoError(t, err)

	t.Run("valid credentials", func(t *testing.T) {
		tokens, err := svc.SignIn(context.Background(), LoginDTO{
			Email:    "login@example.com",
			Password: "correctpass",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, 15*60, tokens.ExpiresIn)
		assert.Equal(t, "Bearer", tokens.Type)
	})

	t.Run("wrong password", func(t *testing.T) {
		_, err := svc.SignIn(context.Background(), LoginDTO{
			Email:    "login@example.com",
			Password: "wrongpass",
		})
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("non-existent user", func(t *testing.T) {
		_, err := svc.SignIn(context.Background(), LoginDTO{
			Email:    "noexist@example.com",
			Password: "anything",
		})
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("deactivated account", func(t *testing.T) {
		repo := newTestRepo(sqlDB)
		user, err := repo.FindByEmail(context.Background(), "login@example.com")
		require.NoError(t, err)

		err = repo.db.SoftDeleteUser(context.Background(), db.SoftDeleteUserParams{
			ID:        user.ID,
			UpdatedAt: user.UpdatedAt,
		})
		require.NoError(t, err)

		_, err = svc.SignIn(context.Background(), LoginDTO{
			Email:    "login@example.com",
			Password: "correctpass",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "deactivated")
	})
}

func TestRefreshToken(t *testing.T) {
	svc, sqlDB := setupTestService(t)
	defer sqlDB.Close()

	_, err := svc.SignUp(context.Background(), RegisterDTO{
		Email:    "refresh@example.com",
		Password: "password",
	})
	require.NoError(t, err)

	_, err = svc.SignIn(context.Background(), LoginDTO{
		Email:    "refresh@example.com",
		Password: "password",
	})
	require.NoError(t, err)

	t.Run("garbage refresh token rejected", func(t *testing.T) {
		_, err := svc.RefreshToken(context.Background(), "garbage.token.value")
		assert.Error(t, err)
	})

	t.Run("empty refresh token rejected", func(t *testing.T) {
		_, err := svc.RefreshToken(context.Background(), "")
		assert.Error(t, err)
	})
}

func TestLogout(t *testing.T) {
	svc, sqlDB := setupTestService(t)
	defer sqlDB.Close()

	_, err := svc.SignUp(context.Background(), RegisterDTO{
		Email:    "logout@example.com",
		Password: "password",
	})
	require.NoError(t, err)

	t.Run("logout revokes all user tokens", func(t *testing.T) {
		repo := newTestRepo(sqlDB)
		dbUser, err := repo.FindByEmail(context.Background(), "logout@example.com")
		require.NoError(t, err)

		tokens, err := svc.SignIn(context.Background(), LoginDTO{
			Email:    "logout@example.com",
			Password: "password",
		})
		require.NoError(t, err)

		err = svc.Logout(context.Background(), dbUser.ID, "")
		require.NoError(t, err)

		_, err = svc.RefreshToken(context.Background(), tokens.RefreshToken)
		assert.Error(t, err)
	})
}
