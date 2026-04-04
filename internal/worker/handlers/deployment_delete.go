package handlers

import (
	"context"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type DeploymentDeleteHandler struct {
	apps        domain.AppRepository
	deployments domain.DeploymentRepository
	backend     domain.DeploymentBackend
	service     interface {
		MarkDeploymentStatus(ctx context.Context, deployment *domain.Deployment, status domain.DeploymentStatus, reason string) error
	}
}

func NewDeploymentDeleteHandler(apps domain.AppRepository, deployments domain.DeploymentRepository, backend domain.DeploymentBackend, service interface {
	MarkDeploymentStatus(ctx context.Context, deployment *domain.Deployment, status domain.DeploymentStatus, reason string) error
}) *DeploymentDeleteHandler {
	return &DeploymentDeleteHandler{
		apps:        apps,
		deployments: deployments,
		backend:     backend,
		service:     service,
	}
}

func (h *DeploymentDeleteHandler) Handle(ctx context.Context, job domain.Job) error {
	deployment, err := h.deployments.GetByID(ctx, job.TenantID, job.DeploymentID)
	if err != nil {
		return err
	}
	if err := h.backend.Delete(ctx, deployment.BackendRef); err != nil {
		_ = h.service.MarkDeploymentStatus(ctx, deployment, domain.DeploymentStatusFailed, err.Error())
		return err
	}
	if err := h.service.MarkDeploymentStatus(ctx, deployment, domain.DeploymentStatusDeleted, "deployment deleted"); err != nil {
		return err
	}
	app, err := h.apps.GetByID(ctx, job.TenantID, job.AppID)
	if err == nil && app.CurrentDeploymentID == deployment.ID {
		app.CurrentDeploymentID = ""
		return h.apps.Update(ctx, app)
	}
	return nil
}
