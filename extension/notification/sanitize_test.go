package notification

import (
	"strings"
	"testing"
)

// TestSafeURLStripsSecrets verifies that SafeURL keeps only the scheme and
// host, so webhook signing secrets in the path, query string or userinfo can
// never reach log output.
func TestSafeURLStripsSecrets(t *testing.T) {
	const (
		teamsKey   = "2c1a3b4c5d6e7f8091a2b3c4d5e6f708"
		discordTok = "9f8e7d6c5b4a39281706f5e4d3c2b1a0"
		mmToken    = "mm-hook-token-abc123xyz789"
		queryTok   = "query-secret-0a1b2c3d4e5f"
		smtpPass   = "smtp-password-9z8y7x"
	)

	tests := []struct {
		name    string
		rawURL  string
		want    string
		secrets []string
	}{
		{
			name:   "teams webhook key in path",
			rawURL: "https://outlook.office.com/webhook/" + teamsKey + "@contoso.com/IncomingWebhook/1a2b3c4d5e6f708192a3b4c5d6e7f809/webhook",
			want:   "https://outlook.office.com",
			secrets: []string{
				teamsKey,
				"1a2b3c4d5e6f708192a3b4c5d6e7f809",
			},
		},
		{
			name:    "discord token in path",
			rawURL:  "https://discord.com/api/webhooks/1038689483310731242/" + discordTok,
			want:    "https://discord.com",
			secrets: []string{discordTok},
		},
		{
			name:    "mattermost token in path",
			rawURL:  "https://mattermost.example.com/hooks/" + mmToken,
			want:    "https://mattermost.example.com",
			secrets: []string{mmToken},
		},
		{
			name:    "token in query string",
			rawURL:  "https://hooks.example.com/notify?token=" + queryTok,
			want:    "https://hooks.example.com",
			secrets: []string{queryTok},
		},
		{
			name:    "credentials in userinfo",
			rawURL:  "https://bot:" + smtpPass + "@hooks.example.com/hook",
			want:    "https://hooks.example.com",
			secrets: []string{smtpPass, "bot"},
		},
		{
			name:    "unparsable input",
			rawURL:  "://not-a-url",
			want:    Redacted,
			secrets: []string{"not-a-url"},
		},
		{
			name:    "missing scheme",
			rawURL:  "hooks.example.com/hook",
			want:    Redacted,
			secrets: []string{"hook"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeURL(tt.rawURL)
			if got != tt.want {
				t.Errorf("SafeURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
			for _, secret := range tt.secrets {
				if strings.Contains(got, secret) {
					t.Errorf("SafeURL(%q) = %q leaks secret %q", tt.rawURL, got, secret)
				}
			}
		})
	}
}

// TestDebugURLRedactsSecretsButKeepsStructure verifies that DebugURL keeps
// the endpoint identifiable (host and structural path segments) while the
// secret parts are still redacted.
func TestDebugURLRedactsSecretsButKeepsStructure(t *testing.T) {
	const (
		teamsKey  = "2c1a3b4c5d6e7f8091a2b3c4d5e6f708"
		discordID = "1038689483310731242"
		discordTk = "9f8e7d6c5b4a39281706f5e4d3c2b1a0"
		queryTok  = "query-secret-0a1b2c3d4e5f"
		smtpUser  = "botuser"
		smtpPass  = "smtp-password-9z8y7x"
	)

	tests := []struct {
		name    string
		rawURL  string
		want    string
		secrets []string
		keep    []string
	}{
		{
			name:   "teams webhook keeps path structure",
			rawURL: "https://outlook.office.com/webhook/" + teamsKey + "@contoso.com/IncomingWebhook/1a2b3c4d5e6f708192a3b4c5d6e7f809/webhook",
			want:   "https://outlook.office.com/webhook/" + Redacted + "/IncomingWebhook/" + Redacted + "/webhook",
			secrets: []string{
				teamsKey,
				"contoso.com",
				"1a2b3c4d5e6f708192a3b4c5d6e7f809",
			},
			keep: []string{"outlook.office.com", "webhook", "IncomingWebhook"},
		},
		{
			name:    "discord keeps path structure",
			rawURL:  "https://discord.com/api/webhooks/" + discordID + "/" + discordTk,
			want:    "https://discord.com/api/webhooks/" + Redacted + "/" + Redacted,
			secrets: []string{discordTk},
			keep:    []string{"discord.com", "api", "webhooks"},
		},
		{
			name:    "query value redacted, key kept",
			rawURL:  "https://hooks.example.com/notify?token=" + queryTok,
			want:    "https://hooks.example.com/notify?token=" + Redacted,
			secrets: []string{queryTok},
			keep:    []string{"hooks.example.com", "notify", "token="},
		},
		{
			name:    "userinfo redacted",
			rawURL:  "https://" + smtpUser + ":" + smtpPass + "@hooks.example.com/hook",
			want:    "https://hooks.example.com/hook",
			secrets: []string{smtpUser, smtpPass},
			keep:    []string{"hooks.example.com", "hook"},
		},
		{
			name:    "unparsable input",
			rawURL:  "://not-a-url",
			want:    Redacted,
			secrets: []string{"not-a-url"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DebugURL(tt.rawURL)
			if got != tt.want {
				t.Errorf("DebugURL(%q) = %q, want %q", tt.rawURL, got, tt.want)
			}
			for _, secret := range tt.secrets {
				if strings.Contains(got, secret) {
					t.Errorf("DebugURL(%q) = %q leaks secret %q", tt.rawURL, got, secret)
				}
			}
			for _, k := range tt.keep {
				if !strings.Contains(got, k) {
					t.Errorf("DebugURL(%q) = %q lost diagnostic part %q", tt.rawURL, got, k)
				}
			}
		})
	}
}
