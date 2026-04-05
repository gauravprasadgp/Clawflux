package kubernetes

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gauravprasad/clawcontrol/internal/domain"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	if err := b.ensureNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("ensure namespace %s: %w", ns, err)
	}

	if err := b.ensureDenyAllEgress(ctx, ns); err != nil {
		return nil, fmt.Errorf("apply network policy in %s: %w", ns, err)
	}

	pod := buildPod(name, ns, req)
	if _, err := b.client.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		return nil, fmt.Errorf("create pod %s/%s: %w", ns, name, err)
	}

	return &domain.BackendStatus{
		Status: domain.DeploymentStatusProvisioning,
		Reason: "pod created, waiting for container start",
		Ref: domain.BackendRef{
			Namespace:  ns,
			Deployment: name,
		},
	}, nil
}

func (b *Backend) Delete(ctx context.Context, ref domain.BackendRef) error {
	err := b.client.CoreV1().Pods(ref.Namespace).Delete(ctx, ref.Deployment, metav1.DeleteOptions{})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}

func (b *Backend) GetStatus(ctx context.Context, ref domain.BackendRef) (*domain.BackendStatus, error) {
	pod, err := b.client.CoreV1().Pods(ref.Namespace).Get(ctx, ref.Deployment, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &domain.BackendStatus{
				Status: domain.DeploymentStatusFailed,
				Reason: "pod not found",
				Ref:    ref,
			}, nil
		}
		return nil, fmt.Errorf("get pod %s/%s: %w", ref.Namespace, ref.Deployment, err)
	}

	status, reason := mapPodStatus(pod)
	return &domain.BackendStatus{Status: status, Reason: reason, Ref: ref}, nil
}

// mapPodStatus inspects container-level state before falling back to the
// pod phase so that conditions like CrashLoopBackOff are caught early.
func mapPodStatus(pod *v1.Pod) (domain.DeploymentStatus, string) {
	for _, cs := range pod.Status.ContainerStatuses {
		if w := cs.State.Waiting; w != nil {
			switch w.Reason {
			case "CrashLoopBackOff", "ImagePullBackOff", "ErrImagePull",
				"InvalidImageName", "CreateContainerConfigError":
				return domain.DeploymentStatusFailed, w.Reason
			}
			return domain.DeploymentStatusProvisioning, w.Reason
		}
		if t := cs.State.Terminated; t != nil {
			if t.ExitCode != 0 {
				return domain.DeploymentStatusFailed,
					fmt.Sprintf("exited with code %d: %s", t.ExitCode, t.Reason)
			}
		}
	}

	switch pod.Status.Phase {
	case v1.PodRunning:
		return domain.DeploymentStatusRunning, "pod running"
	case v1.PodSucceeded:
		return domain.DeploymentStatusRunning, "pod completed successfully"
	case v1.PodFailed:
		msg := pod.Status.Reason
		if msg == "" {
			msg = "pod failed"
		}
		return domain.DeploymentStatusFailed, msg
	default:
		return domain.DeploymentStatusProvisioning, string(pod.Status.Phase)
	}
}

func buildPod(name, namespace string, req domain.BackendDeployRequest) *v1.Pod {
	cfg := req.App.Config

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

	for k, v := range cfg.Env {
		container.Env = append(container.Env, v1.EnvVar{Name: k, Value: v})
	}

	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       trimForKubeName(req.App.Slug),
				"app.kubernetes.io/managed-by": "clawplane",
				"clawplane/tenant":             trimForKubeName(req.App.TenantID),
				"clawplane/app":                trimForKubeName(req.App.ID),
				"clawplane/deployment":         trimForKubeName(req.Deployment.ID),
			},
		},
		Spec: v1.PodSpec{
			Containers:    []v1.Container{container},
			RestartPolicy: v1.RestartPolicyNever,
		},
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
				"app.kubernetes.io/managed-by": "clawplane",
			},
		},
	}, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func (b *Backend) ensureDenyAllEgress(ctx context.Context, ns string) error {
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deny-egress",
			Namespace: ns,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{},
			PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeEgress},
			Egress:      []networkingv1.NetworkPolicyEgressRule{},
		},
	}
	_, err := b.client.NetworkingV1().NetworkPolicies(ns).Create(ctx, policy, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
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
