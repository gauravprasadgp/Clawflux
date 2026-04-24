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
	backend     domain.DeploymentBackend
	backendName string
}

func NewDeploymentService(apps domain.AppRepository, deployments domain.DeploymentRepository, events domain.EventRepository, scheduler domain.Scheduler) *DeploymentService {
	return &DeploymentService{
		apps:        apps,
		deployments: deployments,
		events:      events,
		scheduler:   scheduler,
		backendName: "kubernetes",
	}
}

func (s *DeploymentService) SetBackend(backend domain.DeploymentBackend) {
	s.backend = backend
	if backend != nil {
		s.backendName = backend.Name()
	}
}

func (s *DeploymentService) BackendCapabilities() domain.BackendCapabilities {
	if s.backend == nil {
		return domain.BackendCapabilities{Name: s.backendName, DisplayName: s.backendName}
	}
	return s.backend.Capabilities()
}

func (s *DeploymentService) PlanDeployment(ctx context.Context, actor domain.Actor, appID string) (*domain.DeploymentPlan, error) {
	if s.backend == nil {
		return nil, domain.ErrValidation
	}
	app, err := s.apps.GetByID(ctx, actor.TenantID, appID)
	if err != nil {
		return nil, err
	}
	version, err := s.deployments.NextVersion(ctx, actor.TenantID, appID)
	if err != nil {
		return nil, err
	}
	return s.backend.Plan(ctx, domain.BackendPlanRequest{
		App:         *app,
		NextVersion: version,
	})
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
		ID:             idgen.NewUUID(),
		TenantID:       actor.TenantID,
		AppID:          app.ID,
		Version:        version,
		ImageRef:       app.Config.Image,
		ConfigSnapshot: app.Config,
		Status:         domain.DeploymentStatusQueued,
		Backend:        s.backendName,
		RequestedBy:    actor.UserID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.deployments.Create(ctx, deployment); err != nil {
		return nil, err
	}
	if err := s.events.Create(ctx, &domain.DeploymentEvent{
		ID:           idgen.NewUUID(),
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

func (s *DeploymentService) RetryDeployment(ctx context.Context, actor domain.Actor, deploymentID string) (*domain.Deployment, error) {
	deployment, err := s.deployments.GetByID(ctx, actor.TenantID, deploymentID)
	if err != nil {
		return nil, err
	}
	if deployment.Status != domain.DeploymentStatusFailed && deployment.Status != domain.DeploymentStatusCancelled {
		return nil, domain.ErrValidation
	}
	deployment.Status = domain.DeploymentStatusQueued
	deployment.StatusReason = ""
	deployment.FinishedAt = nil
	deployment.UpdatedAt = time.Now().UTC()
	if err := s.deployments.Update(ctx, deployment); err != nil {
		return nil, err
	}
	if err := s.events.Create(ctx, &domain.DeploymentEvent{
		ID:           idgen.NewUUID(),
		DeploymentID: deployment.ID,
		TenantID:     deployment.TenantID,
		Type:         "retry_queued",
		Message:      "deployment retry queued",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		return nil, err
	}
	if err := s.scheduler.ScheduleDeployment(ctx, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func (s *DeploymentService) CancelDeployment(ctx context.Context, actor domain.Actor, deploymentID string) (*domain.Deployment, error) {
	deployment, err := s.deployments.GetByID(ctx, actor.TenantID, deploymentID)
	if err != nil {
		return nil, err
	}
	switch deployment.Status {
	case domain.DeploymentStatusQueued, domain.DeploymentStatusProvisioning:
	default:
		return nil, domain.ErrValidation
	}
	if err := s.MarkDeploymentStatus(ctx, deployment, domain.DeploymentStatusCancelled, "deployment cancelled"); err != nil {
		return nil, err
	}
	return deployment, nil
}

func (s *DeploymentService) DeleteDeployment(ctx context.Context, actor domain.Actor, deploymentID string) (*domain.Deployment, error) {
	deployment, err := s.deployments.GetByID(ctx, actor.TenantID, deploymentID)
	if err != nil {
		return nil, err
	}
	if deployment.Status == domain.DeploymentStatusDeleted || deployment.Status == domain.DeploymentStatusDeleting {
		return deployment, nil
	}
	if err := s.MarkDeploymentStatus(ctx, deployment, domain.DeploymentStatusDeleting, "deployment deletion queued"); err != nil {
		return nil, err
	}
	if err := s.scheduler.ScheduleDelete(ctx, deployment); err != nil {
		return nil, err
	}
	return deployment, nil
}

func (s *DeploymentService) MarkDeploymentStatus(ctx context.Context, deployment *domain.Deployment, status domain.DeploymentStatus, reason string) error {
	now := time.Now().UTC()
	deployment.Status = status
	deployment.StatusReason = reason
	deployment.UpdatedAt = now
	if status == domain.DeploymentStatusProvisioning && deployment.StartedAt == nil {
		deployment.StartedAt = &now
	}
	if status == domain.DeploymentStatusRunning || status == domain.DeploymentStatusFailed || status == domain.DeploymentStatusCancelled || status == domain.DeploymentStatusDeleted {
		deployment.FinishedAt = &now
	}
	if err := s.deployments.Update(ctx, deployment); err != nil {
		return err
	}
	return s.events.Create(ctx, &domain.DeploymentEvent{
		ID:           idgen.NewUUID(),
		DeploymentID: deployment.ID,
		TenantID:     deployment.TenantID,
		Type:         string(status),
		Message:      reason,
		CreatedAt:    now,
	})
}
