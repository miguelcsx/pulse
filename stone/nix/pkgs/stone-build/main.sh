# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/stone"

OUT_DIR="${1:-./bin}"
mkdir -p "${OUT_DIR}"

echo "[INFO] Building stone binaries → ${OUT_DIR}/"

CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/server" ./cmd/server
echo "[INFO] Built ${OUT_DIR}/server"

CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/migrate" ./cmd/migrate
echo "[INFO] Built ${OUT_DIR}/migrate"

CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "${OUT_DIR}/seed" ./cmd/seed
echo "[INFO] Built ${OUT_DIR}/seed"

echo "[INFO] All stone binaries built successfully"
