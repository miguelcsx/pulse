# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
PG_DATA="${PULSE_DATA_DIR}/postgres"
PG_LOG="${PULSE_DATA_DIR}/postgres.log"
PG_PORT="${PULSE_PG_PORT:-5432}"
PG_SOCKET_DIR="${PULSE_DATA_DIR}/pg_sockets"

mkdir -p "${PULSE_DATA_DIR}" "${PG_SOCKET_DIR}"

if [ ! -d "${PG_DATA}" ]; then
  echo "[INFO] Initializing PostgreSQL data directory at ${PG_DATA}"
  initdb \
    --pgdata="${PG_DATA}" \
    --username=pulse \
    --auth=trust \
    --no-locale \
    --encoding=UTF8 \
    > /dev/null

  # Configure for local development
  {
    echo "unix_socket_directories = '${PG_SOCKET_DIR}'"
    echo "port = ${PG_PORT}"
    echo "listen_addresses = '127.0.0.1'"
    echo "log_destination = 'stderr'"
    echo "logging_collector = off"
    echo "shared_preload_libraries = 'vector'"
  } >> "${PG_DATA}/postgresql.conf"

  echo "[INFO] PostgreSQL data directory initialized"
fi

# Check if already running
if pg_isready -h 127.0.0.1 -p "${PG_PORT}" -U pulse > /dev/null 2>&1; then
  echo "[INFO] PostgreSQL is already running on port ${PG_PORT}"
  exit 0
fi

echo "[INFO] Starting PostgreSQL on port ${PG_PORT}"
pg_ctl \
  -D "${PG_DATA}" \
  -l "${PG_LOG}" \
  -o "-k ${PG_SOCKET_DIR}" \
  start \
  > /dev/null

# Wait for it to be ready
for i in $(seq 1 30); do
  if pg_isready -h 127.0.0.1 -p "${PG_PORT}" -U pulse > /dev/null 2>&1; then
    echo "[INFO] PostgreSQL is ready on port ${PG_PORT}"
    exit 0
  fi
  sleep 0.5
done

echo "[ERROR] PostgreSQL failed to start within 15 seconds — check ${PG_LOG}"
exit 1
