#!/usr/bin/env sh
set -eu

if [ "$#" -lt 1 ]; then
  echo "usage: ./generate-env.sh http://PUBLIC_IP [output-file]" >&2
  echo "   or: ./generate-env.sh https://pulse.example.com [output-file]" >&2
  exit 1
fi

origin="${1%/}"
out="${2:-.env}"
if [ -f .lightsail/secrets.env ]; then
  # shellcheck disable=SC1091
  . .lightsail/secrets.env
else
  postgres_password="$(openssl rand -hex 24)"
  jwt_secret="$(openssl rand -hex 32)"
fi

cat > "$out" <<EOF
ENV=production
PORT=8080
STONE_IMAGE=ghcr.io/miguelcsx/pulse/stone:latest
WEB_IMAGE=ghcr.io/miguelcsx/pulse/web:latest

POSTGRES_PASSWORD=${postgres_password}
DATABASE_URL=postgres://pulse:${postgres_password}@postgres:5432/pulse?sslmode=disable
REDIS_URL=redis://redis:6379/0
JWT_SECRET=${jwt_secret}

DEMO_AUTH_ENABLED=true
CORS_ORIGINS=${origin}
WS_ORIGINS=${origin}
AUTH_COOKIE_SECURE=false

STORAGE_PATH=/app/uploads
STORAGE_BASE_URL=/uploads
UPLOAD_PUBLIC_PATH=/uploads
STORAGE_MAX_SIZE_MB=25

TRUSTED_PROXIES=172.16.0.0/12,127.0.0.1,::1
RATE_LIMIT_RPS=30
RATE_LIMIT_BURST=60
RATE_LIMIT_FAIL_OPEN=true

METRICS_ENABLED=false
OLLAMA_BASE_URL=
OLLAMA_MODEL=
EOF

echo "Wrote ${out} for ${origin}"
