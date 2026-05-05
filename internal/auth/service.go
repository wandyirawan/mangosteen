package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"mangosteen/pkg/cache"
	"mangosteen/pkg/crypto"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailExists        = errors.New("email already exists")
	ErrTokenRevoked       = errors.New("token has been revoked")
	ErrTokenExpired       = errors.New("token has expired")
)

type Service struct {
	repo     *Repository
	cache    *cache.ValkeyClient
	jwt      *JWTManager
	password *crypto.PasswordHasher
}

func NewService(repo *Repository, cache *cache.ValkeyClient, jwt *JWTManager) *Service {
	return &Service{
		repo:     repo,
		cache:    cache,
		jwt:      jwt,
		password: crypto.NewPasswordHasher(),
	}
}

func (s *Service) SignIn(ctx context.Context, credentials LoginDTO) (*TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, credentials.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if user.Active != 1 {
		return nil, errors.New("account is deactivated")
	}

	ok, err := s.password.Check(credentials.Password, user.PasswordHash)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokens(ctx, user)
}

func (s *Service) SignUp(ctx context.Context, data RegisterDTO) (*UserResponse, error) {
	existingUser, err := s.repo.FindByEmail(ctx, data.Email)
	if err == nil && existingUser != nil {
		return nil, ErrEmailExists
	}

	hash, err := s.password.Hash(data.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &User{
		ID:           uuid.New().String(),
		Email:        data.Email,
		PasswordHash: hash,
		Role:         "user",
		Active:       1,
		CreatedAt:    now.Format(time.RFC3339),
		UpdatedAt:    now.Format(time.RFC3339),
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	// Validate the JWT refresh token first
	claims, err := s.jwt.Validate(refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check token type
	if claims["type"] != "refresh" {
		return nil, errors.New("invalid token type")
	}

	// Get the token hash from the database
	token, err := s.repo.GetRefreshTokenByHash(ctx, refreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if token.Revoked != 0 {
		return nil, ErrTokenRevoked
	}

	// Check expiration
	expiresAt, err := time.Parse(time.RFC3339, token.ExpiresAt)
	if err != nil {
		return nil, ErrTokenExpired
	}
	if time.Now().After(expiresAt) {
		return nil, ErrTokenExpired
	}

	user, err := s.repo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if user.Active != 1 {
		return nil, errors.New("account is deactivated")
	}

	// Revoke old token
	if err := s.repo.RevokeRefreshToken(ctx, token.ID); err != nil {
		return nil, err
	}

	return s.generateTokens(ctx, user)
}

func (s *Service) Logout(ctx context.Context, userID, tokenID string) error {
	return s.repo.RevokeAllUserTokens(ctx, userID)
}

func (s *Service) generateTokens(ctx context.Context, user *User) (*TokenPair, error) {
	accessToken, err := s.jwt.IssueAccess(user.ID, user.Email, user.Role)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.jwt.IssueRefresh()
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(time.Hour * 24 * 7)
	tokenHash, err := s.password.Hash(refreshToken)
	if err != nil {
		return nil, err
	}

	if _, err := s.repo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    15 * 60,
		Type:         "Bearer",
	}, nil
}
