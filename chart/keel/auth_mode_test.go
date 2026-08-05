package keel_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func renderChart(t *testing.T, overrides map[string]interface{}) (map[string]string, error) {
	t.Helper()
	_, filename, _, _ := runtime.Caller(0)
	chart, err := loader.Load(filepath.Dir(filename))
	if err != nil {
		t.Fatalf("load chart: %v", err)
	}
	values, err := chartutil.CoalesceValues(chart, overrides)
	if err != nil {
		t.Fatalf("coalesce values: %v", err)
	}
	return engine.Render(chart, chartutil.Values{
		"Values": values,
		"Release": map[string]interface{}{
			"Name": "test", "Namespace": "keel", "Service": "Helm", "IsInstall": true,
		},
		"Chart":        chart.Metadata,
		"Capabilities": chartutil.DefaultCapabilities,
	})
}

func TestLegacyAuthChartDefaultsRemainUnchanged(t *testing.T) {
	rendered, err := renderChart(t, nil)
	if err != nil {
		t.Fatalf("render defaults: %v", err)
	}
	deployment := rendered["keel/templates/deployment.yaml"]
	if !strings.Contains(deployment, `value: "legacy"`) {
		t.Fatal("default deployment does not render AUTH_MODE=legacy")
	}
	if strings.Contains(deployment, "name: oauth2-proxy") {
		t.Fatal("default deployment unexpectedly contains oauth2-proxy")
	}
}

func TestExternalProxyChartUsesOnlyProxyServiceEntrypoint(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"auth":        map[string]interface{}{"mode": "external-proxy"},
		"oauth2Proxy": map[string]interface{}{"enabled": true, "existingSecret": "oauth2-proxy"},
		"service":     map[string]interface{}{"enabled": true, "type": "ClusterIP"},
	})
	if err != nil {
		t.Fatalf("render external proxy mode: %v", err)
	}
	deployment := rendered["keel/templates/deployment.yaml"]
	service := rendered["keel/templates/service.yaml"]
	for _, expected := range []string{"name: oauth2-proxy", "--upstream=http://127.0.0.1:9300", "name: AUTH_PROXY_USER_HEADER"} {
		if !strings.Contains(deployment, expected) {
			t.Fatalf("deployment missing %q", expected)
		}
	}
	if !strings.Contains(service, "targetPort: 4180") || strings.Contains(service, "targetPort: 9300") {
		t.Fatalf("external proxy Service must target only oauth2-proxy:\n%s", service)
	}
}

func TestUnsafeOrConflictingAuthChartValuesFail(t *testing.T) {
	tests := []struct {
		name      string
		overrides map[string]interface{}
		message   string
	}{
		{name: "unknown mode", overrides: map[string]interface{}{"auth": map[string]interface{}{"mode": "oidc"}}, message: "auth.mode must be one of"},
		{name: "external proxy without sidecar", overrides: map[string]interface{}{"auth": map[string]interface{}{"mode": "external-proxy"}}, message: "requires oauth2Proxy.enabled=true"},
		{name: "external proxy plus basic", overrides: map[string]interface{}{"auth": map[string]interface{}{"mode": "external-proxy"}, "basicauth": map[string]interface{}{"enabled": true, "user": "admin", "password": "secret"}}, message: "conflicts with basicauth"},
		{name: "sidecar outside proxy mode", overrides: map[string]interface{}{"oauth2Proxy": map[string]interface{}{"enabled": true}}, message: "requires auth.mode=external-proxy"},
		{name: "basic missing credentials", overrides: map[string]interface{}{"auth": map[string]interface{}{"mode": "basic"}, "basicauth": map[string]interface{}{"enabled": true}}, message: "non-empty basicauth"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := renderChart(t, tt.overrides)
			if err == nil || !strings.Contains(err.Error(), tt.message) {
				t.Fatalf("error = %v, want message containing %q", err, tt.message)
			}
		})
	}
}
