package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Backend struct {
	client *kubernetes.Clientset
}

func NewBackend() (*Backend, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		home := homedir.HomeDir()
		if home == "" {
			return nil, fmt.Errorf("unable to determine home directory for kubeconfig")
		}
		config, err = clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig: %w", err)
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	return &Backend{client: client}, nil
}

func (b *Backend) Name() string { return "kubernetes" }

func (b *Backend) Submit(ctx context.Context, req domain.BackendDeployRequest) (*domain.BackendStatus, error) {
	ns := namespaceForTenant(req.App.TenantID)
	name := resourceName(req.App.Slug, req.Deployment.Version)
	labels := deploymentLabels(req.App, req.Deployment)

	if err := b.ensureNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("ensure namespace %s: %w", ns, err)
	}
	if err := b.ensureBaselineEgressPolicy(ctx, ns); err != nil {
		return nil, fmt.Errorf("apply network policy in %s: %w", ns, err)
	}

	if err := b.reconcileOpenClawConfig(ctx, req, ns, name, labels); err != nil {
		return nil, err
	}

	deployment := buildDeployment(name, ns, req, labels)
	if err := b.upsertDeployment(ctx, deployment); err != nil {
		return nil, fmt.Errorf("apply deployment %s/%s: %w", ns, name, err)
	}

	service := buildService(name, ns, req, labels)
	if err := b.upsertService(ctx, service); err != nil {
		return nil, fmt.Errorf("apply service %s/%s: %w", ns, name, err)
	}

	ingress := ""
	if req.App.Config.Public {
		ingress = ingressName(req.App)
		if err := b.upsertIngress(ctx, buildIngress(ingress, ns, req, name)); err != nil {
			return nil, fmt.Errorf("apply ingress %s/%s: %w", ns, ingress, err)
		}
	} else {
		_ = b.client.NetworkingV1().Ingresses(ns).Delete(ctx, ingressName(req.App), metav1.DeleteOptions{})
	}

	return &domain.BackendStatus{
		Status: domain.DeploymentStatusProvisioning,
		Reason: "kubernetes resources applied, waiting for rollout",
		Ref: domain.BackendRef{
			Namespace:   ns,
			Deployment:  name,
			Service:     name,
			IngressName: ingress,
		},
	}, nil
}

func (b *Backend) Delete(ctx context.Context, ref domain.BackendRef) error {
	if ref.Namespace == "" || ref.Deployment == "" {
		return nil
	}

	_ = b.client.NetworkingV1().Ingresses(ref.Namespace).Delete(ctx, ref.IngressName, metav1.DeleteOptions{})
	_ = b.client.CoreV1().Services(ref.Namespace).Delete(ctx, ref.Service, metav1.DeleteOptions{})
	if err := b.client.AppsV1().Deployments(ref.Namespace).Delete(ctx, ref.Deployment, metav1.DeleteOptions{}); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	_ = b.client.CoreV1().ConfigMaps(ref.Namespace).Delete(ctx, openClawConfigMapName(ref.Deployment), metav1.DeleteOptions{})
	_ = b.client.CoreV1().Secrets(ref.Namespace).Delete(ctx, openClawManagedSecretName(ref.Deployment), metav1.DeleteOptions{})
	return nil
}

func (b *Backend) GetStatus(ctx context.Context, ref domain.BackendRef) (*domain.BackendStatus, error) {
	deployment, err := b.client.AppsV1().Deployments(ref.Namespace).Get(ctx, ref.Deployment, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &domain.BackendStatus{
				Status: domain.DeploymentStatusFailed,
				Reason: "deployment not found",
				Ref:    ref,
			}, nil
		}
		return nil, fmt.Errorf("get deployment %s/%s: %w", ref.Namespace, ref.Deployment, err)
	}

	status, reason := mapDeploymentStatus(deployment)
	if ref.Service == "" {
		ref.Service = ref.Deployment
	}
	return &domain.BackendStatus{Status: status, Reason: reason, Ref: ref}, nil
}

