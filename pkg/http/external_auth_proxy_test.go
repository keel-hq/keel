package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keel-hq/keel/pkg/auth"
)

func newExternalProxyServer() *TriggerServer {
	return NewTriggerServer(&Opts{
		Authenticator:       auth.New(&auth.Opts{}),
		AuthMode:            auth.ModeExternalProxy,
		AuthProxyUserHeader: auth.DefaultProxyUserHeader,
		AuthProxyLogoutURL:  auth.DefaultProxyLogoutURL,
	})
}

func TestExternalProxyAdminAuthorization(t *testing.T) {
	server := newExternalProxyServer()
	server.registerRoutes(server.router)

	tests := []struct {
		name       string
		path       string
		identity   string
		basic      bool
		wantStatus int
	}{
		{name: "missing forwarded identity", path: "/v1/auth/user", wantStatus: http.StatusUnauthorized},
		{name: "basic credentials do not substitute for proxy identity", path: "/v1/auth/user", basic: true, wantStatus: http.StatusUnauthorized},
		{name: "forwarded identity authorizes API", path: "/v1/auth/user", identity: "alice@example.test", wantStatus: http.StatusOK},
		{name: "second admin path uses same middleware", path: "/v1/auth/info", identity: "alice@example.test", wantStatus: http.StatusOK},
		{name: "control characters rejected", path: "/v1/auth/user", identity: "alice\x7f", wantStatus: http.StatusUnauthorized},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			if tt.identity != "" {
				req.Header.Set(auth.DefaultProxyUserHeader, tt.identity)
			}
			if tt.basic {
				req.SetBasicAuth("admin", "secret")
			}
			resp := httptest.NewRecorder()
			server.router.ServeHTTP(resp, req)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
			if tt.path == "/v1/auth/user" && tt.wantStatus == http.StatusOK && !strings.Contains(resp.Body.String(), `"name":"alice@example.test"`) {
				t.Fatalf("forwarded user missing from response: %s", resp.Body.String())
			}
		})
	}
}

func TestExternalProxyDoesNotRegisterLocalLoginOrRefresh(t *testing.T) {
	server := newExternalProxyServer()
	server.registerRoutes(server.router)
	for _, endpoint := range []struct{ method, path string }{{http.MethodPost, "/v1/auth/login"}, {http.MethodGet, "/v1/auth/refresh"}} {
		req := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(`{}`))
		req.Header.Set(auth.DefaultProxyUserHeader, "alice")
		resp := httptest.NewRecorder()
		server.router.ServeHTTP(resp, req)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want 404", endpoint.method, endpoint.path, resp.Code)
		}
	}
}

func TestExternalProxyApprovalAttributionUsesAuthenticatedIdentity(t *testing.T) {
	server := newExternalProxyServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/approvals", nil)
	req = auth.SetAuthenticationDetails(req, &auth.User{Username: "alice"})
	if got := server.approvalVoter(req, "spoofed-client-voter"); got != "alice" {
		t.Fatalf("approval voter = %q, want authenticated proxy identity", got)
	}

	legacy := NewTriggerServer(&Opts{Authenticator: auth.New(&auth.Opts{Username: "admin", Password: "secret"})})
	if got := legacy.approvalVoter(req, "legacy-client-voter"); got != "legacy-client-voter" {
		t.Fatalf("legacy approval voter = %q, want existing client-supplied behavior", got)
	}
}
