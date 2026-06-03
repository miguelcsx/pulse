#!/usr/bin/env sh
set -eu

INSTANCE_NAME="${INSTANCE_NAME:-pulse-demo}"
AWS_REGION="${AWS_REGION:-us-east-1}"
BLUEPRINT_ID="${BLUEPRINT_ID:-ubuntu_24_04}"
BUNDLE_ID="${BUNDLE_ID:-micro_3_0}"
AVAILABILITY_ZONE="${AVAILABILITY_ZONE:-${AWS_REGION}a}"
KEY_PAIR_NAME="${KEY_PAIR_NAME:-pulse-demo-actions-2}"
LOCAL_KEY_PATH="${LOCAL_KEY_PATH:-.lightsail/pulse-demo-actions-2.pem}"
SECRETS_FILE="${SECRETS_FILE:-.lightsail/secrets.env}"
REPO_SLUG="${REPO_SLUG:-}"
LIGHTSAIL_USER="${LIGHTSAIL_USER:-ubuntu}"
SEED_DEMO_DATA="${SEED_DEMO_DATA:-true}"

if [ -z "$REPO_SLUG" ]; then
  if command -v gh >/dev/null 2>&1; then
    REPO_SLUG="$(gh repo view --json nameWithOwner --jq .nameWithOwner 2>/dev/null || true)"
  fi
fi

mkdir -p "$(dirname "$LOCAL_KEY_PATH")"

if ! aws lightsail get-key-pair \
  --region "$AWS_REGION" \
  --key-pair-name "$KEY_PAIR_NAME" >/dev/null 2>&1; then
  private_key_output="$(aws lightsail create-key-pair \
    --region "$AWS_REGION" \
    --key-pair-name "$KEY_PAIR_NAME" \
    --query 'privateKeyBase64' \
    --output text)"
  if printf '%s' "$private_key_output" | grep -q "BEGIN RSA PRIVATE KEY"; then
    printf '%s\n' "$private_key_output" > "$LOCAL_KEY_PATH"
  else
    printf '%s' "$private_key_output" | base64 -d > "$LOCAL_KEY_PATH"
  fi
  chmod 600 "$LOCAL_KEY_PATH"
elif [ ! -f "$LOCAL_KEY_PATH" ]; then
  echo "Lightsail key pair ${KEY_PAIR_NAME} exists, but ${LOCAL_KEY_PATH} is missing." >&2
  echo "Use a new KEY_PAIR_NAME or restore the private key." >&2
  exit 1
fi

if ! aws lightsail get-instance \
  --region "$AWS_REGION" \
  --instance-name "$INSTANCE_NAME" >/dev/null 2>&1; then
  aws lightsail create-instances \
    --region "$AWS_REGION" \
    --instance-names "$INSTANCE_NAME" \
    --availability-zone "$AVAILABILITY_ZONE" \
    --blueprint-id "$BLUEPRINT_ID" \
    --bundle-id "$BUNDLE_ID" \
    --key-pair-name "$KEY_PAIR_NAME" \
    --user-data file://cloud-init.sh >/dev/null
fi

aws lightsail open-instance-public-ports \
  --region "$AWS_REGION" \
  --instance-name "$INSTANCE_NAME" \
  --port-info fromPort=80,toPort=80,protocol=TCP >/dev/null

for _ in 1 2 3 4 5 6 7 8 9 10 11 12; do
  public_ip="$(aws lightsail get-instance \
    --region "$AWS_REGION" \
    --instance-name "$INSTANCE_NAME" \
    --query 'instance.publicIpAddress' \
    --output text)"
  if [ -n "$public_ip" ] && [ "$public_ip" != "None" ]; then
    break
  fi
  sleep 5
done

origin="http://${public_ip}"
if [ -f "$SECRETS_FILE" ]; then
  # shellcheck disable=SC1090
  . "$SECRETS_FILE"
else
  postgres_password="$(openssl rand -hex 24)"
  jwt_secret="$(openssl rand -hex 32)"
  cat > "$SECRETS_FILE" <<EOF
postgres_password=${postgres_password}
jwt_secret=${jwt_secret}
EOF
  chmod 600 "$SECRETS_FILE"
fi
ssh_key="$(cat "$LOCAL_KEY_PATH")"

if [ -n "$REPO_SLUG" ] && command -v gh >/dev/null 2>&1; then
  printf '%s' "$public_ip" | gh secret set LIGHTSAIL_HOST --repo "$REPO_SLUG"
  printf '%s' "$LIGHTSAIL_USER" | gh secret set LIGHTSAIL_USER --repo "$REPO_SLUG"
  printf '%s' "$ssh_key" | gh secret set LIGHTSAIL_SSH_KEY --repo "$REPO_SLUG"
  printf '%s' "$origin" | gh secret set PULSE_ORIGIN --repo "$REPO_SLUG"
  printf '%s' "$postgres_password" | gh secret set PULSE_POSTGRES_PASSWORD --repo "$REPO_SLUG"
  printf '%s' "$jwt_secret" | gh secret set PULSE_JWT_SECRET --repo "$REPO_SLUG"
  printf '%s' "$SEED_DEMO_DATA" | gh secret set SEED_DEMO_DATA --repo "$REPO_SLUG"
fi

cat <<EOF
Lightsail instance: ${INSTANCE_NAME}
Region: ${AWS_REGION}
Public URL: ${origin}
SSH: ssh -i ${LOCAL_KEY_PATH} ${LIGHTSAIL_USER}@${public_ip}

GitHub secrets configured: $([ -n "$REPO_SLUG" ] && echo "yes, for ${REPO_SLUG}" || echo "no")

If GitHub secrets were not configured automatically, set:
  LIGHTSAIL_HOST=${public_ip}
  LIGHTSAIL_USER=${LIGHTSAIL_USER}
  LIGHTSAIL_SSH_KEY=<contents of ${LOCAL_KEY_PATH}>
  PULSE_ORIGIN=${origin}
  PULSE_POSTGRES_PASSWORD=${postgres_password}
  PULSE_JWT_SECRET=${jwt_secret}
  SEED_DEMO_DATA=${SEED_DEMO_DATA}
EOF