func mapDeploymentStatus(deployment *appsv1.Deployment) (domain.DeploymentStatus, string) {
	if deployment.Generation > deployment.Status.ObservedGeneration {
		return domain.DeploymentStatusProvisioning, "waiting for controller to observe new generation"
	}

	for _, cond := range deployment.Status.Conditions {
		if cond.Type == appsv1.DeploymentReplicaFailure && cond.Status == v1.ConditionTrue {
			if cond.Message != "" {
				return domain.DeploymentStatusFailed, cond.Message
			}
			return domain.DeploymentStatusFailed, cond.Reason
		}
		if cond.Type == appsv1.DeploymentProgressing && cond.Reason == "ProgressDeadlineExceeded" {
			if cond.Message != "" {
				return domain.DeploymentStatusFailed, cond.Message
			}
			return domain.DeploymentStatusFailed, "deployment rollout exceeded progress deadline"
		}
	}

	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	if desired <= 0 {
		desired = 1
	}

	if deployment.Status.ReadyReplicas >= desired && deployment.Status.UpdatedReplicas >= desired {
		return domain.DeploymentStatusRunning, "deployment ready"
	}

	return domain.DeploymentStatusProvisioning, fmt.Sprintf(
		"ready %d/%d replicas",
		deployment.Status.ReadyReplicas,
		desired,
	)
}

func buildDeployment(name, namespace string, req domain.BackendDeployRequest, labels map[string]string) *appsv1.Deployment {
	cfg := req.App.Config
	selectorLabels := map[string]string{
		"clawflux/deployment": trimForKubeName(req.Deployment.ID),
	}

	container := v1.Container{
		Name:  "openclaw",
		Image: req.Deployment.ImageRef,
		Resources: v1.ResourceRequirements{
			Requests: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(cfg.CPURequest),
				v1.ResourceMemory: resource.MustParse(cfg.MemoryRequest),
			},
			Limits: v1.ResourceList{
				v1.ResourceCPU:    resource.MustParse(cfg.CPULimit),
				v1.ResourceMemory: resource.MustParse(cfg.MemoryLimit),
			},
		},
	}

	if cfg.Port > 0 {
		container.Ports = []v1.ContainerPort{{ContainerPort: int32(cfg.Port)}}
	}

	container.Env = appendSortedEnvMap(container.Env, cfg.Env)

	podSpec := v1.PodSpec{
		Containers: []v1.Container{container},
	}

	if cfg.ServiceAccountName != "" {
		podSpec.ServiceAccountName = cfg.ServiceAccountName
	}

	if cfg.OpenClaw != nil && cfg.OpenClaw.Enabled {
		configureOpenClawRuntime(&podSpec, req, name)
	}

	replicas := cfg.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: selectorLabels},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: mergeLabels(labels, selectorLabels)},
				Spec:       podSpec,
			},
		},
	}
}

func buildService(name, namespace string, req domain.BackendDeployRequest, labels map[string]string) *v1.Service {
	port := int32(appServicePort(req.App.Config))
	if port <= 0 {
		port = 3000
	}
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{
				"clawflux/deployment": trimForKubeName(req.Deployment.ID),
			},
			Ports: []v1.ServicePort{{
				Name:       "http",
				Port:       port,
				TargetPort: intstr.FromInt(int(port)),
			}},
			Type: v1.ServiceTypeClusterIP,
		},
	}
}

func buildIngress(name, namespace string, req domain.BackendDeployRequest, serviceName string) *networkingv1.Ingress {
	host := strings.TrimSpace(req.App.Config.Domain)
	if host == "" {
		host = fmt.Sprintf("%s.example.local", trimForKubeName(req.App.Slug))
	}
	servicePort := int32(appServicePort(req.App.Config))

	pathType := networkingv1.PathTypePrefix
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "clawflux",
				"clawflux/app":                 trimForKubeName(req.App.ID),
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{{
				Host: host,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{Paths: []networkingv1.HTTPIngressPath{{
						Path:     "/",
						PathType: &pathType,
						Backend: networkingv1.IngressBackend{
							Service: &networkingv1.IngressServiceBackend{
								Name: serviceName,
								Port: networkingv1.ServiceBackendPort{Number: servicePort},
							},
						},
					}}},
				},
			}},
		},
	}
}

