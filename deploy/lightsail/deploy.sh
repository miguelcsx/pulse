#!/usr/bin/env sh
set -eu

cd "$(dirname "$0")"

if [ ! -f .env ]; then
  echo "deploy/lightsail/.env is missing" >&2
  exit 1
fi

docker compose pull postgres redis stone web
docker compose --profile ops run --rm migrate

if [ "${SEED_DEMO_DATA:-false}" = "true" ] && [ ! -f .seeded ]; then
  docker compose --profile ops run --rm seed
  touch .seeded
fi

docker compose up -d stone web
docker image prune -f >/dev/null 2>&1 || true
