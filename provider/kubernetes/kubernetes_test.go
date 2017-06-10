package kubernetes

import (
	"fmt"
	"time"

	"github.com/rusenask/keel/types"
	"github.com/rusenask/keel/util/version"

	"testing"
)

var currentVersion = "0.0.2"
var newVersion = "0.0.3"

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

func TestGetImageName(t *testing.T) {
	name := versionreg.ReplaceAllString("gcr.io/v2-namespace/hello-world:1.1", "")
	fmt.Println(name)
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
	found := false
	for _, c := range deps[0].Spec.Template.Spec.Containers {
		ver, err := version.GetVersionFromImageName(c.Image)
		if err != nil {
			continue
		}
		if ver.String() == repo.Tag {
			found = true
		}
	}
	// fmt.Println(len(deps.Items))
	fmt.Println(len(deps))
	fmt.Println(found)

}
func TestProcessEvent(t *testing.T) {
	provider, err := NewProvider(&Opts{ConfigPath: ".kubeconfig"})
	if err != nil {
		t.Fatalf("failed to get provider: %s", err)
	}

	repo := types.Repository{
		Name: "karolisr/webhook-demo",
		Tag:  newVersion,
	}

	event := &types.Event{Repository: repo}
	updated, err := provider.processEvent(event)
	if err != nil {
		t.Errorf("got error while processing event: %s", err)
	}

	//
	time.Sleep(100 * time.Millisecond)
	for _, upd := range updated {
		current, err := provider.getDeployment(upd.Namespace, upd.Name)
		if err != nil {
			t.Fatalf("failed to get deployment %s, error: %s", upd.Name, err)
		}
		currentVer, err := version.GetVersionFromImageName(current.Spec.Template.Spec.Containers[0].Image)
		if err != nil {
			t.Fatalf("failed to get version from %s, error: %s", current.Spec.Template.Spec.Containers[0].Image, err)
		}

		if currentVer.String() != newVersion {
			t.Errorf("deployment version wasn't updated, got: %s while expected: %s", currentVer.String(), newVersion)
		}
	}

}
