package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/keel-hq/keel/pkg/auth"
	"sigs.k8s.io/yaml"
)

type apiOperation struct {
	method      string
	path        string
	operationID string
}

var documentedAPIOperations = []apiOperation{
	{http.MethodGet, "/healthz", "healthCheck"},
	{http.MethodGet, "/version", "getVersion"},
	{http.MethodPost, "/v1/auth/login", "login"},
	{http.MethodGet, "/v1/auth/info", "getAuthInfo"},
	{http.MethodGet, "/v1/auth/user", "getAuthUser"},
	{http.MethodGet, "/v1/auth/logout", "logoutViaGet"},
	{http.MethodPost, "/v1/auth/logout", "logout"},
	{http.MethodGet, "/v1/auth/refresh", "refreshAuth"},
	{http.MethodGet, "/v1/approvals", "listApprovals"},
	{http.MethodPost, "/v1/approvals", "updateApproval"},
	{http.MethodPut, "/v1/approvals", "setResourceApprovals"},
	{http.MethodGet, "/v1/resources", "listResources"},
	{http.MethodPut, "/v1/policies", "updateResourcePolicy"},
	{http.MethodGet, "/v1/tracked", "listTrackedImages"},
	{http.MethodPut, "/v1/tracked", "updateTrackedImage"},
	{http.MethodGet, "/v1/audit", "listAuditLogs"},
	{http.MethodGet, "/v1/stats", "getStats"},
	{http.MethodPost, "/v1/webhooks/native", "receiveNativeWebhook"},
	{http.MethodPost, "/v1/webhooks/dockerhub", "receiveDockerHubWebhook"},
	{http.MethodPost, "/v1/webhooks/jfrog", "receiveJFrogWebhook"},
	{http.MethodPost, "/v1/webhooks/quay", "receiveQuayWebhook"},
	{http.MethodPost, "/v1/webhooks/azure", "receiveAzureWebhook"},
	{http.MethodPost, "/v1/webhooks/github", "receiveGitHubWebhook"},
	{http.MethodPost, "/v1/webhooks/harbor", "receiveHarborWebhook"},
	{http.MethodPost, "/v1/webhooks/registry", "receiveRegistryWebhook"},
}

var documentedAPIExclusions = []string{
	"OPTIONS and CORS preflight",
	"/metrics",
	"static UI assets and catch-all",
	"DEBUG-only /debug/vars and /debug/pprof routes",
}

func TestAPIRouteInventory(t *testing.T) {
	server := newInventoryServer(true, true)
	server.registerRoutes(server.router)

	want := make([]string, 0, len(documentedAPIOperations))
	for _, operation := range documentedAPIOperations {
		want = append(want, operation.method+" "+operation.path)
	}

	var got []string
	err := server.router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil || (path != "/healthz" && path != "/version" && !strings.HasPrefix(path, "/v1/")) {
			return nil
		}

		methods, err := route.GetMethods()
		if err != nil {
			return nil
		}
		for _, method := range methods {
			if method != http.MethodOptions {
				got = append(got, method+" "+path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk routes: %v", err)
	}

	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("documented operation count = %d, router operation count = %d\ndocumented: %v\nrouter: %v", len(want), len(got), want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("operation mismatch at %d: documented %q, router %q", i, want[i], got[i])
		}
	}

	if len(documentedAPIExclusions) != 4 {
		t.Fatalf("expected four documented exclusion categories, got %d", len(documentedAPIExclusions))
	}
}

func TestConditionalRouteAndWebhookAuthentication(t *testing.T) {
	tests := []struct {
		name                  string
		authEnabled           bool
		authenticatedWebhooks bool
		method                string
		path                  string
		body                  string
		wantStatus            int
	}{
		{"admin routes absent without authenticator", false, false, http.MethodPost, "/v1/auth/login", `{}`, http.StatusNotFound},
		{"admin route requires authorization", true, false, http.MethodGet, "/v1/approvals", "", http.StatusUnauthorized},
		{"native webhook is open by default", true, false, http.MethodPost, "/v1/webhooks/native", `{}`, http.StatusBadRequest},
		{"native webhook can require authorization", true, true, http.MethodPost, "/v1/webhooks/native", `{}`, http.StatusUnauthorized},
		{"registry webhook never requires authorization", true, true, http.MethodPost, "/v1/webhooks/registry", `{}`, http.StatusOK},
		{"registry malformed payload remains a bad request", true, true, http.MethodPost, "/v1/webhooks/registry", `{`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := newInventoryServer(tt.authEnabled, tt.authenticatedWebhooks)
			server.registerRoutes(server.router)

			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			resp := httptest.NewRecorder()
			server.router.ServeHTTP(resp, req)
			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body: %s", resp.Code, tt.wantStatus, resp.Body.String())
			}
		})
	}
}

func TestOpenAPIContract(t *testing.T) {
	document, err := os.ReadFile("../../docs/swagger.yaml")
	if err != nil {
		t.Fatalf("read Swagger document: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string `json:"operationId"`
		} `json:"paths"`
	}
	if err := yaml.Unmarshal(document, &spec); err != nil {
		t.Fatalf("parse Swagger document: %v", err)
	}

	want := make(map[string]string, len(documentedAPIOperations))
	for _, operation := range documentedAPIOperations {
		want[operation.method+" "+operation.path] = operation.operationID
	}

	got := make(map[string]string)
	for path, pathItem := range spec.Paths {
		for method, operation := range pathItem {
			method = strings.ToUpper(method)
			switch method {
			case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodHead:
				got[method+" "+path] = operation.OperationID
			}
		}
	}

	if len(got) != len(want) {
		t.Fatalf("Swagger operation count = %d, want %d", len(got), len(want))
	}

	keys := make([]string, 0, len(want))
	for key := range want {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		operationID, ok := got[key]
		if !ok {
			t.Errorf("Swagger operation missing: %s -> %s", key, want[key])
			continue
		}
		if operationID != want[key] {
			t.Errorf("Swagger operation ID for %s = %q, want %q", key, operationID, want[key])
			continue
		}
		t.Logf("API operation: %s -> %s", key, operationID)
	}

	for _, exclusion := range documentedAPIExclusions {
		t.Logf("Excluded from application API: %s", exclusion)
	}
}

func newInventoryServer(authEnabled, authenticatedWebhooks bool) *TriggerServer {
	opts := &auth.Opts{}
	if authEnabled {
		opts.Username = "admin"
		opts.Password = "password"
	}

	return NewTriggerServer(&Opts{
		Authenticator:         auth.New(opts),
		AuthenticatedWebhooks: authenticatedWebhooks,
	})
}
