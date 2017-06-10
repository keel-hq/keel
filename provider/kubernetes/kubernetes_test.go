package kubernetes

import (
	"fmt"

	"github.com/rusenask/keel/types"

	"testing"
)

func TestGetNamespaces(t *testing.T) {
	provider, err := NewProvider(&Opts{ConfigPath: ".kubeconfig"})
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	namespaces, err := provider.namespaces()
	if err != nil {
		t.Errorf("failed to get namespaces: %s", err)
	}

	fmt.Println(namespaces.Items)
}

func TestGetDeployments(t *testing.T) {
	provider, err := NewProvider(&Opts{ConfigPath: ".kubeconfig"})
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	deps, err := provider.deployments()
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}
	// fmt.Println(len(deps.Items))
	fmt.Println(deps)
}

func TestGetImpacted(t *testing.T) {
	provider, err := NewProvider(&Opts{ConfigPath: ".kubeconfig"})
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	repo := &types.Repository{
		Name: "karolisr/webhook-demo",
		Tag:  "0.0.3",
	}

	deps, err := provider.impactedDeployments(repo)
	if err != nil {
		t.Errorf("failed to get deployments: %s", err)
	}
	// fmt.Println(len(deps.Items))
	fmt.Println(deps)

}
