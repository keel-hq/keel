# Shoutrrr notifications

Keel can deliver notifications through
[Shoutrrr](https://github.com/nicholas-fedor/shoutrrr), which reaches a large
number of services using a single URL format. This is the way to notify a
service that Keel has no dedicated sender for.

Supported services include ntfy, Gotify, Telegram, Matrix, Pushover, Pushbullet,
Bark, Join, OpsGenie, PagerDuty, Signal, Twilio, Rocket.Chat, Zulip, Lark,
WeCom, Notifiarr, Google Chat, MQTT, IFTTT, SMTP and a `generic` webhook, as
well as Slack, Discord, Teams and Mattermost. See the
[shoutrrr services documentation](https://shoutrrr.nickfedor.com/latest/services/overview/)
for the URL format of each one.

Keel's dedicated Slack, Discord, Teams, Mattermost and mail senders are
unaffected and remain the better choice for those services, because they build
service-specific payloads (Discord embeds, for example) that Shoutrrr's generic
message format cannot express.

## Configuration

| Variable | Required | Default | Description |
| --- | --- | --- | --- |
| `SHOUTRRR_URLS` | yes | | Whitespace separated service URLs. Enables the sender. |
| `SHOUTRRR_TIMEOUT` | no | `10s` | Per-service send timeout, as a [Go duration](https://pkg.go.dev/time#ParseDuration). |

```bash
SHOUTRRR_URLS="ntfy://ntfy.sh/my-keel-topic gotify://gotify.example.com/AzyoeNS.D-A_yolo"
```

URLs are separated by **whitespace or newlines, not commas**. Shoutrrr encodes
list properties inside the URL query using commas, so a comma is part of a URL
rather than a separator between two of them:

```bash
SHOUTRRR_URLS="telegram://token@telegram?chats=111,222"
```

### Helm

The chart stores the URLs in the Keel secret, because they carry credentials.

```yaml
shoutrrr:
  enabled: true
  urls:
    - "ntfy://ntfy.sh/my-keel-topic"
    - "telegram://token@telegram?chats=111,222"
  # optional, defaults to 10s
  timeout: "30s"
```

`shoutrrr.urls` also accepts a plain pre-formatted string if you would rather
supply the list from an existing value.

## Behaviour

**Credentials are kept out of the logs.** Service URLs embed bot tokens, API
keys and SMTP passwords. Keel never logs a URL; it logs a redacted identifier of
the form `scheme://host/***`. Shoutrrr's own logger, which prints raw URLs, is
discarded. Send failures are reported by service scheme, for example `gotify`.

**Each URL is initialised independently.** If one URL is malformed or names an
unknown service, Keel logs it (redacted), ignores it, and keeps using the rest.
The sender is only disabled when no URL at all is usable, which is reported as a
configuration error.

**A notification is considered delivered once any service accepts it.** Keel's
notifier retries a failed sender with a backoff, which would re-deliver the
message to every service that already received it. To avoid that duplication,
Keel reports failure only when every service failed. Individual failures are
always logged.

**All notifications go to every configured URL.** Shoutrrr does not take part in
the per-deployment channel overrides (`keel.sh/notify`) that Slack and Hipchat
support. Use `NOTIFICATION_LEVEL` to control the volume.

## Message format

The notification type and name become the title, and the event level is passed
as the shoutrrr message level. Keel's `success` level has no shoutrrr
equivalent and is sent as `Info`; `fatal` is sent as `Error`.

The body carries the message followed by the event details, with metadata keys
sorted so that identical events render identically:

```
Deployment default/whatever updated

Resource: deployment
Identifier: default/whatever
Level: success
```

Details are rendered into the message body rather than passed as structured
fields, because only two shoutrrr services implement its rich-sender interface
and every other service silently discards fields.
