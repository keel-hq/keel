package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIndexHandlerServesSPAEntryForDirectRoute(t *testing.T) {
	uiDir := t.TempDir()
	index := "<!doctype html><html><body><div id=\"root\"></div></body></html>"
	if err := os.WriteFile(filepath.Join(uiDir, "index.html"), []byte(index), 0o600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/approvals", nil)
	resp := httptest.NewRecorder()
	indexHandler(uiDir)(resp, req)

	if resp.Code != http.StatusOK || !strings.Contains(resp.Body.String(), "id=\"root\"") {
		t.Fatalf("direct SPA route response = %d %q", resp.Code, resp.Body.String())
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if got := resp.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q, want no-cache", got)
	}
}

func TestStaticAssetHandlerServesViteAssets(t *testing.T) {
	uiDir := t.TempDir()
	assetDir := filepath.Join(uiDir, "assets")
	if err := os.Mkdir(assetDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "index-abc123.js"), []byte("export default true"), 0o600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/assets/index-abc123.js", nil)
	resp := httptest.NewRecorder()
	staticAssetHandler(uiDir).ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("asset status = %d", resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); !strings.Contains(got, "javascript") {
		t.Fatalf("Content-Type = %q, want JavaScript", got)
	}
	if got := resp.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control = %q", got)
	}
}
