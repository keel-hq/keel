#!/usr/bin/env bash

set -Eeuo pipefail

HELM_VERSION="v3.18.4"
readonly HELM_VERSION

install_helm() {
  local destination="$1"
  local architecture archive checksum expected operating_system temporary
  operating_system="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "${operating_system}" in
    linux|darwin) ;;
    *) printf 'unsupported Helm operating system: %s\n' "${operating_system}" >&2; return 1 ;;
  esac
  architecture="$(uname -m)"
  case "${architecture}" in
    x86_64) architecture="amd64" ;;
    aarch64|arm64) architecture="arm64" ;;
    *) printf 'unsupported Helm architecture: %s\n' "${architecture}" >&2; return 1 ;;
  esac
  case "${operating_system}-${architecture}" in
    linux-amd64) expected="f8180838c23d7c7d797b208861fecb591d9ce1690d8704ed1e4cb8e2add966c1" ;;
    linux-arm64) expected="c0a45e67eef0c7416a8a8c9e9d5d2d30d70e4f4d3f7bea5de28241fffa8f3b89" ;;
    darwin-amd64) expected="860a7238285b44b5dc7b3c4dad6194316885d7015d77c34e23177e0e9554af8f" ;;
    darwin-arm64) expected="041849741550b20710d7ad0956e805ebd960b483fe978864f8e7fdd03ca84ec8" ;;
  esac

  mkdir -p "$(dirname "${destination}")"
  temporary="$(mktemp -d)"
  archive="${temporary}/helm.tar.gz"
  curl --fail --location --show-error --silent \
    --output "${archive}" "https://get.helm.sh/helm-${HELM_VERSION}-${operating_system}-${architecture}.tar.gz"
  if command -v sha256sum >/dev/null; then
    checksum="$(sha256sum "${archive}" | awk '{print $1}')"
  else
    checksum="$(shasum -a 256 "${archive}" | awk '{print $1}')"
  fi
  [[ "${checksum}" == "${expected}" ]] || {
    printf 'Helm archive checksum mismatch: expected %s, got %s\n' "${expected}" "${checksum}" >&2
    return 1
  }
  tar -xzf "${archive}" -C "${temporary}" "${operating_system}-${architecture}/helm"
  install -m 0755 "${temporary}/${operating_system}-${architecture}/helm" "${destination}"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  [[ $# -eq 1 ]] || { printf 'usage: %s DESTINATION\n' "$0" >&2; exit 2; }
  install_helm "$1"
fi
