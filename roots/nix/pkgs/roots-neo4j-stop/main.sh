# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
NEO4J_DATA="${PULSE_DATA_DIR}/neo4j"
NEO4J_CONF_DIR="${NEO4J_DATA}/conf"

NEO4J_BIN="$(command -v neo4j)"
NEO4J_HOME="$(cd "$(dirname "${NEO4J_BIN}")/.." && pwd)"

export NEO4J_HOME
export NEO4J_CONF="${NEO4J_CONF_DIR}"

if [ ! -d "${NEO4J_DATA}" ]; then
  echo "[INFO] Neo4j data directory not found; nothing to stop"
  exit 0
fi

echo "[INFO] Stopping Neo4j"
neo4j stop >/dev/null
echo "[INFO] Neo4j stopped"
