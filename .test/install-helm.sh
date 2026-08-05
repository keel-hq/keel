#!/usr/bin/env bash

set -Eeuo pipefail

HELM_VERSION="v3.18.4"
readonly HELM_VERSION

install_helm() {
  local destination="$1"
  local architecture archive checksum expected temporary
  architecture="$(uname -m)"
  case "${architecture}" in
    x86_64) architecture="amd64"; expected="f8180838c23d7c7d797b208861fecb591d9ce1690d8704ed1e4cb8e2add966c1" ;;
    aarch64|arm64) architecture="arm64"; expected="c0a45e67eef0c7416a8a8c9e9d5d2d30d70e4f4d3f7bea5de28241fffa8f3b89" ;;
    *) printf 'unsupported Helm architecture: %s\n' "${architecture}" >&2; return 1 ;;
  esac

  mkdir -p "$(dirname "${destination}")"
  temporary="$(mktemp -d)"
  archive="${temporary}/helm.tar.gz"
  curl --fail --location --show-error --silent \
    --output "${archive}" "https://get.helm.sh/helm-${HELM_VERSION}-linux-${architecture}.tar.gz"
  checksum="$(sha256sum "${archive}" | awk '{print $1}')"
  [[ "${checksum}" == "${expected}" ]] || {
    printf 'Helm archive checksum mismatch: expected %s, got %s\n' "${expected}" "${checksum}" >&2
    return 1
  }
  tar -xzf "${archive}" -C "${temporary}" "linux-${architecture}/helm"
  install -m 0755 "${temporary}/linux-${architecture}/helm" "${destination}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  [[ $# -eq 1 ]] || { printf 'usage: %s DESTINATION\n' "$0" >&2; exit 2; }
  install_helm "$1"
fi
