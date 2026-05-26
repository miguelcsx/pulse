# shellcheck shell=bash

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"

export ENV="${ENV:-development}"
export DATABASE_URL="${DATABASE_URL:-postgres://pulse@127.0.0.1:5433/pulse_dev?sslmode=disable}"
export REDIS_URL="${REDIS_URL:-redis://127.0.0.1:6379/0}"
export JWT_SECRET="${JWT_SECRET:-pulse-local-development-secret}"
export CORS_ORIGINS="${CORS_ORIGINS:-http://localhost:5173,http://127.0.0.1:5173,http://localhost:5174,http://127.0.0.1:5174}"
export WS_ORIGINS="${WS_ORIGINS:-${CORS_ORIGINS}}"

# ── Start infrastructure ──────────────────────────────────────────────────────
echo "[INFO] Starting infrastructure"
roots-services start

echo "[INFO] Running database migrations"
cd "${ROOT}/stone" && go run ./cmd/migrate
cd "${ROOT}"

# ── Generate mprocs config ────────────────────────────────────────────────────
MPROCS_CONFIG="$(mktemp /tmp/pulse-mprocs-XXXXXX.yaml)"
trap 'rm -f "${MPROCS_CONFIG}"' EXIT

cat > "${MPROCS_CONFIG}" << YAML
procs:
  stone:
    cmd: ["bash", "-c", "cd ${ROOT}/stone && ENV='${ENV}' DATABASE_URL='${DATABASE_URL}' REDIS_URL='${REDIS_URL}' JWT_SECRET='${JWT_SECRET}' CORS_ORIGINS='${CORS_ORIGINS}' WS_ORIGINS='${WS_ORIGINS}' go run ./cmd/server"]
YAML

# ── Launch mprocs ─────────────────────────────────────────────────────────────
echo ""
echo "  stone  http://localhost:8080  (Go API)"
echo ""

mprocs --config "${MPROCS_CONFIG}"
