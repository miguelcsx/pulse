# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
PG_DATA="${PULSE_DATA_DIR}/postgres"
PG_PORT="${PULSE_PG_PORT:-5433}"

if [ ! -d "${PG_DATA}" ]; then
  echo "[INFO] No PostgreSQL data directory found at ${PG_DATA} — nothing to stop"
  exit 0
fi

if ! pg_isready -h 127.0.0.1 -p "${PG_PORT}" -U pulse > /dev/null 2>&1; then
  echo "[INFO] PostgreSQL is not running on port ${PG_PORT}"
  exit 0
fi

echo "[INFO] Stopping PostgreSQL on port ${PG_PORT}"
pg_ctl \
  -D "${PG_DATA}" \
  -m fast \
  stop \
  > /dev/null

echo "[INFO] PostgreSQL stopped"
