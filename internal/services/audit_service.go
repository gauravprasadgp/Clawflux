package services

import (
	"context"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type AuditService struct {
	repo domain.AuditRepository
}

func NewAuditService(repo domain.AuditRepository) *AuditService {
	return &AuditService{repo: repo}
}

func (s *AuditService) Record(ctx context.Context, actor domain.Actor, action, resourceType, resourceID, message string) error {
	if s == nil || s.repo == nil {
		return nil
	}
	return s.repo.Create(ctx, &domain.AuditLog{
		ID:            idgen.NewUUID(),
		TenantID:      actor.TenantID,
		ActorUserID:   actor.UserID,
		ActorAPIKeyID: actor.APIKeyID,
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Message:       message,
		CreatedAt:     time.Now().UTC(),
	})
}

func (s *AuditService) ListTenantAuditLogs(ctx context.Context, actor domain.Actor, limit int) ([]domain.AuditLog, error) {
	if s == nil || s.repo == nil {
		return []domain.AuditLog{}, nil
	}
	return s.repo.ListByTenant(ctx, actor.TenantID, limit)
}
