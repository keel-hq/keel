package tests

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var dockerHub0150Webhook = `{
	"push_data": {
		"pushed_at": 1497467660,
		"images": [],
		"tag": "0.0.15",
		"pusher": "karolisr"
	},
	"repository": {
		"status": "Active",
		"description": "",
		"is_trusted": false,		
		"repo_url": "https://hub.docker.com/r/webhook-demo",
		"owner": "karolisr",
		"is_official": false,
		"is_private": false,
		"name": "keel",
		"namespace": "karolisr",
		"star_count": 0,
		"comment_count": 0,
		"date_created": 1497032538,	
		"repo_name": "karolisr/webhook-demo"
	}
}`

func TestSemverUpdate(t *testing.T) {

	// stop := make(chan struct{})
	context, cancel := context.WithCancel(context.Background())
	// defer close(ctx)
	defer cancel()

	go startKeel(context)

	_, kcs := getKubernetesClient()

	t.Run("UpdateThroughDockerHubWebhook", func(t *testing.T) {

		testNamespace := createNamespaceForTest()
		defer deleteTestNamespace(testNamespace)

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:        "deployment-1",
				Namespace:   testNamespace,
				Labels:      map[string]string{types.KeelPolicyLabel: "all"},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Selector: &meta_v1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "wd-1",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						Labels: map[string]string{
							"app":     "wd-1",
							"release": "1",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							v1.Container{
								Name:  "wd-1",
								Image: "karolisr/webhook-demo:0.0.14",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}

		_, err := kcs.AppsV1().Deployments(testNamespace).Create(dep)
		if err != nil {
			t.Fatalf("failed to create deployment: %s", err)
		}
		// giving some time to get started
		// TODO: replace with a readiness check function to wait for 1/1 READY
		time.Sleep(2 * time.Second)

		// sending webhook
		client := http.DefaultClient
		buf := bytes.NewBufferString(dockerHub0150Webhook)
		req, err := http.NewRequest("POST", "http://localhost:9300/v1/webhooks/dockerhub", buf)
		if err != nil {
			t.Fatalf("failed to create req: %s", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("failed to make a webhook request to keel: %s", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("unexpected webhook response from keel: %d", resp.StatusCode)
		}

		time.Sleep(2 * time.Second)

		updated, err := kcs.AppsV1().Deployments(testNamespace).Get(dep.ObjectMeta.Name, meta_v1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get deployment: %s", err)
		}

		if updated.Spec.Template.Spec.Containers[0].Image != "karolisr/webhook-demo:0.0.15" {
			t.Errorf("expected 'karolisr/webhook-demo:0.0.15', got: '%s'", updated.Spec.Template.Spec.Containers[0].Image)
		}
	})

	t.Run("UpdateThroughDockerHubPolling", func(t *testing.T) {

		testNamespace := createNamespaceForTest()
		defer deleteTestNamespace(testNamespace)

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "deployment-1",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:  "major",
					types.KeelTriggerLabel: "poll",
				},
				Annotations: map[string]string{},
			},
			apps_v1.DeploymentSpec{
				Selector: &meta_v1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "wd-1",
					},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: meta_v1.ObjectMeta{
						Labels: map[string]string{
							"app":     "wd-1",
							"release": "1",
						},
					},
					Spec: v1.PodSpec{
						Containers: []v1.Container{
							v1.Container{
								Name:  "wd-1",
								Image: "keelhq/push-workflow-example:0.1.0-dev",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}

		_, err := kcs.AppsV1().Deployments(testNamespace).Create(dep)
		if err != nil {
			t.Fatalf("failed to create deployment: %s", err)
		}
		// giving some time to get started
		// TODO: replace with a readiness check function to wait for 1/1 READY
		time.Sleep(2 * time.Second)

		updated, err := kcs.AppsV1().Deployments(testNamespace).Get(dep.ObjectMeta.Name, meta_v1.GetOptions{})
		if err != nil {
			t.Fatalf("failed to get deployment: %s", err)
		}

		if updated.Spec.Template.Spec.Containers[0].Image != "keelhq/push-workflow-example:0.5.0-dev" {
			t.Errorf("expected 'keelhq/push-workflow-example:0.5.0-dev', got: '%s'", updated.Spec.Template.Spec.Containers[0].Image)
		}
	})

}
