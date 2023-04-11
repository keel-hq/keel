package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/keel-hq/keel/constants"

	"github.com/keel-hq/keel/types"

	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	log "github.com/sirupsen/logrus"
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

func TestWebhooksSemverUpdate(t *testing.T) {

	// stop := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	// defer close(ctx)
	defer cancel()

	// go startKeel(ctx)
	keel := &KeelCmd{}
	go func() {
		err := keel.Start(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to start Keel process")
		}
	}()

	defer func() {
		err := keel.Stop()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to stop Keel process")
		}
	}()

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
							{
								Name:  "wd-1",
								Image: "karolisr/webhook-demo:0.0.14",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}
		createOptions := meta_v1.CreateOptions{}

		_, err := kcs.AppsV1().Deployments(testNamespace).Create(context.Background(), dep, createOptions)
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
}

// Test to ensure Keel doesn't try to be tolerant and parse integers as semver versions, for example
// 45000 shouldn't become 45000.0.0 version (https://github.com/keel-hq/keel/issues/296)
func TestWebhookHighIntegerUpdate(t *testing.T) {

	// stop := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	// defer close(ctx)
	defer cancel()

	// go startKeel(ctx)
	keel := &KeelCmd{}
	go func() {
		err := keel.Start(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to start Keel process")
		}
	}()

	defer func() {
		err := keel.Stop()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to stop Keel process")
		}
	}()

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
							{
								Name:  "wd-1",
								Image: "karolisr/webhook-demo:0.0.14",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}
		createOptions := meta_v1.CreateOptions{}

		_, err := kcs.AppsV1().Deployments(testNamespace).Create(context.Background(), dep, createOptions)
		if err != nil {
			t.Fatalf("failed to create deployment: %s", err)
		}
		// giving some time to get started
		// TODO: replace with a readiness check function to wait for 1/1 READY
		time.Sleep(2 * time.Second)

		var payload = `{
			"push_data": {
				"pushed_at": 1497467660,
				"images": [],
				"tag": "45000",
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

		// sending webhook
		client := http.DefaultClient
		buf := bytes.NewBufferString(payload)
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

		time.Sleep(3 * time.Second)

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "karolisr/webhook-demo:0.0.14")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})
}

func TestApprovals(t *testing.T) {

	// approvals endpoint shouldn't work without at least basic auth
	// see http.go:134
	t.Skip()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// go startKeel(ctx)
	keel := &KeelCmd{}
	go func() {
		err := keel.Start(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to start Keel process")
		}
	}()

	defer func() {
		err := keel.Stop()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to stop Keel process")
		}
	}()

	_, kcs := getKubernetesClient()

	t.Run("CreateDeploymentWithApprovals", func(t *testing.T) {

		testNamespace := createNamespaceForTest()
		defer deleteTestNamespace(testNamespace)

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "deployment-1",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:           "all",
					types.KeelMinimumApprovalsLabel: "1",
					types.KeelApprovalDeadlineLabel: "5",
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
							{
								Name:  "wd-1",
								Image: "karolisr/webhook-demo:0.0.14",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}
		createOptions := meta_v1.CreateOptions{}

		_, err := kcs.AppsV1().Deployments(testNamespace).Create(context.Background(), dep, createOptions)
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

		// req2, err := http.NewRequest("GET", "http://localhost:9300/v1/approvals", nil)

		resp, err = client.Get("http://localhost:9300/v1/approvals")
		if err != nil {
			t.Fatalf("failed to get approvals: %s", err)
		}

		var approvals []*types.Approval
		dec := json.NewDecoder(resp.Body)
		defer resp.Body.Close()
		err = dec.Decode(&approvals)
		if err != nil {
			t.Fatalf("failed to decode approvals resp: %s", err)
		}

		if len(approvals) != 1 {
			t.Errorf("expected to find 1 approval, got: %d", len(approvals))
		} else {
			if approvals[0].VotesRequired != 1 {
				t.Errorf("expected 1 required vote, got: %d", approvals[0].VotesRequired)
			}
			log.Infof("approvals deadline: %s, time since: %v", approvals[0].Deadline, time.Since(approvals[0].Deadline))
			if time.Since(approvals[0].Deadline) > -4*time.Hour && time.Since(approvals[0].Deadline) < -5*time.Hour {
				t.Errorf("deadline is for: %s", approvals[0].Deadline)
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "karolisr/webhook-demo:0.0.14")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})
}

func TestApprovalsWithAuthentication(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	username := "foobar"
	password := "barfood"

	// go startKeel(ctx)
	keel := &KeelCmd{
		env: []string{
			fmt.Sprintf("%s=%s", constants.EnvBasicAuthUser, username),
			fmt.Sprintf("%s=%s", constants.EnvBasicAuthPassword, password),
		},
	}
	go func() {
		err := keel.Start(ctx)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to start Keel process")
		}
	}()

	defer func() {
		err := keel.Stop()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("failed to stop Keel process")
		}
	}()

	_, kcs := getKubernetesClient()

	t.Run("CreateDeploymentWithApprovals", func(t *testing.T) {

		testNamespace := createNamespaceForTest()
		defer deleteTestNamespace(testNamespace)

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "dep-1-auth-test",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:           "all",
					types.KeelMinimumApprovalsLabel: "1",
					types.KeelApprovalDeadlineLabel: "5",
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
							{
								Name:  "wd-1",
								Image: "karolisr/webhook-demo:0.0.14",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}
		createOptions := meta_v1.CreateOptions{}

		_, err := kcs.AppsV1().Deployments(testNamespace).Create(context.Background(), dep, createOptions)
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

		reqNoAuth, err := http.NewRequest("GET", "http://localhost:9300/v1/approvals", nil)
		if err != nil {
			t.Fatalf("failed to create req: %s", err)
		}
		respNoAuth, err := client.Do(reqNoAuth)
		if err != nil {
			t.Logf("failed to make req: %s", err)
		}
		defer respNoAuth.Body.Close()
		if respNoAuth.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401, got: %d", respNoAuth.StatusCode)
		}

		// doing it again with authentication
		reqAuth, err := http.NewRequest("GET", "http://localhost:9300/v1/approvals", nil)
		if err != nil {
			t.Fatalf("failed to create req: %s", err)
		}
		reqAuth.SetBasicAuth(username, password)
		resp, err = client.Do(reqAuth)
		if err != nil {
			t.Errorf("failed to make req: %s", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected %d, got: %d", http.StatusOK, resp.StatusCode)
		}

		var approvals []*types.Approval
		dec := json.NewDecoder(resp.Body)
		defer resp.Body.Close()
		err = dec.Decode(&approvals)
		if err != nil {
			t.Fatalf("failed to decode approvals resp: %s", err)
		}

		if len(approvals) == 0 {
			t.Errorf("no approvals found")
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "karolisr/webhook-demo:0.0.14")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})
}
