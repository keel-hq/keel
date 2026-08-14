package docker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTagsSupportsHarborProjectPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/library/ai-rag/tags/list" {
			t.Errorf("unexpected registry path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(tagsResponse{Tags: []string{"latest", "1.2.3"}})
	}))
	defer server.Close()

	tags, err := New(server.URL, "", "").Tags("library/ai-rag")
	if err != nil {
		t.Fatalf("failed to get Harbor repository tags: %s", err)
	}
	if len(tags) != 2 || tags[0] != "latest" || tags[1] != "1.2.3" {
		t.Fatalf("unexpected tags: %v", tags)
	}
}

func TestGetDigestDockerHub(t *testing.T) {
	client := New("https://index.docker.io", "", "")

	tags, err := client.Tags("karolisr/keel")
	if err != nil {
		t.Errorf("failed to get tags, error: %s", err)
	}

	if len(tags) == 0 {
		t.Errorf("no tags?")
	}
}
