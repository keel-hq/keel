package tests

import (
	"context"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const e2eRunLabel = "keel.sh/e2e-run"

type e2eConfig struct {
	runID            string
	registry         string
	keelImage        string
	kubeconfig       string
	kubectl          string
	k3sDataDir       string
	artifactDir      string
	repositoryPrefix string
}

func loadE2EConfig() (e2eConfig, error) {
	cfg := e2eConfig{
		runID:            os.Getenv("KEEL_E2E_RUN_ID"),
		registry:         os.Getenv("KEEL_E2E_REGISTRY"),
		keelImage:        os.Getenv("KEEL_E2E_IMAGE"),
		kubeconfig:       os.Getenv("KEEL_E2E_KUBECONFIG"),
		kubectl:          os.Getenv("KEEL_E2E_KUBECTL"),
		k3sDataDir:       os.Getenv("KEEL_E2E_K3S_DATA_DIR"),
		artifactDir:      os.Getenv("KEEL_E2E_ARTIFACT_DIR"),
		repositoryPrefix: os.Getenv("KEEL_E2E_REPOSITORY_PREFIX"),
	}
	if cfg.runID == "" || cfg.registry == "" || cfg.keelImage == "" || cfg.kubeconfig == "" || cfg.kubectl == "" || cfg.k3sDataDir == "" || cfg.artifactDir == "" || cfg.repositoryPrefix == "" {
		return e2eConfig{}, fmt.Errorf("all KEEL_E2E_* runtime variables are required; run through make e2e")
	}
	return cfg, nil
}

func (cfg e2eConfig) systemNamespace() string {
	return "keel-e2e-system-" + cfg.runID
}

func (cfg e2eConfig) labels() map[string]string {
	return map[string]string{"app": "keel-e2e", e2eRunLabel: cfg.runID}
}

func createKeelResources(ctx context.Context, client kubernetes.Interface, cfg e2eConfig) error {
	labels := cfg.labels()
	namespace := cfg.systemNamespace()
	objects := []func() error{
		func() error {
			_, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: labels},
			}, metav1.CreateOptions{})
			return err
		},
		func() error {
			_, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "keel", Namespace: namespace, Labels: labels},
			}, metav1.CreateOptions{})
			return err
		},
		func() error {
			_, err := client.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{
				ObjectMeta: metav1.ObjectMeta{Name: "keel-e2e-" + cfg.runID, Labels: labels},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{"namespaces"}, Verbs: []string{"list", "watch"}},
					{APIGroups: []string{""}, Resources: []string{"nodes"}, Verbs: []string{"get", "list", "watch"}},
					{APIGroups: []string{""}, Resources: []string{"secrets"}, Verbs: []string{"get", "list", "watch"}},
					{APIGroups: []string{"", "apps", "batch"}, Resources: []string{"pods", "replicasets", "replicationcontrollers", "statefulsets", "deployments", "daemonsets", "jobs", "cronjobs"}, Verbs: []string{"get", "delete", "list", "update", "watch"}},
				},
			}, metav1.CreateOptions{})
			return err
		},
		func() error {
			_, err := client.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "keel-e2e-" + cfg.runID, Labels: labels},
				RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: "keel-e2e-" + cfg.runID},
				Subjects:   []rbacv1.Subject{{Kind: "ServiceAccount", Name: "keel", Namespace: namespace}},
			}, metav1.CreateOptions{})
			return err
		},
		func() error {
			_, err := client.AppsV1().Deployments(namespace).Create(ctx, keelDeployment(cfg), metav1.CreateOptions{})
			return err
		},
		func() error {
			_, err := client.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "keel", Namespace: namespace, Labels: labels},
				Spec: corev1.ServiceSpec{
					Selector: labels,
					Ports:    []corev1.ServicePort{{Name: "http", Port: 9300}},
				},
			}, metav1.CreateOptions{})
			return err
		},
	}
	for _, create := range objects {
		if err := create(); err != nil {
			return err
		}
	}
	return nil
}

func keelDeployment(cfg e2eConfig) *appsv1.Deployment {
	labels := cfg.labels()
	replicas := int32(1)
	nonRoot := true
	noPrivilegeEscalation := false
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "keel", Namespace: cfg.systemNamespace(), Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					ServiceAccountName: "keel",
					SecurityContext:    &corev1.PodSecurityContext{RunAsNonRoot: &nonRoot},
					Containers: []corev1.Container{{
						Name:            "keel",
						Image:           cfg.keelImage,
						ImagePullPolicy: corev1.PullIfNotPresent,
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &noPrivilegeEscalation},
						Env: []corev1.EnvVar{
							{Name: "DEBUG", Value: "true"},
							{Name: "INSECURE_REGISTRY", Value: "true"},
							{Name: "POLL", Value: "true"},
							{Name: "POLL_DEFAULTSCHEDULE", Value: "@every 2s"},
						},
						Ports:          []corev1.ContainerPort{{Name: "http", ContainerPort: 9300}},
						ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromString("http")}}, InitialDelaySeconds: 1, PeriodSeconds: 1, FailureThreshold: 30},
						LivenessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromString("http")}}, InitialDelaySeconds: 5, PeriodSeconds: 5},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("64Mi")},
							Limits:   corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("256Mi")},
						},
					}},
				},
			},
		},
	}
}