func (b *Backend) reconcileOpenClawConfig(ctx context.Context, req domain.BackendDeployRequest, namespace, deploymentName string, labels map[string]string) error {
	cfg := req.App.Config.OpenClaw
	if cfg == nil || !cfg.Enabled {
		return nil
	}

	if err := b.ensureOpenClawWorkspacePVC(ctx, namespace, req.App); err != nil {
		return fmt.Errorf("ensure workspace pvc in %s: %w", namespace, err)
	}

	if err := b.reconcileOpenClawConfigMap(ctx, namespace, deploymentName, cfg, labels); err != nil {
		return fmt.Errorf("apply configmap in %s: %w", namespace, err)
	}

	if err := b.reconcileOpenClawSecret(ctx, namespace, deploymentName, cfg, labels); err != nil {
		return fmt.Errorf("apply secret in %s: %w", namespace, err)
	}

	return nil
}

func (b *Backend) ensureOpenClawWorkspacePVC(ctx context.Context, namespace string, app domain.App) error {
	pvcName := openClawWorkspacePVCName(app)
	_, err := b.client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	storage := "10Gi"
	if app.Config.OpenClaw != nil && strings.TrimSpace(app.Config.OpenClaw.WorkspaceStorage) != "" {
		storage = strings.TrimSpace(app.Config.OpenClaw.WorkspaceStorage)
	}

	_, err = b.client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "clawflux",
				"clawflux/app":                 trimForKubeName(app.ID),
			},
		},
		Spec: v1.PersistentVolumeClaimSpec{
			AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
			Resources: v1.VolumeResourceRequirements{
				Requests: v1.ResourceList{v1.ResourceStorage: resource.MustParse(storage)},
			},
		},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (b *Backend) reconcileOpenClawConfigMap(ctx context.Context, namespace, deploymentName string, cfg *domain.OpenClawConfig, labels map[string]string) error {
	data := map[string]string{}
	if strings.TrimSpace(cfg.AgentsMarkdown) != "" {
		data["AGENTS.md"] = cfg.AgentsMarkdown
	}
	if strings.TrimSpace(cfg.SettingsJSON) != "" {
		data["settings.json"] = cfg.SettingsJSON
	}

	name := openClawConfigMapName(deploymentName)
	if len(data) == 0 {
		_ = b.client.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		return nil
	}

	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
	return b.upsertConfigMap(ctx, cm)
}

func (b *Backend) reconcileOpenClawSecret(ctx context.Context, namespace, deploymentName string, cfg *domain.OpenClawConfig, labels map[string]string) error {
	if cfg.ExistingSecretName != "" {
		return nil
	}

	stringData := map[string]string{}
	if strings.TrimSpace(cfg.GatewayToken) != "" {
		stringData["OPENCLAW_GATEWAY_TOKEN"] = strings.TrimSpace(cfg.GatewayToken)
	}
	for k, v := range cfg.ProviderAPIKeys {
		key := strings.TrimSpace(k)
		if key == "" || strings.TrimSpace(v) == "" {
			continue
		}
		stringData[key] = strings.TrimSpace(v)
	}

	name := openClawManagedSecretName(deploymentName)
	if len(stringData) == 0 {
		_ = b.client.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
		return nil
	}

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Type:       v1.SecretTypeOpaque,
		StringData: stringData,
	}
	return b.upsertSecret(ctx, secret)
}

