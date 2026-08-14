#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
CHART_FILE="${REPO_ROOT}/chart/keel/Chart.yaml"

if [[ -n "${KEEL_APP_VERSION:-}" ]]; then
  app_version="${KEEL_APP_VERSION}"
elif [[ "${GITHUB_REF:-}" == refs/tags/* && "${GITHUB_REF:-}" != refs/tags/chart-* ]]; then
  app_version="${GITHUB_REF#refs/tags/}"
else
  app_version="$(awk '$1 == "appVersion:" {print $2; exit}' "${CHART_FILE}")"
fi

if [[ ! "${app_version}" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.+][0-9A-Za-z.-]+)?$ ]]; then
  printf '[release-version] ERROR: invalid application version: %s\n' "${app_version}" >&2
  exit 1
fi

printf '%s\n' "${app_version}"
