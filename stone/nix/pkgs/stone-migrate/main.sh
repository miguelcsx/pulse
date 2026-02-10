# shellcheck shell=bash

set -euo pipefail

cd "$(git rev-parse --show-toplevel)/stone"

# shellcheck disable=SC1091
source stone-envars dev

echo "[INFO] Running stone migrations"
go run ./cmd/migrate "$@"