func configureOpenClawRuntime(podSpec *v1.PodSpec, req domain.BackendDeployRequest, deploymentName string) {
	cfg := req.App.Config.OpenClaw
	if cfg == nil {
		return
	}

	container := &podSpec.Containers[0]
	if strings.TrimSpace(cfg.GatewayBindAddress) != "" {
		container.Env = upsertLiteralEnv(container.Env, "OPENCLAW_GATEWAY_BIND_ADDRESS", strings.TrimSpace(cfg.GatewayBindAddress))
	}
	if cfg.GatewayPort > 0 {
		container.Env = upsertLiteralEnv(container.Env, "OPENCLAW_GATEWAY_PORT", strconv.Itoa(cfg.GatewayPort))
	}

	managedSecretName := openClawManagedSecretName(deploymentName)
	secretName := strings.TrimSpace(cfg.ExistingSecretName)
	if secretName == "" {
		secretName = managedSecretName
	}

	if strings.TrimSpace(cfg.GatewayToken) != "" || cfg.ExistingSecretName != "" {
		container.Env = upsertSecretEnv(container.Env, "OPENCLAW_GATEWAY_TOKEN", secretName, "OPENCLAW_GATEWAY_TOKEN")
	}
	for _, key := range sortedKeys(cfg.ProviderAPIKeys) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		if secretName == managedSecretName && strings.TrimSpace(cfg.ProviderAPIKeys[key]) == "" {
			continue
		}
		container.Env = upsertSecretEnv(container.Env, key, secretName, key)
	}

	container.Env = appendSortedEnvMap(container.Env, cfg.ExtraEnv)

	container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
		Name:      "openclaw-workspace",
		MountPath: "/home/openclaw/.openclaw/workspace",
	})
	podSpec.Volumes = append(podSpec.Volumes, v1.Volume{
		Name: "openclaw-workspace",
		VolumeSource: v1.VolumeSource{PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
			ClaimName: openClawWorkspacePVCName(req.App),
		}},
	})

	items := []v1.KeyToPath{}
	if strings.TrimSpace(cfg.AgentsMarkdown) != "" {
		items = append(items, v1.KeyToPath{Key: "AGENTS.md", Path: "AGENTS.md"})
	}
	if strings.TrimSpace(cfg.SettingsJSON) != "" {
		items = append(items, v1.KeyToPath{Key: "settings.json", Path: "settings.json"})
	}
	if len(items) > 0 {
		container.VolumeMounts = append(container.VolumeMounts, v1.VolumeMount{
			Name:      "openclaw-config",
			MountPath: "/home/openclaw/.openclaw",
			ReadOnly:  true,
		})
		podSpec.Volumes = append(podSpec.Volumes, v1.Volume{
			Name: "openclaw-config",
			VolumeSource: v1.VolumeSource{ConfigMap: &v1.ConfigMapVolumeSource{
				LocalObjectReference: v1.LocalObjectReference{Name: openClawConfigMapName(deploymentName)},
				Items:                items,
			}},
		})
	}
}

func deploymentLabels(app domain.App, deployment domain.Deployment) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       trimForKubeName(app.Slug),
		"app.kubernetes.io/managed-by": "clawflux",
		"clawflux/tenant":              trimForKubeName(app.TenantID),
		"clawflux/app":                 trimForKubeName(app.ID),
		"clawflux/deployment":          trimForKubeName(deployment.ID),
	}
}

func (b *Backend) ensureNamespace(ctx context.Context, ns string) error {
	_, err := b.client.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	_, err = b.client.CoreV1().Namespaces().Create(ctx, &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "clawflux",
			},
		},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (b *Backend) ensureBaselineEgressPolicy(ctx context.Context, ns string) error {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clawflux-allow-egress",
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/managed-by": "clawflux"},
			},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{{}},
		},
	}
	return b.upsertNetworkPolicy(ctx, policy)
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

func openClawConfigMapName(deploymentName string) string {
	return trimForKubeName(deploymentName) + "-config"
}

func openClawManagedSecretName(deploymentName string) string {
	return trimForKubeName(deploymentName) + "-secrets"
}

