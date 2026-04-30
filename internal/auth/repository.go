package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
	"mangosteen/internal/db"
)

type Repository struct {
	db *db.Queries
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database.Query()}
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row, err := r.db.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return toUser(row), nil
}

func (r *Repository) Create(ctx context.Context, user *User) error {
	return r.db.CreateUser(ctx, db.CreateUserParams{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         user.Role,
		Active:       true,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	})
}

func (r *Repository) FindByID(ctx context.Context, id string) (*User, error) {
	row, err := r.db.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toUser(row), nil
}

func (r *Repository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (string, error) {
	id := uuid.New().String()
	err := r.db.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: time.Now(),
	})
	return id, err
}

func (r *Repository) GetRefreshToken(ctx context.Context, id string) (*RefreshToken, error) {
	row, err := r.db.GetRefreshToken(ctx, id)
	if err != nil {
		return nil, err
	}
	return toRefreshToken(row), nil
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	row, err := r.db.GetRefreshTokenByHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	return toRefreshToken(row), nil
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, id string) error {
	return r.db.RevokeRefreshToken(ctx, id)
}

func (r *Repository) RevokeAllUserTokens(ctx context.Context, userID string) error {
	return r.db.RevokeAllUserTokens(ctx, userID)
}

func toUser(row db.User) *User {
	return &User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		Active:       row.Active,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}

func toRefreshToken(row db.RefreshToken) *RefreshToken {
	return &RefreshToken{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		Revoked:   row.Revoked,
		CreatedAt: row.CreatedAt,
	}
}
