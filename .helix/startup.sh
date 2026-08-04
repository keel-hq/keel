#!/bin/bash
set -euo pipefail

if ! command -v gcc >/dev/null 2>&1; then
  sudo apt-get update
  sudo apt-get install -y --no-install-recommends build-essential
fi
