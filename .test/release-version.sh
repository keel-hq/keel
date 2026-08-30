#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
RESOLVER="${REPO_ROOT}/scripts/resolve-application-version.sh"
chart_version="$(awk '$1 == "appVersion:" {print $2; exit}' "${REPO_ROOT}/chart/keel/Chart.yaml")"

[[ "$("${RESOLVER}")" == "${chart_version}" ]]
[[ "$(GITHUB_REF=refs/tags/9.8.7 "${RESOLVER}")" == 9.8.7 ]]
[[ "$(GITHUB_REF=refs/tags/chart-v7.8.9 "${RESOLVER}")" == "${chart_version}" ]]
[[ "$(KEEL_APP_VERSION=3.2.1 GITHUB_REF=refs/tags/9.8.7 "${RESOLVER}")" == 3.2.1 ]]

if GITHUB_REF=refs/tags/not-semver "${RESOLVER}" >/dev/null 2>&1; then
  printf '[release-version-test] ERROR: invalid application tag was accepted\n' >&2
  exit 1
fi

if GITHUB_REF=refs/tags/9.8.7 KEEL_RELEASE_SKIP_IMAGE=true \
  "${REPO_ROOT}/scripts/package-release.sh" >/dev/null 2>&1; then
  printf '[release-version-test] ERROR: application tag without matching chart appVersion was accepted\n' >&2
  exit 1
fi

printf '[release-version-test] application version resolution passed\n'
