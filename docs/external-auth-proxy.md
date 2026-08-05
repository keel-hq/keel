# Admin UI behind an external authentication proxy

Keel can delegate the authentication boundary for its Admin UI and Admin API to
an external reverse proxy such as oauth2-proxy. This is explicitly opt-in:
`AUTH_MODE=external-proxy` (or `auth.mode: external-proxy` in the Helm chart).
The default `legacy` mode retains the existing Basic Auth/JWT behavior.

## Security boundary

In external-proxy mode Keel does not validate an OAuth token or create a local
session. It trusts that the proxy authenticated every request before forwarding
it. To make that trust boundary enforceable, Keel listens only on
`127.0.0.1:9300`. The supported Helm topology runs oauth2-proxy in the same Pod
and targets only its port from the Service and Ingress:

```
browser -> Ingress -> Service:4180 -> oauth2-proxy -> 127.0.0.1:9300 -> Keel
```

Do not expose port 9300 with a second Service, `hostPort`, `hostNetwork`, or
another sidecar. An identity header is attribution data, not authentication;
any client that can reach the Keel listener could forge it. Keel rejects Admin
API requests without the configured header, but loopback-only reachability is
the control that prevents spoofing.

The proxy user header defaults to `X-Forwarded-User`. Keel uses it for the user
shown in the UI and, in this mode only, overrides client-supplied approval voter
names with the authenticated identity. Keep oauth2-proxy configured to replace
forwarded headers (`--pass-user-headers=true`), and do not configure it to
preserve a client-supplied identity header.

## Helm example with oauth2-proxy

Create a Secret using oauth2-proxy's environment variable names. The cookie
secret value delivered to the container must be 16, 24, or 32 bytes (a random
32-character value is suitable when using `stringData`).

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: keel-oauth2-proxy
  namespace: keel
type: Opaque
stringData:
  OAUTH2_PROXY_CLIENT_ID: keel
  OAUTH2_PROXY_CLIENT_SECRET: replace-with-provider-client-secret
  OAUTH2_PROXY_COOKIE_SECRET: replace-with-a-random-32-character-value
```

Then install with values like these (replace the issuer and public URL):

```yaml
auth:
  mode: external-proxy
  proxyUserHeader: X-Forwarded-User
  proxyLogoutURL: /oauth2/sign_out?rd=/

basicauth:
  enabled: false

oauth2Proxy:
  enabled: true
  existingSecret: keel-oauth2-proxy
  extraArgs:
    - --provider=oidc
    - --oidc-issuer-url=https://identity.example.com
    - --redirect-url=https://keel.example.com/oauth2/callback
    - --email-domain=example.com
    - --scope=openid profile email
    - --prefer-email-to-user=true
    - --cookie-secure=true
    - --code-challenge-method=S256

service:
  enabled: true
  type: ClusterIP

ingress:
  enabled: true
  hosts:
    - host: keel.example.com
      paths:
        - /
  tls:
    - secretName: keel-tls
      hosts: [keel.example.com]
```

The chart pins oauth2-proxy v7.8.1 by its multi-architecture image digest. It
fails template rendering if external-proxy mode is combined with Basic Auth,
has no oauth2-proxy sidecar/Secret, or has no Service. The Service target changes
from Keel's port 9300 to oauth2-proxy's port 4180.

The Ingress must route `/oauth2/*`, the SPA, static assets, and `/v1/*` to the
same Service. Keel has no websocket routes. oauth2-proxy protects all paths by
default, including webhook endpoints; if provider webhooks must remain public,
review each `--skip-auth-route` exception separately because it changes the
external boundary.

## Runtime configuration and validation

| Variable | Meaning |
| --- | --- |
| `AUTH_MODE` | `legacy` (default), `basic`, or `external-proxy` |
| `AUTH_PROXY_USER_HEADER` | Stable user header; external-proxy only; default `X-Forwarded-User` |
| `AUTH_PROXY_LOGOUT_URL` | Same-origin proxy sign-out path; external-proxy only; default `/oauth2/sign_out?rd=/` |

`AUTH_MODE=basic` requires both Basic Auth variables. External-proxy mode rejects
either Basic Auth variable, invalid header names, and absolute/cross-origin
logout URLs. Proxy-specific variables in other modes also fail startup. Startup
logs record the selected mode, loopback address, and trusted identity header.

The UI probes the current-user API on startup without manufacturing Keel
credentials. Thus an authenticated proxy session enters the application without
the Keel login screen, while legacy mode still falls back to the existing local
login after a 401. Logout clears the oauth2-proxy cookie through the configured
path. Whether that also terminates the identity provider's SSO session depends
on provider configuration; returning to Keel may immediately authenticate again.

## Deterministic local verification

`make e2e` extends the existing isolated k3s harness with oauth2-proxy v7.8.1
and Dex v2.41.1, both pinned by digest. It exercises redirect, password login
against an in-cluster test identity, Admin UI and `/v1/resources`, the forwarded
audit identity, direct-listener isolation, spoofed unauthenticated requests, and
logout. The harness uses no external OAuth account or repository secret and
collects all three components' logs and cluster events under `.test/artifacts/`.

```bash
make e2e
```

This command creates its own k3s process and namespaces, uses bounded readiness
polling, and removes the cluster/resources on exit. Its preflight intentionally
refuses to run over an existing local k3s installation.
