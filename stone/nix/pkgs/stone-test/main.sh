# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/stone"

echo "[INFO] Running stone tests"
go test ./... "$@"
