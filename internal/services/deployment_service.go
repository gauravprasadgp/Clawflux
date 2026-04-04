package services

import (
	"context"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/platform/idgen"
)

type DeploymentService struct {
	apps        domain.AppRepository
	deployments domain.DeploymentRepository
	events      domain.EventRepository
	scheduler   domain.Scheduler
}

func NewDeploymentService(apps domain.AppRepository, deployments domain.DeploymentRepository, events domain.EventRepository, scheduler domain.Scheduler) *DeploymentService {
	return &DeploymentService{
		apps:        apps,
		deployments: deployments,
		events:      events,
		scheduler:   scheduler,
	}
}

func (s *DeploymentService) CreateDeployment(ctx context.Context, actor domain.Actor, appID string) (*domain.Deployment, error) {
	app, err := s.apps.GetByID(ctx, actor.TenantID, appID)
	if err != nil {
		return nil, err
	}
	version, err := s.deployments.NextVersion(ctx, actor.TenantID, appID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	deployment := &domain.Deployment{
		ID:             idgen.New("dep"),
		TenantID:       actor.TenantID,
		AppID:          app.ID,
		Version:        version,
		ImageRef:       app.Config.Image,
		ConfigSnapshot: app.Config,
		Status:         domain.DeploymentStatusQueued,
		Backend:        "kubernetes",
		RequestedBy:    actor.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.deployments.Create(ctx, deployment); err != nil {
		return nil, err
	}
	if err := s.events.Create(ctx, &domain.DeploymentEvent{
		ID:           idgen.New("evt"),
		DeploymentID: deployment.ID,
		TenantID:     deployment.TenantID,
		Type:         "queued",
		Message:      "deployment queued",
		CreatedAt:    now,
	}); err != nil {
		return nil, err
	}
	if err := s.scheduler.ScheduleDeployment(ctx, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func (s *DeploymentService) GetDeployment(ctx context.Context, actor domain.Actor, deploymentID string) (*domain.Deployment, error) {
	return s.deployments.GetByID(ctx, actor.TenantID, deploymentID)
}

func (s *DeploymentService) ListDeployments(ctx context.Context, actor domain.Actor, appID string) ([]domain.Deployment, error) {
	return s.deployments.ListByApp(ctx, actor.TenantID, appID)
}

func (s *DeploymentService) ListEvents(ctx context.Context, actor domain.Actor, deploymentID string) ([]domain.DeploymentEvent, error) {
	return s.events.ListByDeployment(ctx, actor.TenantID, deploymentID)
}

func (s *DeploymentService) MarkDeploymentStatus(ctx context.Context, deployment *domain.Deployment, status domain.DeploymentStatus, reason string) error {
	now := time.Now().UTC()
	deployment.Status = status
	deployment.StatusReason = reason
	deployment.UpdatedAt = now
	if status == domain.DeploymentStatusProvisioning && deployment.StartedAt == nil {
		deployment.StartedAt = &now
	}
	if status == domain.DeploymentStatusRunning || status == domain.DeploymentStatusFailed || status == domain.DeploymentStatusDeleted {
		deployment.FinishedAt = &now
	}
	if err := s.deployments.Update(ctx, deployment); err != nil {
		return err
	}
	return s.events.Create(ctx, &domain.DeploymentEvent{
		ID:           idgen.New("evt"),
		DeploymentID: deployment.ID,
		TenantID:     deployment.TenantID,
		Type:         string(status),
		Message:      reason,
		CreatedAt:    now,
	})
}
