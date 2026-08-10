package keel_test

import (
	"strings"
	"testing"
)

func TestStartupProbeProtectsKeelFromLivenessRestarts(t *testing.T) {
	rendered, err := renderChart(t, nil)
	if err != nil {
		t.Fatalf("render defaults: %v", err)
	}
	deployment := rendered["keel/templates/deployment.yaml"]
	for _, expected := range []string{
		"startupProbe:",
		"path: /healthz",
		"port: 9300",
		"periodSeconds: 5",
		"failureThreshold: 30",
	} {
		if !strings.Contains(deployment, expected) {
			t.Fatalf("default deployment startup probe missing %q:\n%s", expected, deployment)
		}
	}
}

func TestStartupProbeCanBeDisabled(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"startupProbe": map[string]interface{}{"enabled": false},
	})
	if err != nil {
		t.Fatalf("render with startup probe disabled: %v", err)
	}
	if strings.Contains(rendered["keel/templates/deployment.yaml"], "startupProbe:") {
		t.Fatal("deployment unexpectedly contains a disabled startup probe")
	}
}

func TestExternalProxyStartupProbeUsesProxyEndpoint(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"auth":        map[string]interface{}{"mode": "external-proxy"},
		"oauth2Proxy": map[string]interface{}{"enabled": true, "existingSecret": "oauth2-proxy"},
		"service":     map[string]interface{}{"enabled": true, "type": "ClusterIP"},
	})
	if err != nil {
		t.Fatalf("render external proxy mode: %v", err)
	}
	deployment := rendered["keel/templates/deployment.yaml"]
	startup := deployment[strings.Index(deployment, "startupProbe:"):strings.Index(deployment, "livenessProbe:")]
	if !strings.Contains(startup, "path: /ping") || !strings.Contains(startup, "port: 4180") {
		t.Fatalf("external-proxy startup probe does not target oauth2-proxy:\n%s", startup)
	}
}