func openClawWorkspacePVCName(app domain.App) string {
	return trimForKubeName(app.Slug) + "-workspace"
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

func appServicePort(cfg domain.AppConfig) int {
	if cfg.Port > 0 {
		return cfg.Port
	}
	if cfg.OpenClaw != nil && cfg.OpenClaw.GatewayPort > 0 {
		return cfg.OpenClaw.GatewayPort
	}
	return 3000
}

func (b *Backend) upsertDeployment(ctx context.Context, deployment *appsv1.Deployment) error {
	existing, err := b.client.AppsV1().Deployments(deployment.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = b.client.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	deployment.ResourceVersion = existing.ResourceVersion
	_, err = b.client.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	return err
}

func (b *Backend) upsertService(ctx context.Context, service *v1.Service) error {
	existing, err := b.client.CoreV1().Services(service.Namespace).Get(ctx, service.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = b.client.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	service.ResourceVersion = existing.ResourceVersion
	service.Spec.ClusterIP = existing.Spec.ClusterIP
	service.Spec.ClusterIPs = existing.Spec.ClusterIPs
	service.Spec.IPFamilies = existing.Spec.IPFamilies
	service.Spec.IPFamilyPolicy = existing.Spec.IPFamilyPolicy
	_, err = b.client.CoreV1().Services(service.Namespace).Update(ctx, service, metav1.UpdateOptions{})
	return err
}

func (b *Backend) upsertIngress(ctx context.Context, ingress *networkingv1.Ingress) error {
	existing, err := b.client.NetworkingV1().Ingresses(ingress.Namespace).Get(ctx, ingress.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = b.client.NetworkingV1().Ingresses(ingress.Namespace).Create(ctx, ingress, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	ingress.ResourceVersion = existing.ResourceVersion
	_, err = b.client.NetworkingV1().Ingresses(ingress.Namespace).Update(ctx, ingress, metav1.UpdateOptions{})
	return err
}

func (b *Backend) upsertConfigMap(ctx context.Context, cm *v1.ConfigMap) error {
	existing, err := b.client.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = b.client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	cm.ResourceVersion = existing.ResourceVersion
	_, err = b.client.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}

func (b *Backend) upsertSecret(ctx context.Context, secret *v1.Secret) error {
	existing, err := b.client.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = b.client.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	secret.ResourceVersion = existing.ResourceVersion
	_, err = b.client.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	return err
}

func (b *Backend) upsertNetworkPolicy(ctx context.Context, policy *networkingv1.NetworkPolicy) error {
	existing, err := b.client.NetworkingV1().NetworkPolicies(policy.Namespace).Get(ctx, policy.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = b.client.NetworkingV1().NetworkPolicies(policy.Namespace).Create(ctx, policy, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}
	policy.ResourceVersion = existing.ResourceVersion
	_, err = b.client.NetworkingV1().NetworkPolicies(policy.Namespace).Update(ctx, policy, metav1.UpdateOptions{})
	return err
}

func mergeLabels(first map[string]string, second map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range first {
		out[k] = v
	}
	for k, v := range second {
		out[k] = v
	}
	return out
}

func appendSortedEnvMap(in []v1.EnvVar, values map[string]string) []v1.EnvVar {
	for _, key := range sortedKeys(values) {
		if strings.TrimSpace(key) == "" {
			continue
		}
		in = upsertLiteralEnv(in, key, values[key])
	}
	return in
}

func upsertLiteralEnv(in []v1.EnvVar, name string, value string) []v1.EnvVar {
	for i := range in {
		if in[i].Name == name {
			in[i].Value = value
			in[i].ValueFrom = nil
			return in
		}
	}
	return append(in, v1.EnvVar{Name: name, Value: value})
}

func upsertSecretEnv(in []v1.EnvVar, envName, secretName, secretKey string) []v1.EnvVar {
	ref := &v1.EnvVarSource{SecretKeyRef: &v1.SecretKeySelector{
		LocalObjectReference: v1.LocalObjectReference{Name: secretName},
		Key:                  secretKey,
	}}
	for i := range in {
		if in[i].Name == envName {
			in[i].Value = ""
			in[i].ValueFrom = ref
			return in
		}
	}
	return append(in, v1.EnvVar{Name: envName, ValueFrom: ref})
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
