# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/ember"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing ember dependencies"
  npm ci
fi

echo "[INFO] Starting ember Vite dev server on :5173"
npm run dev "$@"
