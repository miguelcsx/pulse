# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"

function usage {
  echo ""
  echo "  Usage: roots-services <command>"
  echo ""
  echo "  Commands:"
  echo "    start    Start PostgreSQL and Redis"
  echo "    stop     Stop PostgreSQL and Redis"
  echo "    status   Show service status"
  echo "    restart  Restart all services"
  echo ""
}

function status_services {
  local pg_port="${PULSE_PG_PORT:-5432}"
  local redis_port="${PULSE_REDIS_PORT:-6379}"

  echo ""
  echo "  Pulse infrastructure status"
  echo "  ─────────────────────────────"

  if pg_isready -h 127.0.0.1 -p "${pg_port}" -U pulse > /dev/null 2>&1; then
    echo "  PostgreSQL .... ✓ running on :${pg_port}"
  else
    echo "  PostgreSQL .... ✗ stopped"
  fi

  if redis-cli -h 127.0.0.1 -p "${redis_port}" ping > /dev/null 2>&1; then
    echo "  Redis ......... ✓ running on :${redis_port}"
  else
    echo "  Redis ......... ✗ stopped"
  fi

  echo ""
}

function start_services {
  echo "[INFO] Starting Pulse infrastructure services"
  echo ""

  roots-db-start
  roots-db-create
  roots-redis-start

  echo ""
  echo "[INFO] All services started"
  status_services
}

function stop_services {
  echo "[INFO] Stopping Pulse infrastructure services"
  echo ""

  roots-redis-stop
  roots-db-stop

  echo ""
  echo "[INFO] All services stopped"
}

function restart_services {
  stop_services
  echo ""
  start_services
}

command="${1:-}"

case "${command}" in
  start)
    start_services
    ;;
  stop)
    stop_services
    ;;
  status)
    status_services
    ;;
  restart)
    restart_services
    ;;
  *)
    usage
    exit 1
    ;;
esac
