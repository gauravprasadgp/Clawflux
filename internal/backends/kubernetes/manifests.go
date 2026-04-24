package kubernetes

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type ManifestSet struct {
	Namespace  string
	Deployment string
	Service    string
	Ingress    string
}

func BuildPlan(app domain.App, version int, capabilities domain.BackendCapabilities) *domain.DeploymentPlan {
	name := resourceName(app.Slug, version)
	namespace := namespaceForTenant(app.TenantID)
	port := appServicePort(app.Config)
	ingress := ingressName(app)
	host := strings.TrimSpace(app.Config.Domain)
	if app.Config.Public && host == "" {
		host = fmt.Sprintf("%s.example.local", trimForKubeName(app.Slug))
	}

	plan := &domain.DeploymentPlan{
		Backend:      capabilities.Name,
		Version:      version,
		ImageRef:     app.Config.Image,
		Capabilities: capabilities,
		BackendRef: domain.BackendRef{
			Namespace:   namespace,
			Deployment:  name,
			Service:     name,
			IngressName: ingress,
		},
		Exposure: domain.PlannedExposure{
			Public:      app.Config.Public,
			Host:        host,
			ServiceName: name,
			Port:        port,
			IngressName: ingress,
		},
		Resources: []domain.PlannedResource{
			{Kind: "Namespace", Name: namespace, Action: "apply"},
			{Kind: "NetworkPolicy", Name: "clawflux-allow-egress", Namespace: namespace, Action: "apply", Note: "allow egress for managed workloads"},
			{Kind: "Deployment", Name: name, Namespace: namespace, Action: "apply"},
			{Kind: "Service", Name: name, Namespace: namespace, Action: "apply"},
		},
	}

	if app.Config.Public {
		plan.Resources = append(plan.Resources, domain.PlannedResource{
			Kind:      "Ingress",
			Name:      ingress,
			Namespace: namespace,
			Action:    "apply",
			Note:      "routes public HTTP traffic to the service",
		})
	} else {
		plan.Resources = append(plan.Resources, domain.PlannedResource{
			Kind:      "Ingress",
			Name:      trimForKubeName(app.Slug),
			Namespace: namespace,
			Action:    "delete-if-present",
			Note:      "private deployments remove the public ingress",
		})
	}

	plan.Environment = appendSortedPlanEnv(plan.Environment, app.Config.Env, "literal")
	if cfg := app.Config.OpenClaw; cfg != nil && cfg.Enabled {
		appendOpenClawPlan(plan, app, name, namespace, cfg)
	}

	if strings.Contains(strings.ToLower(app.Config.Image), ":latest") {
		plan.Warnings = append(plan.Warnings, "Image uses the latest tag; pin a version for repeatable rollouts.")
	}
	if app.Config.Public && strings.TrimSpace(app.Config.Domain) == "" {
		plan.Warnings = append(plan.Warnings, "Public deployment has no explicit domain; Kubernetes ingress will use a generated local host.")
	}
	if app.Config.Replicas > 1 && app.Config.OpenClaw != nil && app.Config.OpenClaw.Enabled {
		plan.Warnings = append(plan.Warnings, "OpenClaw workspace PVC uses ReadWriteOnce; more than one replica can require storage-class support or a shared workspace strategy.")
	}

	return plan
}

