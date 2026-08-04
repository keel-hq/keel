#!/bin/bash
set -euo pipefail

if ! command -v gcc >/dev/null 2>&1; then
  sudo apt-get update
  sudo apt-get install -y --no-install-recommends build-essential
fi

K3D_VERSION="v5.9.0"
if ! command -v k3d >/dev/null 2>&1; then
  k3d_download="$(mktemp)"
  curl -fsSL "https://github.com/k3d-io/k3d/releases/download/${K3D_VERSION}/k3d-linux-amd64" -o "$k3d_download"
  chmod +x "$k3d_download"
  sudo install "$k3d_download" /usr/local/bin/k3d
  rm -f "$k3d_download"
fi
