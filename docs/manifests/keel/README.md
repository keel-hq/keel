# Keel static Kubernetes manifests
#
# This directory provides a ready-to-install set of Kubernetes manifests for
# deploying Keel without Helm. These manifests are intended as an alternative to
# the maintained Helm chart (see chart/keel).
#
# WARNING: The Helm chart is the recommended way to install Keel because it
# supports upgrades, configuration via values, and keeps the deployment in sync
# with future releases. Use these static manifests only if you cannot use Helm
# or need a quick self-hosted install.
#
# NOTE: The sunstone.dev installation helper previously referenced in some
# guides is no longer available. Use either the Helm chart (preferred) or the
# manifests in this directory as a self-hosted replacement.

## Usage

1. Edit the Secret (`secret.yaml`) to set your basic-auth credentials, or set
   `basicauth.enabled` to `false` in the values and remove the secret.
2. Apply the manifests in order:

   ```bash
   kubectl apply -f docs/manifests/keel/namespace.yaml
   kubectl apply -f docs/manifests/keel/service-account.yaml
   kubectl apply -f docs/manifests/keel/clusterrole.yaml
   kubectl apply -f docs/manifests/keel/clusterrolebinding.yaml
   kubectl apply -f docs/manifests/keel/secret.yaml
   kubectl apply -f docs/manifests/keel/deployment.yaml
   kubectl apply -f docs/manifests/keel/service.yaml
   ```

   Or apply everything at once:

   ```bash
   kubectl apply -R -f docs/manifests/keel/
   ```

## Placeholders / Notes

- `IMAGE` and `IMAGE_TAG` — set these to the Keel container image and version
  you want to run. The latest release is published under the `keelhq/keel`
  Docker Hub repository and `ghcr.io/keel-hq/keel` (GHCR). A nightly build from
  `master` is available under the `nightly` tag.
- `BASIC_AUTH_USER` / `BASIC_AUTH_PASSWORD` — credentials for the Keel Admin UI
  and API. When `basicauth.enabled` is `true`, a Secret named `keel` must
  contain a `BASIC_AUTH_PASSWORD` key holding the base64-encoded password.
  Update the Secret, or set `basicauth.enabled: false` and remove the Secret
  and `envFrom` reference if you do not want basic auth.

## Prerequisites

- [Helm](https://docs.helm.sh/using_helm/#installing-helm) is not required for
  this install method, but is recommended.
- Kubernetes cluster access and `kubectl` configured.

See the main [README](../../readme.md) for the recommended Helm install
instructions and configuration reference.
