# shellcheck shell=bash

set -euo pipefail

function usage {
  echo ""
  echo "  Usage: roots-services <command>"
  echo ""
  echo "  Commands:"
  echo "    start    Start PostgreSQL, Redis, and Neo4j"
  echo "    stop     Stop PostgreSQL, Redis, and Neo4j"
  echo "    status   Show service status"
  echo "    restart  Restart all services"
  echo ""
}

function status_services {
  local pg_port="${PULSE_PG_PORT:-5433}"
  local redis_port="${PULSE_REDIS_PORT:-6379}"
  local neo4j_port="${PULSE_NEO4J_BOLT_PORT:-7687}"

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

  if (echo > /dev/tcp/127.0.0.1/"${neo4j_port}") >/dev/null 2>&1; then
    echo "  Neo4j ......... ✓ running on :${neo4j_port}"
  else
    echo "  Neo4j ......... ✗ stopped"
  fi

  echo ""
}

function start_services {
  echo "[INFO] Starting Pulse infrastructure services"
  echo ""

  roots-db-start
  roots-db-create
  roots-redis-start
  roots-neo4j-start

  echo ""
  echo "[INFO] All services started"
  status_services
}

function stop_services {
  echo "[INFO] Stopping Pulse infrastructure services"
  echo ""

  roots-redis-stop
  roots-db-stop
  roots-neo4j-stop

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
