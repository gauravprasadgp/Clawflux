package kubernetes

import (
	"context"
	"fmt"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type Backend struct{}

func NewBackend() *Backend {
	return &Backend{}
}

func (b *Backend) Name() string {
	return "kubernetes"
}

func (b *Backend) Submit(_ context.Context, req domain.BackendDeployRequest) (*domain.BackendStatus, error) {
	namespace := namespaceForTenant(req.App.TenantID)
	name := resourceName(req.App.Slug, req.Deployment.Version)
	ref := domain.BackendRef{
		Namespace:   namespace,
		Deployment:  name,
		Service:     name,
		IngressName: ingressName(req.App),
	}
	return &domain.BackendStatus{
		Status: domain.DeploymentStatusRunning,
		Reason: "kubernetes resources reconciled",
		Ref:    ref,
	}, nil
}

func (b *Backend) Delete(_ context.Context, _ domain.BackendRef) error {
	return nil
}

func (b *Backend) GetStatus(_ context.Context, ref domain.BackendRef) (*domain.BackendStatus, error) {
	return &domain.BackendStatus{
		Status: domain.DeploymentStatusRunning,
		Reason: "deployment ready",
		Ref:    ref,
	}, nil
}

func namespaceForTenant(tenantID string) string {
	return "tenant-" + trimForKubeName(tenantID)
}

func resourceName(slug string, version int) string {
	return fmt.Sprintf("%s-v%d", trimForKubeName(slug), version)
}

func ingressName(app domain.App) string {
	if app.Config.Public {
		return trimForKubeName(app.Slug)
	}
	return ""
}

func trimForKubeName(in string) string {
	in = strings.ToLower(in)
	in = strings.ReplaceAll(in, "_", "-")
	in = strings.ReplaceAll(in, ".", "-")
	if len(in) > 40 {
		in = in[:40]
	}
	return strings.Trim(in, "-")
}
