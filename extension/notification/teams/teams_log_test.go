package teams

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

// TestConfigureDoesNotLogWebhookSecret proves that a Teams webhook signing
// key embedded in the URL (the <guid>@<tenant> path segment) never appears
// in log output, neither at info level nor in the debug-only endpoint detail.
func TestConfigureDoesNotLogWebhookSecret(t *testing.T) {
	key := "2c1a3b4c5d6e7f8091a2b3c4d5e6f708"
	tenant := "contoso.example"
	hookID := "1a2b3c4d5e6f708192a3b4c5d6e7f809"
	endpoint := "https://outlook.office.com/webhook/" + key + "@" + tenant + "/IncomingWebhook/" + hookID + "/webhook"

	for _, level := range []log.Level{log.InfoLevel, log.DebugLevel} {
		t.Run(level.String(), func(t *testing.T) {
			out := captureLogs(t, level)

			s := &sender{}
			enabled, err := s.Configure(&notification.Config{
				Notifications: config.NotificationConfig{
					Teams: config.TeamsConfig{WebhookURL: endpoint},
				},
			})
			if err != nil || !enabled {
				t.Fatalf("Configure = enabled %v, err %v", enabled, err)
			}

			captured := out()
			if !strings.Contains(captured, "outlook.office.com") {
				t.Fatalf("expected the endpoint host to be logged, got: %s", captured)
			}
			for _, secret := range []string{key, tenant, hookID} {
				if strings.Contains(captured, secret) {
					t.Errorf("captured log output contains secret %q:\n%s", secret, captured)
				}
			}
		})
	}
}
