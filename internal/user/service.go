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

func (s *Service) GetByID(ctx context.Context, id string) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *Service) ListActiveUsers(ctx context.Context) ([]User, error) {
	return s.repo.FindActive(ctx)
}

func (s *Service) ListAllUsers(ctx context.Context) ([]User, error) {
	return s.repo.FindAll(ctx)
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