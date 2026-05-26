# shellcheck shell=bash

set -euo pipefail

PULSE_DATA_DIR="${PULSE_DATA_DIR:-$(git rev-parse --show-toplevel)/.data}"
NEO4J_DATA="${PULSE_DATA_DIR}/neo4j"
NEO4J_CONF_DIR="${NEO4J_DATA}/conf"
BOLT_PORT="${PULSE_NEO4J_BOLT_PORT:-7687}"
HTTP_PORT="${PULSE_NEO4J_HTTP_PORT:-7474}"
HTTPS_PORT="${PULSE_NEO4J_HTTPS_PORT:-7473}"
NEO4J_USER="${PULSE_NEO4J_USER:-neo4j}"
NEO4J_PASSWORD="${PULSE_NEO4J_PASSWORD:-pulse_dev}"
NEO4J_DB="${PULSE_NEO4J_DATABASE:-pulse}"

NEO4J_BIN="$(command -v neo4j)"
NEO4J_HOME="$(cd "$(dirname "${NEO4J_BIN}")/.." && pwd)"

mkdir -p "${NEO4J_CONF_DIR}" "${NEO4J_DATA}/data" "${NEO4J_DATA}/logs" "${NEO4J_DATA}/run" "${NEO4J_DATA}/plugins"

CONF_FILE="${NEO4J_CONF_DIR}/neo4j.conf"
cat > "${CONF_FILE}" <<EOF
server.http.listen_address=127.0.0.1:${HTTP_PORT}
server.https.listen_address=127.0.0.1:${HTTPS_PORT}
server.bolt.listen_address=127.0.0.1:${BOLT_PORT}
server.directories.data=${NEO4J_DATA}/data
server.directories.logs=${NEO4J_DATA}/logs
server.directories.run=${NEO4J_DATA}/run
server.directories.plugins=${NEO4J_DATA}/plugins
initial.dbms.default_database=${NEO4J_DB}
dbms.security.auth_enabled=true
EOF

export NEO4J_HOME
export NEO4J_CONF="${NEO4J_CONF_DIR}"

AUTH_FILE="${NEO4J_DATA}/data/dbms/auth"
if [ ! -f "${AUTH_FILE}" ]; then
  echo "[INFO] Initializing Neo4j password for user ${NEO4J_USER}"
  neo4j-admin dbms set-initial-password "${NEO4J_PASSWORD}"
fi

if (echo > /dev/tcp/127.0.0.1/"${BOLT_PORT}") >/dev/null 2>&1; then
  echo "[INFO] Neo4j already running on port ${BOLT_PORT}"
  exit 0
fi

echo "[INFO] Starting Neo4j (bolt:${BOLT_PORT}, http:${HTTP_PORT})"
neo4j start >/dev/null

for _i in $(seq 1 30); do
  if (echo > /dev/tcp/127.0.0.1/"${BOLT_PORT}") >/dev/null 2>&1; then
    echo "[INFO] Neo4j is ready on port ${BOLT_PORT}"
    exit 0
  fi
  sleep 0.5
done

echo "[ERROR] Neo4j failed to start within 15 seconds"
exit 1
