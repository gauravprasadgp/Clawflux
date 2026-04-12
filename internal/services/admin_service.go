package services

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type AdminService struct {
	repo             domain.AdminRepository
	repositoryDriver string
}

func NewAdminService(repo domain.AdminRepository, repositoryDriver string) *AdminService {
	return &AdminService{repo: repo, repositoryDriver: repositoryDriver}
}

func (s *AdminService) Summary(ctx context.Context, actor domain.Actor) (*domain.AdminSummary, error) {
	if !actor.IsPlatformAdmin {
		return nil, domain.ErrForbidden
	}
	summary, err := s.repo.Summary(ctx)
	if err != nil {
		return nil, err
	}
	summary.RepositoryDriver = s.repositoryDriver
	return summary, nil
}

func (s *AdminService) ListAllInstances(ctx context.Context, actor domain.Actor) ([]domain.AdminInstance, error) {
	if !actor.IsPlatformAdmin {
		return nil, domain.ErrForbidden
	}
	return s.repo.ListAllInstances(ctx)
}
