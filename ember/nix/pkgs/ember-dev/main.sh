# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/ember"

export VITE_DEV_BACKEND_ORIGIN="${VITE_DEV_BACKEND_ORIGIN:-http://localhost:8080}"

if [ ! -d "node_modules" ]; then
  echo "[INFO] Installing ember dependencies"
  npm ci
fi

echo "[INFO] Starting ember Vite dev server on :5173"
echo "[INFO] Backend proxy → ${VITE_DEV_BACKEND_ORIGIN}"
npm run dev -- --host 127.0.0.1 "$@"
