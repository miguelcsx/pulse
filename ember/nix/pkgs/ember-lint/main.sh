# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/ember"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing ember dependencies"
  npm ci
fi

echo "[INFO] Running ember lint"
npm run lint

echo "[INFO] All ember lint checks passed"
