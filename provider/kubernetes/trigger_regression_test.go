package kubernetes

import (
	"testing"

	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	regressionRepository = "registry.example.com/team/app"
	regressionInitialTag = "main_0000000000000000000000000000000000000000"
	regressionWebhookTag = "main_1111111111111111111111111111111111111111"
	regressionPollTag    = "main_ffffffffffffffffffffffffffffffffffffffff"
	regressionPolicy     = `regexp:^main_[a-fA-F0-9]{40}$`
)

func regressionStatefulSet(trigger string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-statefulset",
			Namespace: "default",
			Annotations: map[string]string{
				types.KeelPolicyLabel:             regressionPolicy,
				types.KeelTriggerLabel:            trigger,
				types.KeelInitContainerAnnotation: "true",
			},
		},
		Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			InitContainers: []corev1.Container{{Name: "tracked-init", Image: regressionRepository + ":" + regressionInitialTag}},
			Containers:     []corev1.Container{{Name: "app", Image: "busybox:1.36"}},
		}}},
	}
}

func regressionDeployment(trigger string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "poll-deployment",
			Namespace: "default",
			Annotations: map[string]string{
				types.KeelPolicyLabel:  regressionPolicy,
				types.KeelTriggerLabel: trigger,
			},
		},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app", Image: regressionRepository + ":" + regressionInitialTag}},
		}}},
	}
}

func TestWebhookStatefulSetInitContainerIsNotRolledBackByPollEvent(t *testing.T) {
	statefulSet, err := k8s.NewGenericResource(regressionStatefulSet("webhook"))
	if err != nil {
		t.Fatal(err)
	}
	pollDeployment, err := k8s.NewGenericResource(regressionDeployment("poll"))
	if err != nil {
		t.Fatal(err)
	}
	cache := &k8s.GenericResourceCache{}
	cache.Add(statefulSet, pollDeployment)
	approvalsManager, teardown := approver()
	defer teardown()
	implementer := &fakeImplementer{}
	p, err := NewProvider(implementer, &fakeSender{}, approvalsManager, cache)
	if err != nil {
		t.Fatal(err)
	}

	webhookUpdated, err := p.processEvent(&types.Event{
		Repository:  types.Repository{Name: regressionRepository, Tag: regressionWebhookTag},
		TriggerName: "native",
	})
	if err != nil {
		t.Fatalf("webhook processEvent() error = %v", err)
	}
	if got := len(webhookUpdated); got != 2 {
		t.Fatalf("webhook updated %d resources, want 2", got)
	}
	cache.Add(webhookUpdated...) // informer resync after the webhook update

	pollUpdated, err := p.processEvent(&types.Event{
		Repository:  types.Repository{Name: regressionRepository, Tag: regressionPollTag},
		TriggerName: types.TriggerTypePoll.String(),
	})
	if err != nil {
		t.Fatalf("poll processEvent() error = %v", err)
	}
	if got := len(pollUpdated); got != 1 {
		t.Fatalf("poll updated %d resources, want only the explicit poll resource", got)
	}
	if got := pollUpdated[0].Name; got != "poll-deployment" {
		t.Fatalf("poll updated %q, want poll-deployment", got)
	}

	for _, resource := range cache.Values() {
		if resource.Name != "webhook-statefulset" {
			continue
		}
		if got, want := resource.InitContainers()[0].Image, regressionRepository+":"+regressionWebhookTag; got != want {
			t.Fatalf("webhook-selected init image changed after poll/resync: got %q, want %q", got, want)
		}
		return
	}
	t.Fatal("webhook StatefulSet missing from cache")
}

func TestPollEventTriggerIsolationAcrossWorkloadKinds(t *testing.T) {
	workloads := map[string]func(metav1.ObjectMeta, corev1.PodSpec) interface{}{
		"deployment": func(meta metav1.ObjectMeta, podSpec corev1.PodSpec) interface{} {
			return &appsv1.Deployment{ObjectMeta: meta, Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: podSpec}}}
		},
		"statefulset": func(meta metav1.ObjectMeta, podSpec corev1.PodSpec) interface{} {
			return &appsv1.StatefulSet{ObjectMeta: meta, Spec: appsv1.StatefulSetSpec{Template: corev1.PodTemplateSpec{Spec: podSpec}}}
		},
		"daemonset": func(meta metav1.ObjectMeta, podSpec corev1.PodSpec) interface{} {
			return &appsv1.DaemonSet{ObjectMeta: meta, Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: podSpec}}}
		},
		"cronjob": func(meta metav1.ObjectMeta, podSpec corev1.PodSpec) interface{} {
			return &batchv1.CronJob{ObjectMeta: meta, Spec: batchv1.CronJobSpec{JobTemplate: batchv1.JobTemplateSpec{Spec: batchv1.JobSpec{Template: corev1.PodTemplateSpec{Spec: podSpec}}}}}
		},
	}

	for kind, workload := range workloads {
		for _, tt := range []struct {
			trigger string
			want    int
		}{{trigger: "webhook", want: 0}, {trigger: "poll", want: 1}} {
			t.Run(kind+"/"+tt.trigger, func(t *testing.T) {
				meta := metav1.ObjectMeta{
					Name:      kind + "-" + tt.trigger,
					Namespace: "default",
					Annotations: map[string]string{
						types.KeelPolicyLabel:             regressionPolicy,
						types.KeelTriggerLabel:            tt.trigger,
						types.KeelInitContainerAnnotation: "true",
					},
				}
				podSpec := corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "init", Image: regressionRepository + ":" + regressionInitialTag}},
					Containers:     []corev1.Container{{Name: "app", Image: regressionRepository + ":" + regressionInitialTag}},
				}
				resource, err := k8s.NewGenericResource(workload(meta, podSpec))
				if err != nil {
					t.Fatal(err)
				}
				cache := &k8s.GenericResourceCache{}
				cache.Add(resource)
				p, err := NewProvider(&fakeImplementer{}, &fakeSender{}, nil, cache)
				if err != nil {
					t.Fatal(err)
				}

				plans, err := p.createUpdatePlansForEvent(&types.Event{
					Repository:  types.Repository{Name: regressionRepository, Tag: regressionPollTag},
					TriggerName: types.TriggerTypePoll.String(),
				})
				if err != nil {
					t.Fatalf("createUpdatePlansForEvent() error = %v", err)
				}
				if got := len(plans); got != tt.want {
					t.Fatalf("poll plans = %d, want %d", got, tt.want)
				}
				if tt.want == 0 {
					return
				}
				wantImage := regressionRepository + ":" + regressionPollTag
				if got := plans[0].Resource.Containers()[0].Image; got != wantImage {
					t.Fatalf("standard container image = %q, want %q", got, wantImage)
				}
				if got := plans[0].Resource.InitContainers()[0].Image; got != wantImage {
					t.Fatalf("init container image = %q, want %q", got, wantImage)
				}
			})
		}
	}
}
