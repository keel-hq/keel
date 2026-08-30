#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
CHECKER="${REPO_ROOT}/scripts/check-chart-version-bump.sh"
FIXTURE="$(mktemp -d)"
trap 'rm -rf "${FIXTURE}"' EXIT

git -C "${FIXTURE}" init --quiet
git -C "${FIXTURE}" config user.name chart-version-test
git -C "${FIXTURE}" config user.email chart-version-test@example.invalid
mkdir -p "${FIXTURE}/chart/keel"
printf 'version: 1.2.0\nappVersion: 0.22.1\n' >"${FIXTURE}/chart/keel/Chart.yaml"
printf 'replicas: 1\n' >"${FIXTURE}/chart/keel/values.yaml"
git -C "${FIXTURE}" add chart/keel
git -C "${FIXTURE}" commit --quiet -m base
base_ref="$(git -C "${FIXTURE}" rev-parse HEAD)"

printf 'replicas: 2\n' >"${FIXTURE}/chart/keel/values.yaml"
git -C "${FIXTURE}" add chart/keel/values.yaml
git -C "${FIXTURE}" commit --quiet -m chart-change
if (cd "${FIXTURE}" && KEEL_CHART_BASE_REF="${base_ref}" "${CHECKER}") >/dev/null 2>&1; then
  printf '[chart-version-test] ERROR: chart change without a version bump was accepted\n' >&2
  exit 1
fi

sed -i.bak 's/version: 1.2.0/version: 1.2.1/' "${FIXTURE}/chart/keel/Chart.yaml"
rm "${FIXTURE}/chart/keel/Chart.yaml.bak"
git -C "${FIXTURE}" add chart/keel/Chart.yaml
git -C "${FIXTURE}" commit --quiet -m version-bump
(cd "${FIXTURE}" && KEEL_CHART_BASE_REF="${base_ref}" "${CHECKER}") >/dev/null

printf '[chart-version-test] chart changes require a version increase\n'
