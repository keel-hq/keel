package notification

import (
	"net/url"
	"strings"
)

// Redacted is substituted for URL components that may carry a secret
// (webhook signing keys, bot tokens, passwords) before a URL is logged.
const Redacted = "[REDACTED]"

// SafeURL returns a version of rawURL that is safe to log at any level.
// Webhook endpoint URLs commonly embed signing secrets in the path or query
// string (e.g. Teams, Discord and Mattermost incoming-webhook URLs), so only
// the scheme and host are retained.
func SafeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Redacted
	}
	return parsed.Scheme + "://" + parsed.Host
}

// DebugURL returns a version of rawURL that may be logged at debug level
// only. The path structure is preserved for diagnostics, while userinfo,
// any path segment that looks like a secret, and every query value are
// redacted. Callers must ensure the result is only logged when debug
// logging is enabled (see log.IsLevelEnabled(log.DebugLevel)).
func DebugURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return Redacted
	}

	var b strings.Builder
	b.WriteString(parsed.Scheme)
	b.WriteString("://")
	if parsed.Host != "" {
		b.WriteString(parsed.Host)
	}
	// parsed.User is intentionally ignored: userinfo may carry credentials.

	for _, seg := range strings.Split(strings.Trim(parsed.Path, "/"), "/") {
		if seg == "" {
			continue
		}
		b.WriteString("/")
		b.WriteString(sanitizePathSegment(seg))
	}

	if parsed.RawQuery != "" {
		b.WriteString("?")
		first := true
		for _, pair := range strings.Split(parsed.RawQuery, "&") {
			if !first {
				b.WriteString("&")
			}
			first = false
			key, _, _ := strings.Cut(pair, "=")
			if key == "" {
				b.WriteString(Redacted)
			} else {
				b.WriteString(key)
				b.WriteString("=")
				b.WriteString(Redacted)
			}
		}
	}

	return b.String()
}

// sanitizePathSegment redacts path segments that look like secrets: anything
// long enough to be a random token (>= 16 characters, which also covers
// UUIDs/GUIDs) or containing '@' (e.g. Teams webhook keys of the form
// "<guid>@<tenant>"). Short structural segments such as "webhook",
// "IncomingWebhook" or "hooks" are preserved so the debug output still
// identifies the endpoint.
func sanitizePathSegment(seg string) string {
	if len(seg) >= 16 || strings.ContainsRune(seg, '@') {
		return Redacted
	}
	return seg
}
