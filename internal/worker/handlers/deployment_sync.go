package handlers

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type DeploymentSyncHandler struct {
	apps        domain.AppRepository
	deployments domain.DeploymentRepository
	backend     domain.DeploymentBackend
	scheduler   domain.Scheduler
	service     interface {
		MarkDeploymentStatus(ctx context.Context, deployment *domain.Deployment, status domain.DeploymentStatus, reason string) error
		QueueAutoRepair(ctx context.Context, deployment *domain.Deployment, reason string) error
	}
}

func NewDeploymentSyncHandler(apps domain.AppRepository, deployments domain.DeploymentRepository, backend domain.DeploymentBackend, service interface {
	MarkDeploymentStatus(ctx context.Context, deployment *domain.Deployment, status domain.DeploymentStatus, reason string) error
	QueueAutoRepair(ctx context.Context, deployment *domain.Deployment, reason string) error
}, scheduler domain.Scheduler) *DeploymentSyncHandler {
	return &DeploymentSyncHandler{
		apps:        apps,
		deployments: deployments,
		backend:     backend,
		scheduler:   scheduler,
		service:     service,
	}
}

func (h *DeploymentSyncHandler) Handle(ctx context.Context, job domain.Job) error {
	deployment, err := h.deployments.GetByID(ctx, job.TenantID, job.DeploymentID)
	if err != nil {
		return err
	}

	switch deployment.Status {
	case domain.DeploymentStatusCancelled, domain.DeploymentStatusDeleted:
		return nil
	}

	status, err := h.backend.GetStatus(ctx, deployment.BackendRef)
	if err != nil {
		return err
	}
	deployment.BackendRef = status.Ref
	if status.Status == domain.DeploymentStatusFailed {
		if err := h.service.QueueAutoRepair(ctx, deployment, status.Reason); err == nil {
			return nil
		}
	}
	if err := h.service.MarkDeploymentStatus(ctx, deployment, status.Status, status.Reason); err != nil {
		return err
	}
	if status.Status == domain.DeploymentStatusProvisioning || status.Status == domain.DeploymentStatusDegraded || status.Status == domain.DeploymentStatusRecovering {
		return h.scheduler.ScheduleSync(ctx, deployment)
	}
	if status.Status != domain.DeploymentStatusRunning {
		return nil
	}

	app, err := h.apps.GetByID(ctx, job.TenantID, job.AppID)
	if err != nil {
		return err
	}
	if app.CurrentDeploymentID == deployment.ID {
		return nil
	}
	app.CurrentDeploymentID = deployment.ID
	return h.apps.Update(ctx, app)
}
