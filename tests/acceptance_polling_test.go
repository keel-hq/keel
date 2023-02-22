package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/keel-hq/keel/secrets"
	"github.com/keel-hq/keel/types"
	log "github.com/sirupsen/logrus"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPollingSemverUpdate(t *testing.T) {

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
				Annotations: map[string]string{
					types.KeelPollScheduleAnnotation: "@every 2s",
				},
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
				Name:      "deployment-2",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:  "major",
					types.KeelTriggerLabel: "poll",
				},
				Annotations: map[string]string{
					types.KeelPollScheduleAnnotation: "@every 2s",
				},
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

	t.Run("UpdateThroughDockerHubPollingC", func(t *testing.T) {

		testNamespace := createNamespaceForTest()
		defer deleteTestNamespace(testNamespace)

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "deployment-2",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:  "major",
					types.KeelTriggerLabel: "poll",
				},
				Annotations: map[string]string{
					types.KeelPollScheduleAnnotation: "@every 2s",
				},
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
								Image: "keelhq/push-workflow-example:0.3.0-alpha",
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

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "keelhq/push-workflow-example:0.11.0-alpha")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})
}

func TestPollingPrivateRegistry(t *testing.T) {

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

	t.Run("UpdateThroughPrivateQuayPollingA", func(t *testing.T) {

		user := os.Getenv("DOCKERHUB_USERNAME")
		password := os.Getenv("DOCKERHUB_PASSWORD")

		if user == "" || password == "" {
			fmt.Println("[X] Skipping UpdateThroughPrivateQuayPollingA test since DOCKERHUB_USERNAME and/or DOCKERHUB_PASSWORD env vars not set")
			t.Skip()
		}

		// UpdateThroughDockerHubPollingA tests a polling trigger when we have a higher version
		// but without a pre-release tag and a lower version with pre-release. The version of the deployment
		// is with pre-prerealse so we should upgrade to that one.

		testNamespace := createNamespaceForTest()
		defer deleteTestNamespace(testNamespace)

		payload, err := secrets.EncodeDockerCfgJson(&secrets.DockerCfg{
			"https://index.docker.io/v1/": &secrets.Auth{
				Auth: secrets.EncodeBase64Secret(user, password),
			},
		})
		if err != nil {
			t.Fatalf("failed to encode docker cfg secret payload: %s", err)
		}

		secretName := "verysecret"

		fmt.Println(string(payload))

		secret := &v1.Secret{
			TypeMeta: meta_v1.TypeMeta{},
			ObjectMeta: meta_v1.ObjectMeta{
				Name:        secretName,
				Namespace:   testNamespace,
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": []byte(payload),
			},
		}

		_, err = kcs.CoreV1().Secrets(testNamespace).Create(secret)
		if err != nil {
			t.Fatalf("failed to create secret: %s", err)
		}

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "deployment-1",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:  "major",
					types.KeelTriggerLabel: "poll",
				},
				Annotations: map[string]string{
					types.KeelPollScheduleAnnotation: "@every 2s",
				},
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
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: secretName,
							},
						},
						Containers: []v1.Container{
							{
								ImagePullPolicy: v1.PullAlways,
								Name:            "wd-1",
								Image:           "karolisr/demo-webhook:0.0.1",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}

		_, err = kcs.AppsV1().Deployments(testNamespace).Create(dep)
		if err != nil {
			t.Fatalf("failed to create deployment: %s", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "karolisr/demo-webhook:0.0.2")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})

	t.Run("UpdateThroughPrivateGitlabPolling", func(t *testing.T) {

		user := os.Getenv("GITLAB_USERNAME")
		password := os.Getenv("GITLAB_PASSWORD")

		if user == "" || password == "" {
			fmt.Println("[X] Skipping UpdateThroughPrivateGitlabPolling test since GITLAB_USERNAME and/or GITLAB_PASSWORD env vars not set")
			t.Skip()
		}

		testNamespace := createNamespaceForTest()
		defer func() {
			err := deleteTestNamespace(testNamespace)
			if err != nil {
				log.WithFields(log.Fields{
					"error":     err,
					"namespace": testNamespace,
				}).Error("error while deleting test namespace")
			}
		}()

		payload, err := secrets.EncodeDockerCfgJson(&secrets.DockerCfg{
			"registry.gitlab.com": &secrets.Auth{
				Auth: secrets.EncodeBase64Secret(user, password),
			},
		})
		if err != nil {
			t.Fatalf("failed to encode docker cfg secret payload: %s", err)
		}

		secretName := "gitlab-registry-credentials"

		secret := &v1.Secret{
			TypeMeta: meta_v1.TypeMeta{},
			ObjectMeta: meta_v1.ObjectMeta{
				Name:        secretName,
				Namespace:   testNamespace,
				Labels:      map[string]string{},
				Annotations: map[string]string{},
			},
			Type: v1.SecretTypeDockerConfigJson,
			Data: map[string][]byte{
				".dockerconfigjson": []byte(payload),
			},
		}

		_, err = kcs.CoreV1().Secrets(testNamespace).Create(secret)
		if err != nil {
			t.Fatalf("failed to create secret: %s", err)
		} else {
			t.Logf("secret '%s' created in namespace '%s'", secret.Name, secret.Namespace)
		}

		dep := &apps_v1.Deployment{
			meta_v1.TypeMeta{},
			meta_v1.ObjectMeta{
				Name:      "deployment-1",
				Namespace: testNamespace,
				Labels: map[string]string{
					types.KeelPolicyLabel:  "major",
					types.KeelTriggerLabel: "poll",
				},
				Annotations: map[string]string{
					types.KeelPollScheduleAnnotation: "@every 2s",
				},
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
						ImagePullSecrets: []v1.LocalObjectReference{
							{
								Name: secretName,
							},
						},
						Containers: []v1.Container{
							{
								ImagePullPolicy: v1.PullAlways,
								Name:            "wd-1",
								Image:           "registry.gitlab.com/karolisr/keel:0.1.0",
							},
						},
					},
				},
			},
			apps_v1.DeploymentStatus{},
		}

		_, err = kcs.AppsV1().Deployments(testNamespace).Create(dep)
		if err != nil {
			t.Fatalf("failed to create deployment: %s", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		err = waitFor(ctx, kcs, testNamespace, dep.ObjectMeta.Name, "registry.gitlab.com/karolisr/keel:0.2.0")
		if err != nil {
			t.Errorf("update failed: %s", err)
		}
	})

}
