package auth

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	appconfig "github.com/keel-hq/keel/pkg/config"
)

type Mode string

const (
	ModeLegacy        Mode = "legacy"
	ModeBasic         Mode = "basic"
	ModeExternalProxy Mode = "external-proxy"
)

const (
	DefaultProxyUserHeader = "X-Forwarded-User"
	DefaultProxyLogoutURL  = "/oauth2/sign_out?rd=/"
)

// Config is the validated administrator authentication configuration.
type Config struct {
	Mode            Mode
	Username        string
	Password        string
	ProxyUserHeader string
	ProxyLogoutURL  string
}

// FromConfig maps application authentication settings to the validated runtime form.
func FromConfig(raw appconfig.AuthConfig) (Config, error) {
	modeValue := strings.TrimSpace(raw.Mode)
	if modeValue == "" {
		modeValue = string(ModeLegacy)
	}

	cfg := Config{
		Mode:            Mode(strings.ToLower(modeValue)),
		Username:        raw.BasicUser,
		Password:        raw.BasicPassword,
		ProxyUserHeader: strings.TrimSpace(raw.ProxyUserHeader),
		ProxyLogoutURL:  strings.TrimSpace(raw.ProxyLogoutURL),
	}

	if (cfg.Username == "") != (cfg.Password == "") {
		return Config{}, fmt.Errorf("BASIC_AUTH_USER and BASIC_AUTH_PASSWORD must either both be set or both be unset")
	}

	switch cfg.Mode {
	case ModeLegacy:
		if cfg.ProxyUserHeader != "" || cfg.ProxyLogoutURL != "" {
			return Config{}, fmt.Errorf("AUTH_PROXY_USER_HEADER and AUTH_PROXY_LOGOUT_URL require AUTH_MODE=external-proxy")
		}
	case ModeBasic:
		if cfg.Username == "" {
			return Config{}, fmt.Errorf("AUTH_MODE=basic requires BASIC_AUTH_USER and BASIC_AUTH_PASSWORD")
		}
		if cfg.ProxyUserHeader != "" || cfg.ProxyLogoutURL != "" {
			return Config{}, fmt.Errorf("AUTH_PROXY_USER_HEADER and AUTH_PROXY_LOGOUT_URL require AUTH_MODE=external-proxy")
		}
	case ModeExternalProxy:
		if cfg.Username != "" {
			return Config{}, fmt.Errorf("AUTH_MODE=external-proxy conflicts with BASIC_AUTH_USER and BASIC_AUTH_PASSWORD; remove the Basic Auth credentials")
		}
		if cfg.ProxyUserHeader == "" {
			cfg.ProxyUserHeader = DefaultProxyUserHeader
		}
		if !validHeaderName(cfg.ProxyUserHeader) {
			return Config{}, fmt.Errorf("AUTH_PROXY_USER_HEADER %q is not a valid HTTP header name", cfg.ProxyUserHeader)
		}
		cfg.ProxyUserHeader = http.CanonicalHeaderKey(cfg.ProxyUserHeader)
		if cfg.ProxyLogoutURL == "" {
			cfg.ProxyLogoutURL = DefaultProxyLogoutURL
		}
		parsed, err := url.Parse(cfg.ProxyLogoutURL)
		if err != nil || parsed.IsAbs() || !strings.HasPrefix(cfg.ProxyLogoutURL, "/") || strings.HasPrefix(cfg.ProxyLogoutURL, "//") {
			return Config{}, fmt.Errorf("AUTH_PROXY_LOGOUT_URL must be a same-origin absolute path beginning with /, got %q", cfg.ProxyLogoutURL)
		}
	default:
		return Config{}, fmt.Errorf("AUTH_MODE must be one of legacy, basic, or external-proxy, got %q", modeValue)
	}

	return cfg, nil
}

func (c Config) AdminEnabled() bool {
	return c.Mode == ModeExternalProxy || c.Username != ""
}

func validHeaderName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && !strings.ContainsRune("!#$%&'*+-.^_`|~", r) {
			return false
		}
	}
	return true
}