func appendOpenClawPlan(plan *domain.DeploymentPlan, app domain.App, deploymentName, namespace string, cfg *domain.OpenClawConfig) {
	storage := strings.TrimSpace(cfg.WorkspaceStorage)
	if storage == "" {
		storage = "10Gi"
	}
	pvcName := openClawWorkspacePVCName(app)
	plan.Resources = append(plan.Resources, domain.PlannedResource{
		Kind:      "PersistentVolumeClaim",
		Name:      pvcName,
		Namespace: namespace,
		Action:    "apply-if-missing",
		Note:      "workspace storage is preserved across rollouts",
	})
	plan.Volumes = append(plan.Volumes, domain.PlannedVolume{
		Name:      "openclaw-workspace",
		Source:    pvcName,
		MountPath: "/home/openclaw/.openclaw/workspace",
		Size:      storage,
	})

	if strings.TrimSpace(cfg.AgentsMarkdown) != "" || strings.TrimSpace(cfg.SettingsJSON) != "" {
		name := openClawConfigMapName(deploymentName)
		plan.Resources = append(plan.Resources, domain.PlannedResource{Kind: "ConfigMap", Name: name, Namespace: namespace, Action: "apply"})
		plan.Volumes = append(plan.Volumes, domain.PlannedVolume{
			Name:      "openclaw-config",
			Source:    name,
			MountPath: "/home/openclaw/.openclaw",
		})
	}

	if strings.TrimSpace(cfg.GatewayBindAddress) != "" {
		plan.Environment = append(plan.Environment, domain.PlannedEnvVar{Name: "OPENCLAW_GATEWAY_BIND_ADDRESS", Source: "literal"})
	}
	if cfg.GatewayPort > 0 {
		plan.Environment = append(plan.Environment, domain.PlannedEnvVar{Name: "OPENCLAW_GATEWAY_PORT", Source: "literal:" + strconv.Itoa(cfg.GatewayPort)})
	}
	plan.Environment = appendSortedPlanEnv(plan.Environment, cfg.ExtraEnv, "literal")

	secretName := strings.TrimSpace(cfg.ExistingSecretName)
	managedSecret := false
	if secretName == "" {
		secretName = openClawManagedSecretName(deploymentName)
		managedSecret = true
	}

	secretKeys := []string{}
	if strings.TrimSpace(cfg.GatewayToken) != "" || strings.TrimSpace(cfg.ExistingSecretName) != "" {
		secretKeys = append(secretKeys, "OPENCLAW_GATEWAY_TOKEN")
		plan.Environment = append(plan.Environment, domain.PlannedEnvVar{Name: "OPENCLAW_GATEWAY_TOKEN", Source: "secret:" + secretName})
	}
	for _, key := range sortedKeys(cfg.ProviderAPIKeys) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if managedSecret && strings.TrimSpace(cfg.ProviderAPIKeys[key]) == "" {
			continue
		}
		secretKeys = append(secretKeys, key)
		plan.Environment = append(plan.Environment, domain.PlannedEnvVar{Name: key, Source: "secret:" + secretName})
	}
	if len(secretKeys) > 0 {
		plan.Secrets = append(plan.Secrets, domain.PlannedSecret{
			Name:    secretName,
			Keys:    secretKeys,
			Managed: managedSecret,
		})
		if managedSecret {
			plan.Resources = append(plan.Resources, domain.PlannedResource{Kind: "Secret", Name: secretName, Namespace: namespace, Action: "apply"})
		}
	} else if managedSecret {
		plan.Resources = append(plan.Resources, domain.PlannedResource{
			Kind:      "Secret",
			Name:      secretName,
			Namespace: namespace,
			Action:    "delete-if-present",
			Note:      "no inline secret values are present",
		})
		plan.Warnings = append(plan.Warnings, "No provider API keys or existing secret are configured for OpenClaw.")
	}
	if !managedSecret {
		plan.Warnings = append(plan.Warnings, "Existing secret references are not verified during dry-run.")
	}
}

func appendSortedPlanEnv(env []domain.PlannedEnvVar, values map[string]string, source string) []domain.PlannedEnvVar {
	for _, key := range sortedKeys(values) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		env = append(env, domain.PlannedEnvVar{Name: key, Source: source})
	}
	return env
}

