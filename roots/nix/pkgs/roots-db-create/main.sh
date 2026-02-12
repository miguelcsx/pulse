# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
PG_PORT="${PULSE_PG_PORT:-5433}"
DB_NAME="${PULSE_DB_NAME:-pulse_dev}"
DB_USER="${PULSE_DB_USER:-pulse}"

if ! pg_isready -h 127.0.0.1 -p "${PG_PORT}" -U "${DB_USER}" > /dev/null 2>&1; then
  echo "[ERROR] PostgreSQL is not running on port ${PG_PORT} — run roots-db-start first"
  exit 1
fi

# Create database if it doesn't exist
if psql -h 127.0.0.1 -p "${PG_PORT}" -U "${DB_USER}" -lqt | cut -d \| -f 1 | grep -qw "${DB_NAME}"; then
  echo "[INFO] Database '${DB_NAME}' already exists"
else
  echo "[INFO] Creating database '${DB_NAME}'"
  createdb \
    -h 127.0.0.1 \
    -p "${PG_PORT}" \
    -U "${DB_USER}" \
    -O "${DB_USER}" \
    "${DB_NAME}"
  echo "[INFO] Database '${DB_NAME}' created"
fi

# Enable extensions
echo "[INFO] Enabling extensions on '${DB_NAME}'"
psql -h 127.0.0.1 -p "${PG_PORT}" -U "${DB_USER}" -d "${DB_NAME}" -c '
  CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
  CREATE EXTENSION IF NOT EXISTS "vector";
' > /dev/null

echo "[INFO] Database '${DB_NAME}' is ready"
echo ""
echo "  Connection string:"
echo "    postgres://${DB_USER}@127.0.0.1:${PG_PORT}/${DB_NAME}?sslmode=disable"
echo ""
