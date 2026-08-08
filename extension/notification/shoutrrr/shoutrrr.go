package shoutrrr

import (
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"

	"github.com/nicholas-fedor/shoutrrr/pkg/router"
	shoutrrrtypes "github.com/nicholas-fedor/shoutrrr/pkg/types"

	log "github.com/sirupsen/logrus"
)

const (
	senderName = "shoutrrr"

	// defaultTimeout is applied per service, per notification.
	defaultTimeout = 10 * time.Second

	// redacted is substituted for any part of a service URL that may carry a secret.
	redacted = "***"
)

type sender struct {
	router *router.ServiceRouter

	// services holds a redacted identifier per configured URL, in the order they
	// were accepted. Used for logging only - raw URLs must never be logged.
	services []string
}

func init() {
	notification.RegisterSender(senderName, &sender{})
}

func (s *sender) Configure(config *notification.Config) (bool, error) {
	urls := splitURLs(config.Application.Notifications.Shoutrrr.URLs)
	if len(urls) == 0 {
		return false, nil
	}

	timeout, err := parseTimeout(config.Application.Notifications.Shoutrrr.Timeout)
	if err != nil {
		return false, err
	}

	// Shoutrrr writes raw service URLs to this logger (see ServiceRouter.initService,
	// which logs custom URLs verbatim). Those URLs embed bot tokens, API keys and SMTP
	// passwords, so the logger is deliberately discarded and Keel emits its own
	// redacted diagnostics instead.
	r, err := router.NewWithOptions(
		stdlog.New(io.Discard, "", 0),
		shoutrrrtypes.SenderOptions{Timeout: timeout},
	)
	if err != nil {
		return false, fmt.Errorf("could not create shoutrrr router: %s", err)
	}

	var accepted []string
	for i, u := range urls {
		// Errors from AddService can embed the raw URL, so they are never logged verbatim.
		if err := r.AddService(u); err != nil {
			log.WithFields(log.Fields{
				"name":    senderName,
				"service": redact(u),
				"index":   i,
			}).Error("extension.notification.shoutrrr: ignoring service URL that could not be initialised")

			continue
		}

		accepted = append(accepted, redact(u))
	}

	if len(accepted) == 0 {
		return false, fmt.Errorf("no usable service URLs in %s", "SHOUTRRR_URLS")
	}

	s.router = r
	s.services = accepted

	log.WithFields(log.Fields{
		"name":     senderName,
		"services": strings.Join(accepted, ", "),
		"timeout":  timeout,
	}).Info("extension.notification.shoutrrr: sender configured")

	return true, nil
}

func (s *sender) Send(event types.EventNotification) error {
	params := shoutrrrtypes.Params{}
	params.SetTitle(title(event))
	params.SetLevel(messageLevel(event.Level))

	results := s.router.Send(body(event), &params)

	var failures []error

	for _, err := range results {
		if err != nil {
			failures = append(failures, err)
		}
	}

	if len(failures) == 0 {
		return nil
	}

	// Failures are wrapped in shoutrrr's *types.TargetError, which identifies the
	// target by service scheme (Service.GetID) rather than by URL, so they carry
	// no credentials and are safe to log.
	for _, err := range failures {
		log.WithError(err).WithFields(log.Fields{
			"name":              senderName,
			"notification name": event.Name,
		}).Error("extension.notification.shoutrrr: could not send notification to service")
	}

	// Keel's notifier retries the whole sender with a backoff when Send returns an
	// error, which would re-deliver to every service that already succeeded. Only
	// report failure when every service failed, so that one broken target cannot
	// turn into duplicate messages everywhere else.
	if len(failures) == len(results) {
		return fmt.Errorf("failed to send notification to all %d shoutrrr service(s): %w",
			len(results), errors.Join(failures...))
	}

	return nil
}

// splitURLs splits the configured service URL list on whitespace.
//
// Comma is deliberately not a separator: shoutrrr encodes list properties inside
// the URL query using commas (for example telegram://token@telegram?chats=1,2), so
// splitting on it would corrupt otherwise valid URLs.
func splitURLs(raw string) []string {
	return strings.Fields(raw)
}

func parseTimeout(raw string) (time.Duration, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultTimeout, nil
	}

	timeout, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %s", "SHOUTRRR_TIMEOUT", raw, err)
	}

	if timeout <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration, got %q", "SHOUTRRR_TIMEOUT", raw)
	}

	return timeout, nil
}

// redact reduces a service URL to its scheme and host so that it can be written to
// logs. Shoutrrr carries credentials in the userinfo, path and query of a URL
// (discord://token@channel, smtp://user:pass@host, generic://host/hook?token=x), so
// everything except the destination host is dropped.
func redact(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return "<unparsable service url>"
	}

	if parsed.Hostname() == "" {
		return parsed.Scheme + "://" + redacted
	}

	return parsed.Scheme + "://" + parsed.Hostname() + "/" + redacted
}

// messageLevel maps a Keel event level onto the levels shoutrrr understands.
// Shoutrrr has no separate success or fatal level, so those fold into the
// nearest equivalent.
func messageLevel(level types.Level) shoutrrrtypes.MessageLevel {
	switch level {
	case types.LevelDebug:
		return shoutrrrtypes.Debug
	case types.LevelInfo, types.LevelSuccess:
		return shoutrrrtypes.Info
	case types.LevelWarn:
		return shoutrrrtypes.Warning
	case types.LevelError, types.LevelFatal:
		return shoutrrrtypes.Error
	default:
		return shoutrrrtypes.Unknown
	}
}

func title(event types.EventNotification) string {
	return fmt.Sprintf("%s: %s", event.Type.String(), event.Name)
}

// body renders the notification as plain text.
//
// Only discord and bark implement shoutrrr's RichSender interface; every other
// service falls back to plain text and discards structured fields, so all of the
// detail is rendered into the message itself rather than passed as fields.
func body(event types.EventNotification) string {
	sections := []string{event.Message}

	var details []string

	if event.ResourceKind != "" {
		details = append(details, "Resource: "+event.ResourceKind)
	}

	if event.Identifier != "" {
		details = append(details, "Identifier: "+event.Identifier)
	}

	details = append(details, "Level: "+event.Level.String())

	keys := make([]string, 0, len(event.Metadata))
	for k := range event.Metadata {
		keys = append(keys, k)
	}

	// Sorted so that repeated notifications render identically.
	sort.Strings(keys)

	for _, k := range keys {
		details = append(details, k+": "+event.Metadata[k])
	}

	return strings.Join(append(sections, strings.Join(details, "\n")), "\n\n")
}
