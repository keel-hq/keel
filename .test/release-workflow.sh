#!/usr/bin/env bash

set -Eeuo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
readonly REPO_ROOT
workflow="${REPO_ROOT}/.github/workflows/ci.yml"

if grep -Eq 'DOCKERHUB_|hub\.docker\.com|keelhq/keel' "${workflow}"; then
  printf '[release-workflow-test] ERROR: CI release workflow must publish only to GHCR\n' >&2
  exit 1
fi

grep -Fq 'outputs: type=image,name=ghcr.io/${{ github.repository }},push-by-digest=true,name-canonical=true,push=true' "${workflow}"
grep -Fq 'IMAGE_NAME: ghcr.io/${{ github.repository }}' "${workflow}"

printf '[release-workflow-test] GHCR-only publication passed\n'
