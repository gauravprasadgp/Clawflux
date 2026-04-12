package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/gauravprasad/clawcontrol/internal/domain"
	"github.com/gauravprasad/clawcontrol/internal/repositories/memory"
	"github.com/gauravprasad/clawcontrol/internal/services"
)

type syncBackend struct {
	status *domain.BackendStatus
	err    error
}

func (b *syncBackend) Name() string {
	return "kubernetes"
}

func (b *syncBackend) Submit(context.Context, domain.BackendDeployRequest) (*domain.BackendStatus, error) {
	return nil, nil
}

func (b *syncBackend) Delete(context.Context, domain.BackendRef) error {
	return nil
}

func (b *syncBackend) GetStatus(context.Context, domain.BackendRef) (*domain.BackendStatus, error) {
	return b.status, b.err
}

type noopScheduler struct{}

func (noopScheduler) ScheduleDeployment(context.Context, *domain.Deployment) error { return nil }
func (noopScheduler) ScheduleDelete(context.Context, *domain.Deployment) error     { return nil }
func (noopScheduler) ScheduleSync(context.Context, *domain.Deployment) error       { return nil }

func TestDeploymentSyncHandlerUpdatesStatusAndCurrentDeployment(t *testing.T) {
	state := memory.NewState()
	appRepo := memory.NewAppRepo(state)
	deploymentRepo := memory.NewDeploymentRepo(state)
	eventRepo := memory.NewEventRepo(state)
	service := services.NewDeploymentService(appRepo, deploymentRepo, eventRepo, noopScheduler{})

	now := time.Now().UTC()
	app := &domain.App{
		ID:        "app_1",
		TenantID:  "tenant_1",
		Name:      "demo",
		Slug:      "demo",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := appRepo.Create(context.Background(), app); err != nil {
		t.Fatalf("create app: %v", err)
	}

	deployment := &domain.Deployment{
		ID:       "dep_1",
		TenantID: "tenant_1",
		AppID:    app.ID,
		Version:  1,
		Status:   domain.DeploymentStatusProvisioning,
		Backend:  "kubernetes",
		BackendRef: domain.BackendRef{
			Namespace:  "tenant-1",
			Deployment: "demo-v1",
			Service:    "demo-v1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := deploymentRepo.Create(context.Background(), deployment); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	handler := NewDeploymentSyncHandler(appRepo, deploymentRepo, &syncBackend{
		status: &domain.BackendStatus{
			Status: domain.DeploymentStatusRunning,
			Reason: "deployment ready",
			Ref:    deployment.BackendRef,
		},
	}, service, noopScheduler{})

	err := handler.Handle(context.Background(), domain.Job{
		ID:           "job_1",
		Type:         domain.JobTypeDeploymentSync,
		TenantID:     deployment.TenantID,
		AppID:        deployment.AppID,
		DeploymentID: deployment.ID,
	})
	if err != nil {
		t.Fatalf("sync handler: %v", err)
	}

	updatedDeployment, err := deploymentRepo.GetByID(context.Background(), deployment.TenantID, deployment.ID)
	if err != nil {
		t.Fatalf("get deployment: %v", err)
	}
	if updatedDeployment.Status != domain.DeploymentStatusRunning {
		t.Fatalf("expected running deployment, got %s", updatedDeployment.Status)
	}

	updatedApp, err := appRepo.GetByID(context.Background(), app.TenantID, app.ID)
	if err != nil {
		t.Fatalf("get app: %v", err)
	}
	if updatedApp.CurrentDeploymentID != deployment.ID {
		t.Fatalf("expected current deployment %s, got %s", deployment.ID, updatedApp.CurrentDeploymentID)
	}
}
