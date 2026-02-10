# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
REDIS_PORT="${PULSE_REDIS_PORT:-6379}"
REDIS_PID="${PULSE_DATA_DIR}/redis.pid"

# Try PID file first
if [ -f "${REDIS_PID}" ]; then
  pid="$(cat "${REDIS_PID}")"
  if kill -0 "${pid}" 2>/dev/null; then
    echo "[INFO] Stopping Redis (PID ${pid}) on port ${REDIS_PORT}"
    redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" shutdown nosave 2>/dev/null || true

    # Wait for process to exit
    for _ in $(seq 1 10); do
      if ! kill -0 "${pid}" 2>/dev/null; then
        echo "[INFO] Redis stopped"
        rm -f "${REDIS_PID}"
        exit 0
      fi
      sleep 0.5
    done

    echo "[WARNING] Redis did not stop gracefully — sending SIGKILL"
    kill -9 "${pid}" 2>/dev/null || true
    rm -f "${REDIS_PID}"
    echo "[INFO] Redis killed"
    exit 0
  else
    echo "[INFO] Stale PID file found — removing"
    rm -f "${REDIS_PID}"
  fi
fi

# Fallback: try connecting directly
if redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" ping > /dev/null 2>&1; then
  echo "[INFO] Stopping Redis on port ${REDIS_PORT} via CLI"
  redis-cli -h 127.0.0.1 -p "${REDIS_PORT}" shutdown nosave 2>/dev/null || true
  echo "[INFO] Redis stopped"
  exit 0
fi

echo "[INFO] Redis is not running on port ${REDIS_PORT}"