func RenderManifestSet(app domain.App, deployment domain.Deployment) ManifestSet {
	namespace := namespaceForTenant(app.TenantID)
	name := resourceName(app.Slug, deployment.Version)

	var deploymentYAML strings.Builder
	deploymentYAML.WriteString("apiVersion: apps/v1\n")
	deploymentYAML.WriteString("kind: Deployment\n")
	deploymentYAML.WriteString("metadata:\n")
	deploymentYAML.WriteString(fmt.Sprintf("  name: %s\n  namespace: %s\n", name, namespace))
	deploymentYAML.WriteString("spec:\n")
	deploymentYAML.WriteString(fmt.Sprintf("  replicas: %d\n", app.Config.Replicas))
	deploymentYAML.WriteString("  selector:\n    matchLabels:\n")
	deploymentYAML.WriteString(fmt.Sprintf("      app.kubernetes.io/name: %s\n", trimForKubeName(app.Slug)))
	deploymentYAML.WriteString("  template:\n    metadata:\n      labels:\n")
	deploymentYAML.WriteString(fmt.Sprintf("        app.kubernetes.io/name: %s\n", trimForKubeName(app.Slug)))
	deploymentYAML.WriteString("    spec:\n")
	deploymentYAML.WriteString("      containers:\n")
	deploymentYAML.WriteString(fmt.Sprintf("      - name: openclaw\n        image: %s\n", deployment.ImageRef))
	deploymentYAML.WriteString(fmt.Sprintf("        ports:\n        - containerPort: %d\n", app.Config.Port))
	deploymentYAML.WriteString("        resources:\n")
	deploymentYAML.WriteString("          requests:\n")
	deploymentYAML.WriteString(fmt.Sprintf("            cpu: %s\n            memory: %s\n", app.Config.CPURequest, app.Config.MemoryRequest))
	deploymentYAML.WriteString("          limits:\n")
	deploymentYAML.WriteString(fmt.Sprintf("            cpu: %s\n            memory: %s\n", app.Config.CPULimit, app.Config.MemoryLimit))
	if envBlock := renderEnv(app.Config.Env); envBlock != "" {
		deploymentYAML.WriteString(envBlock)
	}

	var serviceYAML strings.Builder
	serviceYAML.WriteString("apiVersion: v1\nkind: Service\nmetadata:\n")
	serviceYAML.WriteString(fmt.Sprintf("  name: %s\n  namespace: %s\n", name, namespace))
	serviceYAML.WriteString("spec:\n  selector:\n")
	serviceYAML.WriteString(fmt.Sprintf("    app.kubernetes.io/name: %s\n", trimForKubeName(app.Slug)))
	serviceYAML.WriteString("  ports:\n")
	serviceYAML.WriteString(fmt.Sprintf("  - port: 80\n    targetPort: %d\n", app.Config.Port))

	var ingressYAML strings.Builder
	if app.Config.Public {
		host := app.Config.Domain
		if host == "" {
			host = fmt.Sprintf("%s.example.local", trimForKubeName(app.Slug))
		}
		ingressYAML.WriteString("apiVersion: networking.k8s.io/v1\nkind: Ingress\nmetadata:\n")
		ingressYAML.WriteString(fmt.Sprintf("  name: %s\n  namespace: %s\n", trimForKubeName(app.Slug), namespace))
		ingressYAML.WriteString("spec:\n  rules:\n")
		ingressYAML.WriteString(fmt.Sprintf("  - host: %s\n    http:\n      paths:\n", host))
		ingressYAML.WriteString("      - path: /\n        pathType: Prefix\n        backend:\n          service:\n")
		ingressYAML.WriteString(fmt.Sprintf("            name: %s\n            port:\n              number: 80\n", name))
	}

	return ManifestSet{
		Namespace:  namespaceYAML(namespace),
		Deployment: deploymentYAML.String(),
		Service:    serviceYAML.String(),
		Ingress:    ingressYAML.String(),
	}
}

func namespaceYAML(namespace string) string {
	return fmt.Sprintf("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n", namespace)
}

func renderEnv(env map[string]string) string {
	if len(env) == 0 {
		return ""
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("        env:\n")
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("        - name: %s\n          value: %q\n", key, env[key]))
	}
	return b.String()
}
