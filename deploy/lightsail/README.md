# Pulse Lightsail Deployment

Pulse is deployed to one AWS Lightsail VM. Normal deploys are fully automated:

1. push to `main`
2. GitHub Actions builds `stone` and `web`
3. Actions pushes images to GitHub Container Registry
4. Actions SSHes into Lightsail
5. Lightsail pulls images, runs migrations, seeds once, restarts, and health-checks

Lightsail does not compile Go or Node during normal deploys.

## Public URL

```text
http://100.27.8.245
```

## Runtime Layout

- `web`: Caddy serving the `ember` build and proxying `/api`, `/ws`, `/uploads`
- `stone`: Go API
- `postgres`: `pgvector/pgvector:pg17`
- `redis`: `redis:8-alpine`
- uploads: bind-mounted from `/home/ubuntu/pulse/stone/uploads`

## Required GitHub Secrets

These are configured in the GitHub repository secrets:

```text
LIGHTSAIL_HOST
LIGHTSAIL_USER
LIGHTSAIL_SSH_KEY
PULSE_ORIGIN
PULSE_POSTGRES_PASSWORD
PULSE_JWT_SECRET
SEED_DEMO_DATA
```

Security notes:

- Secrets are not committed to the repo.
- The workflow only runs on `push` to `main` and `workflow_dispatch`.
- Pull requests do not run this deploy workflow with secrets.
- The remote Docker daemon logs into GHCR only for the deploy command and logs out via `trap`.
- The VM receives only `deploy/lightsail`, demo media, and generated `.env`.

## Images

GitHub Actions publishes:

```text
ghcr.io/miguelcsx/pulse/stone:<commit-sha>
ghcr.io/miguelcsx/pulse/web:<commit-sha>
```

The `.env` written by the workflow pins the VM to the commit SHA images for that deploy.

## Verify

```sh
curl http://100.27.8.245/api/v1/health
```

Expected:

```json
{"checks":{"postgres":"ok","redis":"ok"},"status":"ok"}
```

## Shutdown After Demo

From the VM:

```sh
cd /home/ubuntu/pulse/deploy/lightsail
docker compose down
```

To stop AWS billing, delete the `pulse-demo` Lightsail instance from AWS.
