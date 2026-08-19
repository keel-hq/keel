package http

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/pkg/auth"
	"github.com/keel-hq/keel/provider"

	log "github.com/sirupsen/logrus"
)

type capturingLogHook struct {
	mu    sync.Mutex
	lines []string
}

func (h *capturingLogHook) Fire(entry *log.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.lines = append(h.lines, line)
	return nil
}

func (h *capturingLogHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *capturingLogHook) Output() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return strings.Join(h.lines, "\n")
}

// captureLogEntries sets the global logrus logger to level, records every
// emitted entry and returns a reader for what was captured. The previous
// level and hooks are restored when the test finishes.
func captureLogEntries(t *testing.T, level log.Level) func() string {
	t.Helper()
	std := log.StandardLogger()
	prevLevel := std.GetLevel()
	prevHooks := cloneHooks(std.Hooks)
	std.SetLevel(level)
	hook := &capturingLogHook{}
	std.AddHook(hook)
	t.Cleanup(func() {
		std.SetLevel(prevLevel)
		std.Hooks = prevHooks
	})
	return hook.Output
}

// cloneHooks deep-copies the level hooks so they can be restored later.
func cloneHooks(hooks log.LevelHooks) log.LevelHooks {
	clone := make(log.LevelHooks, len(hooks))
	for lvl, hs := range hooks {
		copied := make([]log.Hook, len(hs))
		copy(copied, hs)
		clone[lvl] = copied
	}
	return clone
}

// newAuthenticatedWebhookTestingServer builds a trigger server with webhook
// authentication enabled, like a deployment that protects its webhook
// endpoints with basic/bearer auth.
func newAuthenticatedWebhookTestingServer(fp provider.Provider) (*TriggerServer, func()) {
	store, teardown := NewTestingUtils()

	am := approvals.New(&approvals.Opts{
		Store: store,
	})

	authenticator := auth.New(&auth.Opts{
		Username: "user-1",
		Password: "correct-horse-battery-staple",
	})

	providers := provider.New([]provider.Provider{fp}, am)
	srv := NewTriggerServer(&Opts{
		Providers:             providers,
		ApprovalManager:       am,
		Authenticator:         authenticator,
		Store:                 store,
		AuthenticatedWebhooks: true,
	})
	srv.registerRoutes(srv.router)

	return srv, teardown
}

// TestWebhookHandlerNeverLogsCredentials processes webhook requests that
// carry credentials and proves that none of the captured log output contains
// the credentials, at info level and at debug level.
func TestWebhookHandlerNeverLogsCredentials(t *testing.T) {
	fp := &fakeProvider{}
	srv, teardown := newAuthenticatedWebhookTestingServer(fp)
	defer teardown()

	const basicPassword = "S3cr3t-Basic-P@ssw0rd-9f86d0"
	const bearerToken = "S3cr3t-Bearer-T0ken-4b7c1e"
	payload := `{"name": "gcr.io/v2-namespace/hello-world", "tag": "1.1.1"}`

	for _, level := range []log.Level{log.InfoLevel, log.DebugLevel} {
		t.Run(level.String(), func(t *testing.T) {
			out := captureLogEntries(t, level)

			// Basic auth with a wrong password exercises the failure path
			// that used to log the password verbatim.
			req, err := http.NewRequest("POST", "/v1/webhooks/native", bytes.NewBufferString(payload))
			if err != nil {
				t.Fatalf("failed to create request: %s", err)
			}
			req.Header.Set("Authorization",
				"Basic "+base64.StdEncoding.EncodeToString([]byte("user-1:"+basicPassword)))
			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("basic auth: expected 401, got %d", rec.Code)
			}

			// Bearer auth with an unknown token exercises the failure path
			// that used to log the token verbatim.
			req2, err := http.NewRequest("POST", "/v1/webhooks/native", bytes.NewBufferString(payload))
			if err != nil {
				t.Fatalf("failed to create request: %s", err)
			}
			req2.Header.Set("Authorization", "Bearer "+bearerToken)
			rec2 := httptest.NewRecorder()
			srv.router.ServeHTTP(rec2, req2)
			if rec2.Code != http.StatusUnauthorized {
				t.Fatalf("bearer auth: expected 401, got %d", rec2.Code)
			}

			// The requests were rejected before any event was submitted.
			if len(fp.submitted) != 0 {
				t.Fatalf("expected no submitted events, got %d", len(fp.submitted))
			}

			captured := out()
			for _, secret := range []string{basicPassword, bearerToken} {
				if strings.Contains(captured, secret) {
					t.Errorf("captured log output contains secret %q:\n%s", secret, captured)
				}
			}
		})
	}
}
