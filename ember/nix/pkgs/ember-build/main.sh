# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/ember"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing ember dependencies"
  npm ci
fi

echo "[INFO] Building ember for production"
npm run build

echo "[INFO] Ember production build complete → dist/"
