package k8s

import (
	"testing"

	"github.com/keel-hq/keel/pkg/config"
	v1 "k8s.io/api/core/v1"
)

func TestNamespaceForUsesTypedConfiguration(t *testing.T) {
	t.Setenv("RESTRICTED_NAMESPACE", "environment")
	for _, tt := range []struct {
		name string
		cfg  config.KubernetesConfig
		want string
	}{
		{"empty scans all", config.KubernetesConfig{}, v1.NamespaceAll},
		{"keel scans all", config.KubernetesConfig{RestrictedNamespace: "keel"}, v1.NamespaceAll},
		{"configured namespace", config.KubernetesConfig{RestrictedNamespace: "typed"}, "typed"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := namespaceFor(tt.cfg); got != tt.want {
				t.Fatalf("namespaceFor(%#v) = %q, want %q", tt.cfg, got, tt.want)
			}
		})
	}
}
