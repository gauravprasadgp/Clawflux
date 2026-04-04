package kubernetes

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"
)

type ManifestSet struct {
	Namespace  string
	Deployment string
	Service    string
	Ingress    string
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
