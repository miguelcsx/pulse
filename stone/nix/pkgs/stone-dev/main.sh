# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/stone"

export ENV="${ENV:-development}"
export DATABASE_URL="${DATABASE_URL:-postgres://pulse@127.0.0.1:5433/pulse_dev?sslmode=disable}"
export REDIS_URL="${REDIS_URL:-redis://127.0.0.1:6379/0}"
export JWT_SECRET="${JWT_SECRET:-pulse-local-development-secret}"
export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174}"
export WS_ORIGINS="${WS_ORIGINS:-${CORS_ORIGINS}}"

echo "[INFO] Starting stone dev server"
go run ./cmd/server "$@"
