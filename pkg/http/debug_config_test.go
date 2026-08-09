package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebugRoutesUseTypedConfig(t *testing.T) {
	for _, tt := range []struct {
		name       string
		debug      bool
		wantStatus int
	}{
		{name: "enabled", debug: true, wantStatus: http.StatusOK},
		{name: "disabled", debug: false, wantStatus: http.StatusNotFound},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := NewTriggerServer(&Opts{Debug: tt.debug})
			server.registerRoutes(server.router)
			response := httptest.NewRecorder()
			server.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/debug/vars", nil))
			if response.Code != tt.wantStatus {
				t.Fatalf("GET /debug/vars status = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}
