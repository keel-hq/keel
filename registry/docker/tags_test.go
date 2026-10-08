package docker

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestTagsAfterFollowsCursorAcrossPages(t *testing.T) {
	var lasts []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/app/tags/list" {
			t.Errorf("unexpected registry path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		last := r.URL.Query().Get("last")
		lasts = append(lasts, last)
		switch last {
		case "v1.0.0":
			w.Header().Set("Link", `</v2/app/tags/list?last=pr-2&n=2>; rel="next"`)
			_ = json.NewEncoder(w).Encode(tagsResponse{Tags: []string{"pr-1", "pr-2"}})
		case "pr-2":
			_ = json.NewEncoder(w).Encode(tagsResponse{Tags: []string{"v1.1.0"}})
		default:
			t.Errorf("unexpected cursor %q", last)
		}
	}))
	defer server.Close()

	tags, err := New(server.URL, "", "").TagsAfter("app", "v1.0.0")
	if err != nil {
		t.Fatalf("failed to get tags after cursor: %s", err)
	}
	if strings.Join(tags, ",") != "pr-1,pr-2,v1.1.0" {
		t.Fatalf("unexpected tags: %v", tags)
	}
	if strings.Join(lasts, ",") != "v1.0.0,pr-2" {
		t.Fatalf("unexpected cursors requested: %v", lasts)
	}
}

func TestTagsAfterEmptyPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(tagsResponse{Tags: []string{}})
	}))
	defer server.Close()

	tags, err := New(server.URL, "", "").TagsAfter("app", "v1.0.0")
	if err != nil {
		t.Fatalf("failed to get tags after cursor: %s", err)
	}
	if len(tags) != 0 {
		t.Fatalf("expected no tags, got: %v", tags)
	}
}
