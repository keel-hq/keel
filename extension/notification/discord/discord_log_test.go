package discord

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

// TestConfigureDoesNotLogWebhookSecret proves that a Discord webhook token
// embedded in the URL never appears in log output, neither at info level nor
// in the debug-only endpoint detail.
func TestConfigureDoesNotLogWebhookSecret(t *testing.T) {
	token := "9f8e7d6c5b4a39281706f5e4d3c2b1a0"
	endpoint := "https://discord.com/api/webhooks/1038689483310731242/" + token

	for _, level := range []log.Level{log.InfoLevel, log.DebugLevel} {
		t.Run(level.String(), func(t *testing.T) {
			out := captureLogs(t, level)

			s := &sender{}
			enabled, err := s.Configure(&notification.Config{
				Notifications: config.NotificationConfig{
					Discord: config.DiscordConfig{WebhookURL: endpoint},
				},
			})
			if err != nil || !enabled {
				t.Fatalf("Configure = enabled %v, err %v", enabled, err)
			}

			captured := out()
			if !strings.Contains(captured, "discord.com") {
				t.Fatalf("expected the endpoint host to be logged, got: %s", captured)
			}
			if strings.Contains(captured, token) {
				t.Errorf("captured log output contains endpoint secret:\n%s", captured)
			}
		})
	}
}

// TestConfigureInvalidEndpointRejected proves that a malformed Discord
// webhook URL is rejected with an error instead of silently accepted with an
// unusable endpoint, and that the raw endpoint leaks neither into the
// returned error nor into log output.
func TestConfigureInvalidEndpointRejected(t *testing.T) {
	bad := "discord.com/api/webhooks/1038689483310731242/9f8e7d6c5b4a39281706f5e4d3c2b1a0"

	out := captureLogs(t, log.DebugLevel)

	s := &sender{}
	enabled, err := s.Configure(&notification.Config{
		Notifications: config.NotificationConfig{
			Discord: config.DiscordConfig{WebhookURL: bad},
		},
	})
	if enabled {
		t.Fatalf("Configure = enabled %v, want false", enabled)
	}
	if err == nil {
		t.Fatal("Configure = nil error, want an error for an unparsable webhook URL")
	}
	if strings.Contains(err.Error(), bad) {
		t.Errorf("returned error echoes the raw endpoint: %s", err)
	}
	if captured := out(); strings.Contains(captured, bad) {
		t.Errorf("captured log output contains the raw endpoint:\n%s", captured)
	}
}
