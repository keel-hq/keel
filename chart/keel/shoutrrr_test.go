package keel_test

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

var shoutrrrURLsRe = regexp.MustCompile(`SHOUTRRR_URLS:\s*(\S+)`)

// decodeShoutrrrURLs pulls SHOUTRRR_URLS out of the rendered secret and decodes it.
func decodeShoutrrrURLs(t *testing.T, secret string) string {
	t.Helper()

	match := shoutrrrURLsRe.FindStringSubmatch(secret)
	if match == nil {
		t.Fatalf("secret does not contain SHOUTRRR_URLS:\n%s", secret)
	}

	decoded, err := base64.StdEncoding.DecodeString(match[1])
	if err != nil {
		t.Fatalf("decode SHOUTRRR_URLS: %v", err)
	}

	return string(decoded)
}

func TestShoutrrrDisabledByDefault(t *testing.T) {
	rendered, err := renderChart(t, nil)
	if err != nil {
		t.Fatalf("render defaults: %v", err)
	}

	if strings.Contains(rendered["keel/templates/secret.yaml"], "SHOUTRRR_URLS") {
		t.Error("SHOUTRRR_URLS rendered while shoutrrr is disabled")
	}
	if strings.Contains(rendered["keel/templates/deployment.yaml"], "SHOUTRRR_TIMEOUT") {
		t.Error("SHOUTRRR_TIMEOUT rendered while shoutrrr is disabled")
	}
}

// URLs must be newline separated, since shoutrrr uses commas within a single URL.
func TestShoutrrrURLListJoinedWithNewlines(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"shoutrrr": map[string]interface{}{
			"enabled": true,
			"urls": []interface{}{
				"ntfy://ntfy.sh/keel",
				"telegram://token@telegram?chats=111,222",
			},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := decodeShoutrrrURLs(t, rendered["keel/templates/secret.yaml"])
	want := "ntfy://ntfy.sh/keel\ntelegram://token@telegram?chats=111,222"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A plain string is accepted too, so operators can supply URLs from an existing value.
func TestShoutrrrURLsAcceptsString(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"shoutrrr": map[string]interface{}{
			"enabled": true,
			"urls":    "ntfy://ntfy.sh/keel discord://token@channel",
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := decodeShoutrrrURLs(t, rendered["keel/templates/secret.yaml"])
	want := "ntfy://ntfy.sh/keel discord://token@channel"

	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Service URLs carry credentials, so they belong in the Secret and never in the
// Deployment, which is readable by anyone who can get pods.
func TestShoutrrrURLsNeverRenderIntoDeployment(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"shoutrrr": map[string]interface{}{
			"enabled": true,
			"urls":    []interface{}{"discord://sup3rs3cret@123456"},
			"timeout": "30s",
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	deployment := rendered["keel/templates/deployment.yaml"]

	if strings.Contains(deployment, "sup3rs3cret") {
		t.Errorf("deployment leaks a shoutrrr credential:\n%s", deployment)
	}
	if strings.Contains(deployment, "SHOUTRRR_URLS") {
		t.Error("SHOUTRRR_URLS must be delivered through the secret, not the deployment")
	}

	// The timeout holds no secret, so it is a plain env var.
	if !strings.Contains(deployment, "SHOUTRRR_TIMEOUT") || !strings.Contains(deployment, `value: "30s"`) {
		t.Errorf("deployment missing SHOUTRRR_TIMEOUT=30s:\n%s", deployment)
	}
}

func TestShoutrrrTimeoutOmittedWhenUnset(t *testing.T) {
	rendered, err := renderChart(t, map[string]interface{}{
		"shoutrrr": map[string]interface{}{
			"enabled": true,
			"urls":    []interface{}{"ntfy://ntfy.sh/keel"},
		},
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	if strings.Contains(rendered["keel/templates/deployment.yaml"], "SHOUTRRR_TIMEOUT") {
		t.Error("SHOUTRRR_TIMEOUT rendered despite being unset")
	}
}
