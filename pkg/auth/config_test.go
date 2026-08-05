package auth

import (
	"testing"

	"github.com/keel-hq/keel/constants"
)

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name        string
		env         map[string]string
		wantMode    Mode
		wantEnabled bool
		wantHeader  string
		wantErr     bool
	}{
		{name: "empty environment preserves disabled legacy behavior", wantMode: ModeLegacy},
		{name: "legacy credentials preserve basic auth", env: map[string]string{constants.EnvBasicAuthUser: "admin", constants.EnvBasicAuthPassword: "secret"}, wantMode: ModeLegacy, wantEnabled: true},
		{name: "explicit basic", env: map[string]string{constants.EnvAuthMode: "basic", constants.EnvBasicAuthUser: "admin", constants.EnvBasicAuthPassword: "secret"}, wantMode: ModeBasic, wantEnabled: true},
		{name: "external proxy has safe defaults", env: map[string]string{constants.EnvAuthMode: "external-proxy"}, wantMode: ModeExternalProxy, wantEnabled: true, wantHeader: DefaultProxyUserHeader},
		{name: "external proxy custom canonical header", env: map[string]string{constants.EnvAuthMode: "external-proxy", constants.EnvAuthProxyUserHeader: "x-auth-request-user"}, wantMode: ModeExternalProxy, wantEnabled: true, wantHeader: "X-Auth-Request-User"},
		{name: "unknown mode", env: map[string]string{constants.EnvAuthMode: "oidc"}, wantErr: true},
		{name: "partial basic credentials", env: map[string]string{constants.EnvBasicAuthUser: "admin"}, wantErr: true},
		{name: "basic without credentials", env: map[string]string{constants.EnvAuthMode: "basic"}, wantErr: true},
		{name: "proxy conflicts with basic", env: map[string]string{constants.EnvAuthMode: "external-proxy", constants.EnvBasicAuthUser: "admin", constants.EnvBasicAuthPassword: "secret"}, wantErr: true},
		{name: "proxy options in legacy", env: map[string]string{constants.EnvAuthProxyUserHeader: "X-User"}, wantErr: true},
		{name: "invalid proxy header", env: map[string]string{constants.EnvAuthMode: "external-proxy", constants.EnvAuthProxyUserHeader: "Bad Header"}, wantErr: true},
		{name: "external logout URL", env: map[string]string{constants.EnvAuthMode: "external-proxy", constants.EnvAuthProxyLogoutURL: "/oauth2/sign_out?rd=/goodbye"}, wantMode: ModeExternalProxy, wantEnabled: true, wantHeader: DefaultProxyUserHeader},
		{name: "external absolute logout URL rejected", env: map[string]string{constants.EnvAuthMode: "external-proxy", constants.EnvAuthProxyLogoutURL: "https://example.com/logout"}, wantErr: true},
		{name: "protocol relative logout URL rejected", env: map[string]string{constants.EnvAuthMode: "external-proxy", constants.EnvAuthProxyLogoutURL: "//example.com/logout"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ConfigFromEnv(func(key string) string { return tt.env[key] })
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected configuration error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected configuration error: %v", err)
			}
			if cfg.Mode != tt.wantMode || cfg.AdminEnabled() != tt.wantEnabled || cfg.ProxyUserHeader != tt.wantHeader {
				t.Fatalf("config = %#v, want mode=%q enabled=%t header=%q", cfg, tt.wantMode, tt.wantEnabled, tt.wantHeader)
			}
		})
	}
}
