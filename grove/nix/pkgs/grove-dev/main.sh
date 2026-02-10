# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/grove"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing grove dependencies"
  npm ci
fi

echo "[INFO] Starting grove Vite dev server on :5174"
npm run dev "$@"
