package keel_test

import (
	"encoding/json"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/yaml"
)

// renderDeployment fails on non-string label values or priorityClassName, as the API server would.
func renderDeployment(t *testing.T, overrides map[string]interface{}) appsv1.Deployment {
	t.Helper()
	rendered, err := renderChart(t, overrides)
	if err != nil {
		t.Fatalf("render chart: %v", err)
	}
	// YAMLToJSON keeps scalar types, unlike yaml.Unmarshal which coerces them to the target field type
	raw, err := yaml.YAMLToJSON([]byte(rendered["keel/templates/deployment.yaml"]))
	if err != nil {
		t.Fatalf("convert deployment: %v", err)
	}
	var deployment appsv1.Deployment
	if err := json.Unmarshal(raw, &deployment); err != nil {
		t.Fatalf("parse deployment: %v", err)
	}
	return deployment
}

func assertDefaultPodLabels(t *testing.T, labels map[string]string) {
	t.Helper()
	if len(labels) != 2 || labels["app"] != "keel" || labels["release"] != "test" {
		t.Fatalf("default pod labels changed: %v", labels)
	}
}

func TestPodLabelsAndPriorityClassDefaultsRenderNothing(t *testing.T) {
	deployment := renderDeployment(t, nil)
	assertDefaultPodLabels(t, deployment.Spec.Template.Labels)
	if deployment.Spec.Template.Spec.PriorityClassName != "" {
		t.Fatalf("default deployment unexpectedly sets priorityClassName %q", deployment.Spec.Template.Spec.PriorityClassName)
	}
}

func TestNullPodLabelsRender(t *testing.T) {
	deployment := renderDeployment(t, map[string]interface{}{"podLabels": nil})
	assertDefaultPodLabels(t, deployment.Spec.Template.Labels)
}

func TestPodLabelsAreAddedToThePodTemplate(t *testing.T) {
	deployment := renderDeployment(t, map[string]interface{}{
		"podLabels": map[string]interface{}{"team": "platform", "example.com/component-group": "platform"},
	})
	labels := deployment.Spec.Template.Labels
	if labels["team"] != "platform" || labels["example.com/component-group"] != "platform" {
		t.Fatalf("pod labels missing: %v", labels)
	}
	if selector := deployment.Spec.Selector.MatchLabels; len(selector) != 2 {
		t.Fatalf("pod labels leaked into the selector: %v", selector)
	}
}

func TestPodLabelValuesAreRenderedAsStrings(t *testing.T) {
	labels := renderDeployment(t, map[string]interface{}{
		"podLabels": map[string]interface{}{"enabled": true, "tier": 2},
	}).Spec.Template.Labels
	if labels["enabled"] != "true" || labels["tier"] != "2" {
		t.Fatalf("non-string label values not rendered as strings: %v", labels)
	}
}

func TestPodLabelsCannotSetSelectorLabels(t *testing.T) {
	for _, key := range []string{"app", "release"} {
		_, err := renderChart(t, map[string]interface{}{
			"podLabels": map[string]interface{}{key: "other"},
		})
		if err == nil || !strings.Contains(err.Error(), "podLabels cannot set \""+key+"\"") {
			t.Fatalf("podLabels.%s: expected a render error, got %v", key, err)
		}
	}
}

func TestPriorityClassNameIsSetOnThePodSpec(t *testing.T) {
	for _, name := range []string{"platform-high", "1000"} {
		spec := renderDeployment(t, map[string]interface{}{"priorityClassName": name}).Spec.Template.Spec
		if spec.PriorityClassName != name {
			t.Fatalf("priorityClassName = %q, want %q", spec.PriorityClassName, name)
		}
	}
}
