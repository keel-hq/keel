#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
BASE_REF="${KEEL_CHART_BASE_REF:-${1:-}}"
TARGET_REF="${KEEL_CHART_TARGET_REF:-HEAD}"
CHART_PATH="chart/keel"
CHART_FILE="${CHART_PATH}/Chart.yaml"

fail() { printf '[chart-version] ERROR: %s\n' "$*" >&2; exit 1; }
log() { printf '[chart-version] %s\n' "$*"; }

version_from_ref() {
  git -C "${REPO_ROOT}" show "$1:${CHART_FILE}" |
    awk '$1 == "version:" {print $2; exit}'
}

semver_gt() {
  local candidate="${1#v}" previous="${2#v}"
  local candidate_core="${candidate%%+*}" previous_core="${previous%%+*}"
  local candidate_pre="" previous_pre=""
  local -a candidate_parts previous_parts candidate_ids previous_ids
  local index candidate_id previous_id

  if [[ "${candidate_core}" == *-* ]]; then
    candidate_pre="${candidate_core#*-}"
    candidate_core="${candidate_core%%-*}"
  fi
  if [[ "${previous_core}" == *-* ]]; then
    previous_pre="${previous_core#*-}"
    previous_core="${previous_core%%-*}"
  fi
  [[ "${candidate_core}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || return 1
  [[ "${previous_core}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || return 1

  IFS=. read -r -a candidate_parts <<<"${candidate_core}"
  IFS=. read -r -a previous_parts <<<"${previous_core}"
  for index in 0 1 2; do
    if ((10#${candidate_parts[index]} > 10#${previous_parts[index]})); then
      return 0
    fi
    if ((10#${candidate_parts[index]} < 10#${previous_parts[index]})); then
      return 1
    fi
  done

  [[ -z "${previous_pre}" && -n "${candidate_pre}" ]] && return 1
  [[ -n "${previous_pre}" && -z "${candidate_pre}" ]] && return 0
  [[ -z "${previous_pre}" && -z "${candidate_pre}" ]] && return 1

  IFS=. read -r -a candidate_ids <<<"${candidate_pre}"
  IFS=. read -r -a previous_ids <<<"${previous_pre}"
  for ((index = 0; index < ${#candidate_ids[@]} || index < ${#previous_ids[@]}; index++)); do
    [[ ${index} -ge ${#candidate_ids[@]} ]] && return 1
    [[ ${index} -ge ${#previous_ids[@]} ]] && return 0
    candidate_id="${candidate_ids[index]}"
    previous_id="${previous_ids[index]}"
    [[ "${candidate_id}" == "${previous_id}" ]] && continue
    if [[ "${candidate_id}" =~ ^[0-9]+$ && "${previous_id}" =~ ^[0-9]+$ ]]; then
      ((10#${candidate_id} > 10#${previous_id})) && return 0
      return 1
    fi
    [[ "${candidate_id}" =~ ^[0-9]+$ ]] && return 1
    [[ "${previous_id}" =~ ^[0-9]+$ ]] && return 0
    [[ "${candidate_id}" > "${previous_id}" ]]
    return
  done
  return 1
}

[[ -n "${BASE_REF}" ]] || fail "KEEL_CHART_BASE_REF or a base-ref argument is required"
if [[ "${BASE_REF}" =~ ^0+$ ]]; then
  log "no base commit exists; skipping the initial branch push"
  exit 0
fi
git -C "${REPO_ROOT}" cat-file -e "${BASE_REF}:${CHART_FILE}" 2>/dev/null || \
  fail "base ref ${BASE_REF} does not contain ${CHART_FILE}"
git -C "${REPO_ROOT}" cat-file -e "${TARGET_REF}:${CHART_FILE}" 2>/dev/null || \
  fail "target ref ${TARGET_REF} does not contain ${CHART_FILE}"

if git -C "${REPO_ROOT}" diff --quiet "${BASE_REF}" "${TARGET_REF}" -- "${CHART_PATH}"; then
  log "no Helm chart changes detected"
  exit 0
fi

base_version="$(version_from_ref "${BASE_REF}")"
target_version="$(version_from_ref "${TARGET_REF}")"
[[ -n "${base_version}" && -n "${target_version}" ]] || fail "could not read chart versions"
semver_gt "${target_version}" "${base_version}" || \
  fail "chart files changed, but Chart.yaml version did not increase (${base_version} -> ${target_version})"

log "chart version increased from ${base_version} to ${target_version}"
