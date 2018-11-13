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
	ctx, cancel := context.WithCancel(context.Background())
	// defer close(ctx)
	defer cancel()

	go startKeel(ctx)

	_, kcs := getKubernetesClient()

	t.Run("UpdateThroughDockerHubWebhook", func(t *testing.T) {

		// t.Skip()

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

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "karolisr/webhook-demo:0.0.15")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})

	t.Run("UpdateThroughDockerHubPollingA", func(t *testing.T) {
		// UpdateThroughDockerHubPollingA tests a polling trigger when we have a higher version
		// but without a pre-release tag and a lower version with pre-release. The version of the deployment
		// is with pre-prerealse so we should upgrade to that one.

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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "keelhq/push-workflow-example:0.5.0-dev")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})

	t.Run("UpdateThroughDockerHubPollingB", func(t *testing.T) {
		// UpdateThroughDockerHubPollingA tests a polling trigger when we have a higher version
		// but without a pre-release tag and a lower version with pre-release. The version of the deployment
		// is without pre-prerealse

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
								Image: "keelhq/push-workflow-example:0.1.0",
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "keelhq/push-workflow-example:0.10.0")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})
}
