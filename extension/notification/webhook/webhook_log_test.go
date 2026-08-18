package webhook

import (
	"strings"
	"sync"
	"testing"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/pkg/config"

	log "github.com/sirupsen/logrus"
)

type logCapture struct {
	mu    sync.Mutex
	lines []string
}

func (l *logCapture) Fire(entry *log.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, line)
	return nil
}

func (l *logCapture) Levels() []log.Level {
	return log.AllLevels
}

func (l *logCapture) Output() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

// captureLogs attaches a hook to the global logrus logger at the given level
// and returns a function that reads everything captured so far. The previous
// level and hooks are restored when the test finishes.
func captureLogs(t *testing.T, level log.Level) func() string {
	t.Helper()
	std := log.StandardLogger()
	prevLevel := std.GetLevel()
	prevHooks := cloneHooks(std.Hooks)
	std.SetLevel(level)
	capture := &logCapture{}
	std.AddHook(capture)
	t.Cleanup(func() {
		std.SetLevel(prevLevel)
		std.Hooks = prevHooks
	})
	return capture.Output
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

// TestConfigureDoesNotLogEndpointSecret proves that a webhook signing secret
// embedded in the configured endpoint (path or query) never appears in log
// output, neither at info level nor in the debug-only endpoint detail.
func TestConfigureDoesNotLogEndpointSecret(t *testing.T) {
	secret := "super-secret-webhook-token-9f86d0"
	endpoint := "https://hooks.example.com/hooks/" + secret + "?api_key=" + secret

	for _, level := range []log.Level{log.InfoLevel, log.DebugLevel} {
		t.Run(level.String(), func(t *testing.T) {
			out := captureLogs(t, level)

			s := &sender{}
			enabled, err := s.Configure(&notification.Config{
				Notifications: config.NotificationConfig{
					Webhook: config.WebhookConfig{Endpoint: endpoint},
				},
			})
			if err != nil || !enabled {
				t.Fatalf("Configure = enabled %v, err %v", enabled, err)
			}

			captured := out()
			if !strings.Contains(captured, "hooks.example.com") {
				t.Fatalf("expected the endpoint host to be logged, got: %s", captured)
			}
			if strings.Contains(captured, secret) {
				t.Errorf("captured log output contains endpoint secret:\n%s", captured)
			}
		})
	}
}
