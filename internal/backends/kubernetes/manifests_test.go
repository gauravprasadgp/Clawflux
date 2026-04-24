package kubernetes

import (
	"testing"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

func TestBuildPlanDescribesOpenClawResourcesWithoutSecretValues(t *testing.T) {
	app := domain.App{
		ID:       "app_123",
		TenantID: "tenant_123",
		Name:     "OpenClaw",
		Slug:     "openclaw",
		Config: domain.AppConfig{
			Image:    "ghcr.io/openclaw/openclaw:latest",
			Port:     18789,
			Replicas: 2,
			Public:   true,
			OpenClaw: &domain.OpenClawConfig{
				Enabled:            true,
				GatewayBindAddress: "0.0.0.0",
				GatewayPort:        18789,
				GatewayToken:       "super-secret",
				WorkspaceStorage:   "25Gi",
				ProviderAPIKeys: map[string]string{
					"OPENAI_API_KEY": "sk-test",
				},
				ExtraEnv: map[string]string{
					"OPENCLAW_MODE": "team",
				},
				AgentsMarkdown: "You are helpful.",
				SettingsJSON:   `{"model":"gpt-5.4-mini"}`,
			},
		},
	}

	plan := BuildPlan(app, 3, Capabilities())

	if plan.Backend != "kubernetes" {
		t.Fatalf("expected kubernetes backend, got %q", plan.Backend)
	}
	if plan.BackendRef.Namespace != "tenant-tenant-123" {
		t.Fatalf("unexpected namespace %q", plan.BackendRef.Namespace)
	}
	if plan.BackendRef.Deployment != "openclaw-v3" {
		t.Fatalf("unexpected deployment %q", plan.BackendRef.Deployment)
	}
	if !hasResource(plan.Resources, "PersistentVolumeClaim", "openclaw-workspace") {
		t.Fatalf("expected workspace pvc resource in plan")
	}
	if !hasResource(plan.Resources, "ConfigMap", "openclaw-v3-config") {
		t.Fatalf("expected configmap resource in plan")
	}
	if !hasResource(plan.Resources, "Secret", "openclaw-v3-secrets") {
		t.Fatalf("expected managed secret resource in plan")
	}
	if !hasEnv(plan.Environment, "OPENAI_API_KEY", "secret:openclaw-v3-secrets") {
		t.Fatalf("expected provider api key to be referenced from secret")
	}
	if hasEnv(plan.Environment, "sk-test", "") || hasEnv(plan.Environment, "super-secret", "") {
		t.Fatalf("plan leaked secret values: %#v", plan.Environment)
	}
	if len(plan.Warnings) == 0 {
		t.Fatalf("expected warnings for latest tag and multi-replica OpenClaw workspace")
	}
}

func hasResource(resources []domain.PlannedResource, kind, name string) bool {
	for _, resource := range resources {
		if resource.Kind == kind && resource.Name == name {
			return true
		}
	}
	return false
}

func hasEnv(env []domain.PlannedEnvVar, name, source string) bool {
	for _, item := range env {
		if item.Name != name {
			continue
		}
		if source == "" || item.Source == source {
			return true
		}
	}
	return false
}
