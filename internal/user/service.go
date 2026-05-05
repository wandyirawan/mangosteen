package user

import (
	"context"

	"mangosteen/pkg/cache"
)

type Service struct {
	repo  *Repository
	cache *cache.ValkeyClient
}

func NewService(repo *Repository, cache *cache.ValkeyClient) *Service {
	return &Service{repo: repo, cache: cache}
}

func (s *Service) UpdateUser(ctx context.Context, id string, updates map[string]interface{}) error {
	return s.repo.Update(ctx, id, updates)
}

func (s *Service) SoftDelete(ctx context.Context, id string) error {
	return s.repo.SoftDelete(ctx, id)
}

func (s *Service) HardDelete(ctx context.Context, id string) error {
	return s.repo.HardDelete(ctx, id)
}

func (s *Service) ActivateUser(ctx context.Context, id string) error {
	return s.repo.Activate(ctx, id)
}

func (s *Service) GetAttributes(ctx context.Context, userID string) (map[string]string, error) {
	return s.repo.FindAttributes(ctx, userID)
}

func (s *Service) SetAttributes(ctx context.Context, userID string, attrs map[string]string) error {
	return s.repo.SetAttributes(ctx, userID, attrs)
}

func (s *Service) DeleteAttribute(ctx context.Context, userID, key string) error {
	return s.repo.DeleteAttribute(ctx, userID, key)
}

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, err
    }
    attrs, _ := s.repo.FindAttributes(ctx, id)
    user.Attributes = attrs
    return user, nil
}

func (s *Service) ListActiveUsers(ctx context.Context) ([]User, error) {
	users, err := s.repo.FindActive(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		attrs, _ := s.repo.FindAttributes(ctx, users[i].ID)
		users[i].Attributes = attrs
	}
	return users, nil
}

func (s *Service) ListAllUsers(ctx context.Context) ([]User, error) {
	users, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	for i := range users {
		attrs, _ := s.repo.FindAttributes(ctx, users[i].ID)
		users[i].Attributes = attrs
	}
	return users, nil
}