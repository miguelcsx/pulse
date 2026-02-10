# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/grove"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing grove dependencies"
  npm ci
fi

echo "[INFO] Running grove lint"
npm run lint

echo "[INFO] All grove lint checks passed"
