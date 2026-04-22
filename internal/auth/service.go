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
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailExists       = errors.New("email already exists")
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

	ok, _ := s.password.Check(credentials.Password, user.PasswordHash)
	if !ok {
		return nil, ErrInvalidCredentials
	}

	return s.generateTokens(ctx, user)
}

func (s *Service) SignUp(ctx context.Context, data RegisterDTO) (*UserResponse, error) {
	_, err := s.repo.FindByEmail(ctx, data.Email)
	if err == nil {
		return nil, ErrEmailExists
	}

	hash, err := s.password.Hash(data.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().Format(time.RFC3339)
	user := &User{
		ID:           uuid.New().String(),
		Email:        data.Email,
		PasswordHash: hash,
		Role:         "user",
		Active:       true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return &UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Role:      user.Role,
		CreatedAt: time.Now(),
	}, nil
}

func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error) {
	token, err := s.repo.GetRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByID(ctx, token.UserID)
	if err != nil {
		return nil, err
	}

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
	tokenHash, _ := s.password.Hash(refreshToken)
	if _, err := s.repo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt.Format(time.RFC3339)); err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:  15 * 60,
		Type:      "Bearer",
	}, nil
}