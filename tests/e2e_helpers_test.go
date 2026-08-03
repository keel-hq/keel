package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var invalidKubernetesName = regexp.MustCompile(`[^a-z0-9-]+`)

func (s *E2ESuite) SetupTest() {
	prefix := invalidKubernetesName.ReplaceAllString(strings.ToLower(strings.ReplaceAll(s.T().Name(), "/", "-")), "-")
	if len(prefix) > 35 {
		prefix = prefix[len(prefix)-35:]
	}
	namespace, err := s.client.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix + "-",
			Labels:       map[string]string{e2eRunLabel: s.cfg.runID, "keel.sh/e2e-test": prefix},
		},
	}, metav1.CreateOptions{})
	s.Require().NoError(err)
	s.testNamespace = namespace.Name
}

func (s *E2ESuite) TearDownTest() {
	if s.testNamespace == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	s.Require().NoError(s.client.CoreV1().Namespaces().Delete(ctx, s.testNamespace, metav1.DeleteOptions{}))
	s.Require().Eventually(func() bool {
		_, err := s.client.CoreV1().Namespaces().Get(ctx, s.testNamespace, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}, 60*time.Second, 500*time.Millisecond, "test namespace %q was not deleted", s.testNamespace)
	s.testNamespace = ""
}

func waitForDeploymentImage(ctx context.Context, client kubernetes.Interface, namespace, name, desired string) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	last := "<not observed>"
	for {
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil && len(deployment.Spec.Template.Spec.Containers) > 0 {
			last = deployment.Spec.Template.Spec.Containers[0].Image
			if last == desired {
				return nil
			}
		} else if err != nil {
			last = "get error: " + err.Error()
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("deployment %s/%s: expected image %q; last observed %q: %w", namespace, name, desired, last, ctx.Err())
		case <-ticker.C:
		}
	}
}

func ensureDeploymentImageUnchanged(ctx context.Context, client kubernetes.Interface, namespace, name, expected string) error {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	last := "<not observed>"
	for {
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if ctx.Err() == nil {
				last = "get error: " + err.Error()
			}
		} else if len(deployment.Spec.Template.Spec.Containers) == 0 {
			last = "deployment has no containers"
		} else {
			last = deployment.Spec.Template.Spec.Containers[0].Image
			if last != expected {
				return fmt.Errorf("deployment %s/%s: expected image to remain %q; observed %q", namespace, name, expected, last)
			}
		}
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded && last == expected {
				return nil
			}
			return fmt.Errorf("deployment %s/%s: could not verify unchanged image %q; last observed %q: %w", namespace, name, expected, last, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *E2ESuite) requireRegistryTags(repository string, expected ...string) {
	response, err := (&http.Client{Timeout: 5 * time.Second}).Get("http://" + s.cfg.registry + "/v2/" + s.cfg.runID + "/" + repository + "/tags/list")
	s.Require().NoError(err, "query tags for isolated repository %s", repository)
	defer response.Body.Close()
	s.Require().Equal(http.StatusOK, response.StatusCode, "query tags for isolated repository %s", repository)
	var result struct {
		Tags []string `json:"tags"`
	}
	s.Require().NoError(json.NewDecoder(response.Body).Decode(&result))
	sort.Strings(result.Tags)
	sort.Strings(expected)
	s.Require().Equal(expected, result.Tags, "repository %s must contain only scenario-owned tags", repository)
}
