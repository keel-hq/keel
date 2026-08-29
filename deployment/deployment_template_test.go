package deployment

import (
	"os"
	"strings"
	"testing"

	rbac_v1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/yaml"
)

func TestDeploymentTemplateGrantsNodeMetadataAccess(t *testing.T) {
	contents, err := os.ReadFile("deployment-template.yaml")
	if err != nil {
		t.Fatal(err)
	}

	var clusterRole rbac_v1.ClusterRole
	found := false
	for _, document := range strings.Split(string(contents), "---") {
		if !strings.Contains(document, "kind: ClusterRole\n") {
			continue
		}
		if err := yaml.Unmarshal([]byte(document), &clusterRole); err != nil {
			t.Fatalf("parse ClusterRole: %v", err)
		}
		found = true
		break
	}
	if !found {
		t.Fatal("deployment template does not contain a ClusterRole")
	}

	for _, rule := range clusterRole.Rules {
		if contains(rule.APIGroups, "") && contains(rule.Resources, "nodes") &&
			contains(rule.Verbs, "get") && contains(rule.Verbs, "list") && contains(rule.Verbs, "watch") {
			return
		}
	}
	t.Fatal("deployment template ClusterRole must grant get, list, and watch on core/v1 nodes")
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
