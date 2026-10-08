package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
)

// tagListRegistry serves the tag list of a single repository "app", two tags
// per page. With lexical set it sorts the tags as the distribution spec
// requires; otherwise it keeps push order and treats `last` as a cursor into
// that order, the way ghcr.io does.
type tagListRegistry struct {
	tags         []string
	lexical      bool
	manifestCode int // status for manifest HEAD requests of known tags, 200 when zero
}

func (reg *tagListRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/v2/app/manifests/") {
		tag := strings.TrimPrefix(r.URL.Path, "/v2/app/manifests/")
		if reg.index(tag) < 0 {
			http.NotFound(w, r)
			return
		}
		if reg.manifestCode != 0 && reg.manifestCode != http.StatusOK {
			w.WriteHeader(reg.manifestCode)
			return
		}
		w.Header().Set("Docker-Content-Digest", "sha256:0604af35299dd37ff23937d115d103532948b568a9dd8197d14c256a8ab8b0bb")
		return
	}
	if r.URL.Path != "/v2/app/tags/list" {
		http.NotFound(w, r)
		return
	}

	tags := reg.ordered()
	start := 0
	if last := r.URL.Query().Get("last"); last != "" {
		if reg.lexical {
			start = sort.SearchStrings(tags, last)
			if start < len(tags) && tags[start] == last {
				start++
			}
		} else if i := reg.index(last); i >= 0 {
			start = i + 1
		} else {
			start = len(tags)
		}
	}
	end := start + 2
	if end > len(tags) {
		end = len(tags)
	}
	page := tags[start:end]
	if end < len(tags) {
		w.Header().Set("Link", fmt.Sprintf(`</v2/app/tags/list?last=%s&n=2>; rel="next"`, url.QueryEscape(page[len(page)-1])))
	}
	_ = json.NewEncoder(w).Encode(map[string][]string{"tags": page})
}

func (reg *tagListRegistry) ordered() []string {
	tags := append([]string(nil), reg.tags...)
	if reg.lexical {
		sort.Strings(tags)
	}
	return tags
}

func (reg *tagListRegistry) index(tag string) int {
	for i, t := range reg.ordered() {
		if t == tag {
			return i
		}
	}
	return -1
}

func getAfter(t *testing.T, reg *tagListRegistry, after string) ([]string, error) {
	t.Helper()
	server := httptest.NewServer(reg)
	defer server.Close()

	repo, err := New().Get(Opts{Registry: server.URL, Name: "app", After: after})
	if err != nil {
		return nil, err
	}
	return repo.Tags, nil
}

func TestGetAfter(t *testing.T) {
	pushOrdered := []string{"v1.0.0", "main", "pr-1", "v1.1.0", "v1.0.1"}

	tests := []struct {
		name  string
		reg   *tagListRegistry
		after string
		want  []string
	}{
		{
			name:  "push-ordered registry returns only tags pushed after the cursor",
			reg:   &tagListRegistry{tags: pushOrdered},
			after: "v1.0.0",
			want:  []string{"main", "pr-1", "v1.1.0", "v1.0.1"},
		},
		{
			name:  "push-ordered registry with nothing newer returns no tags",
			reg:   &tagListRegistry{tags: pushOrdered},
			after: "v1.0.1",
			want:  nil,
		},
		{
			name:  "unknown cursor falls back to listing every tag",
			reg:   &tagListRegistry{tags: pushOrdered},
			after: "v0.9.0",
			want:  pushOrdered,
		},
		{
			name:  "lexical registry falls back to listing every tag",
			reg:   &tagListRegistry{tags: []string{"v1.0.0", "v1.9.0", "v1.10.0", "v1.9.1"}, lexical: true},
			after: "v1.0.0",
			want:  []string{"v1.0.0", "v1.10.0", "v1.9.0", "v1.9.1"},
		},
		{
			name:  "lexical registry with the cursor sorting last still finds newer versions",
			reg:   &tagListRegistry{tags: []string{"v1.0.0", "v1.9.0", "v1.10.0"}, lexical: true},
			after: "v1.9.0",
			want:  []string{"v1.0.0", "v1.10.0", "v1.9.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getAfter(t, tt.reg, tt.after)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAfterManifestErrorIsReturned(t *testing.T) {
	reg := &tagListRegistry{tags: []string{"v1.0.0", "main", "v1.0.1"}, manifestCode: http.StatusTooManyRequests}

	if _, err := getAfter(t, reg, "v1.0.1"); err == nil {
		t.Fatalf("expected the manifest error to be returned instead of a full tag walk")
	}
}
