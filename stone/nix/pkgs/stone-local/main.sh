# shellcheck shell=bash

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"

# ── Load secrets ──────────────────────────────────────────────────────────────
# shellcheck disable=SC1091
source stone-envars dev

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
    cmd: ["bash", "-c", "cd ${ROOT}/stone && source stone-envars dev && go run ./cmd/server"]
YAML

# ── Launch mprocs ─────────────────────────────────────────────────────────────
echo ""
echo "  stone  http://localhost:8080  (Go API)"
echo ""

mprocs --config "${MPROCS_CONFIG}"
