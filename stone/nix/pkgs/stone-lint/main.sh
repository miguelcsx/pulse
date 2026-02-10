# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/stone"

echo "[INFO] Running go vet"
go vet ./...

echo "[INFO] Running govulncheck"
govulncheck ./...

echo "[INFO] All stone lint checks passed"
