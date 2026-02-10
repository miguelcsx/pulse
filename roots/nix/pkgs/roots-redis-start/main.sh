# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
REDIS_DATA="${PULSE_DATA_DIR}/redis"
REDIS_LOG="${PULSE_DATA_DIR}/redis.log"
REDIS_PORT="${PULSE_REDIS_PORT:-6379}"
REDIS_PID="${PULSE_DATA_DIR}/redis.pid"

mkdir -p "${REDIS_DATA}"

# Check if already running
if [ -f "${REDIS_PID}" ] && kill -0 "$(cat "${REDIS_PID}")" 2>/dev/null; then
  echo "[INFO] Redis is already running on port ${REDIS_PORT} (PID $(cat "${REDIS_PID}"))"
  exit 0
fi

# Check if port is already in use
if redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
  echo "[INFO] Redis is already responding on port ${REDIS_PORT}"
  exit 0
fi

echo "[INFO] Starting Redis on port ${REDIS_PORT}"
redis-server \
  --daemonize yes \
  --bind 127.0.0.1 \
  --port "${REDIS_PORT}" \
  --dir "${REDIS_DATA}" \
  --dbfilename pulse.rdb \
  --logfile "${REDIS_LOG}" \
  --pidfile "${REDIS_PID}" \
  --save "60 1" \
  --appendonly no

# Wait for it to be ready
for _ in $(seq 1 30); do
  if redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
    echo "[INFO] Redis is ready on port ${REDIS_PORT}"
    exit 0
  fi
  sleep 0.5
done

echo "[ERROR] Redis failed to start within 15 seconds — check ${REDIS_LOG}"
exit 1
