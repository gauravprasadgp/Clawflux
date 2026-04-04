package handlers

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/services"
)

type DeploymentCreateHandler struct {
	apps        domain.AppRepository
	deployments domain.DeploymentRepository
	backend     domain.DeploymentBackend
	service     *services.DeploymentService
}

func NewDeploymentCreateHandler(apps domain.AppRepository, deployments domain.DeploymentRepository, backend domain.DeploymentBackend, service *services.DeploymentService) *DeploymentCreateHandler {
	return &DeploymentCreateHandler{
		apps:        apps,
		deployments: deployments,
		backend:     backend,
		service:     service,
	}
}

func (h *DeploymentCreateHandler) Handle(ctx context.Context, job domain.Job) error {
	deployment, err := h.deployments.GetByID(ctx, job.TenantID, job.DeploymentID)
	if err != nil {
		return err
	}
	switch deployment.Status {
	case domain.DeploymentStatusCancelled, domain.DeploymentStatusDeleting, domain.DeploymentStatusDeleted:
		return nil
	}
	app, err := h.apps.GetByID(ctx, job.TenantID, job.AppID)
	if err != nil {
		return err
	}
	if err := h.service.MarkDeploymentStatus(ctx, deployment, domain.DeploymentStatusProvisioning, "creating kubernetes resources"); err != nil {
		return err
	}
	status, err := h.backend.Submit(ctx, domain.BackendDeployRequest{
		App:        *app,
		Deployment: *deployment,
	})
	if err != nil {
		_ = h.service.MarkDeploymentStatus(ctx, deployment, domain.DeploymentStatusFailed, err.Error())
		return err
	}
	deployment.BackendRef = status.Ref
	if err := h.service.MarkDeploymentStatus(ctx, deployment, status.Status, status.Reason); err != nil {
		return err
	}
	if status.Status == domain.DeploymentStatusRunning {
		app.CurrentDeploymentID = deployment.ID
		return h.apps.Update(ctx, app)
	}
	return nil
}
