# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/grove"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing grove dependencies"
  npm ci
fi

echo "[INFO] Building grove for production"
npm run build

echo "[INFO] Grove production build complete → dist/"
