package pubsub

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/keel-hq/keel/util/image"
)

func unsafeImageRef(img string) *image.Reference {
	ref, err := image.Parse(img)
	if err != nil {
		panic(err)
	}
	return ref
}

func Test_isGoogleArtifactRegistry(t *testing.T) {
	type args struct {
		registry string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "gcr",
			args: args{registry: unsafeImageRef("gcr.io/v2-namespace/hello-world:1.1").Registry()},
			want: true,
		},
		{
			name: "google artifact registry",
			args: args{registry: unsafeImageRef("europe-west3-docker.pkg.dev/v2-namespace/hello-world:1.1").Registry()},
			want: true,
		},
		{
			name: "docker registry",
			args: args{registry: unsafeImageRef("docker.io/v2-namespace/hello-world:1.1").Registry()},
			want: false,
		},
		{
			name: "custom registry",
			args: args{registry: unsafeImageRef("localhost:4000/v2-namespace/hello-world:1.1").Registry()},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGoogleArtifactRegistry(tt.args.registry); got != tt.want {
				t.Errorf("isGoogleArtifactRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClusterName(t *testing.T) {

	cn := "my-cluster-x"

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.WriteHeader(200)
		resp.Write([]byte(cn))
	}

	// create test server with handler
	ts := httptest.NewServer(http.HandlerFunc(handler))
	defer ts.Close()

	name, err := getClusterName(ts.URL)
	if err != nil {
		t.Errorf("unexpected error while getting cluster name")
	}

	if name != cn {
		t.Errorf("unexpected cluster name: %s", name)
	}
}

func TestGetContainerRegistryURI(t *testing.T) {

	name := containerRegistrySubName("", "project-1", "topic-1")

	if name != "keel-unknown-project-1-topic-1" {
		t.Errorf("unexpected topic name: %s", name)
	}
}

func TestGetContainerRegistryURIWithClusterNameSet(t *testing.T) {

	name := containerRegistrySubName("testxxx", "project-1", "topic-1")

	if name != "keel-testxxx-project-1-topic-1" {
		t.Errorf("unexpected topic name: %s", name)
	}
}
