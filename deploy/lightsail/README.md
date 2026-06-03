# Pulse demo on AWS Lightsail

This deploy runs Pulse on one cheap Lightsail VM:

- Caddy serves the `ember` build on port 80.
- Caddy proxies `/api`, `/ws`, and `/uploads` to `stone`.
- `stone` runs with Postgres, Redis, and local upload storage.
- Demo auth is enabled with `DEMO_AUTH_ENABLED=true`, so visitors only need a username.

## Fast path: provision once, auto-deploy on main

From your local machine, with AWS CLI and GitHub CLI logged in:

```sh
cd deploy/lightsail
./provision.sh
```

The script:

- creates/imports a local SSH key under `.lightsail/`
- creates the Lightsail instance if it does not exist
- installs Docker through `cloud-init.sh`
- opens port `80`
- creates GitHub Actions secrets for `miguelcsx/pulse`

Then push to `main`. `.github/workflows/deploy-lightsail.yml` will:

1. build `stone` and `web` in GitHub Actions
2. push both images to GitHub Container Registry
3. sync only deploy files and demo media to the VM
4. write `.env`
5. run `docker compose pull`, migrations, one-time seed, restart, and health check

Lightsail does not compile Go or Node during normal deploys.

Defaults:

```sh
AWS_REGION=us-east-1
INSTANCE_NAME=pulse-demo
BLUEPRINT_ID=ubuntu_24_04
BUNDLE_ID=micro_3_0
```

Use the cheapest bundle at your own risk. `micro_3_0` is the safer minimum because local Docker builds need memory; the script also enables a 2 GB swapfile.

## Manual path

### 1. Create the Lightsail instance

1. Create or sign in to an AWS account.
2. Open AWS Lightsail.
3. Create an instance:
   - Platform: Linux/Unix
   - Blueprint: Ubuntu 24.04 LTS
   - Size: cheapest plan that is available. Use 1 GB RAM minimum for a smoother demo.
4. In Networking, allow TCP ports `22` and `80`.
5. Copy the public IP.

For a one-day demo without a domain, use `http://PUBLIC_IP`. If you have a domain, point an A record to the public IP and use `Caddyfile.https`.

### 2. Install Docker on the VM

SSH into the instance and run:

```sh
sudo apt-get update
sudo apt-get install -y ca-certificates curl git
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo usermod -aG docker ubuntu
```

Log out and SSH back in so the Docker group applies.

On the smallest instances, add swap before building:

```sh
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### 3. Upload or clone the repo

Clone the repo on the VM, or copy this working tree to the VM. Then:

```sh
cd pulse/deploy/lightsail
chmod +x generate-env.sh
./generate-env.sh http://PUBLIC_IP
```

Or copy `.env.example` manually:

- Set `POSTGRES_PASSWORD`.
- Set the same password inside `DATABASE_URL`.
- Set `JWT_SECRET`.
- Replace `CORS_ORIGINS` and `WS_ORIGINS` with `http://PUBLIC_IP`.

Generate good local secrets with:

```sh
openssl rand -hex 24
openssl rand -hex 32
```

### 4. Build, migrate, seed, and start

```sh
docker compose build
docker compose --profile ops run --rm migrate
docker compose --profile ops run --rm seed
docker compose up -d stone web
```

Open:

```text
http://PUBLIC_IP
```

Use the demo username field to enter as any handle.

### 5. Verify

```sh
curl http://PUBLIC_IP/api/v1/health
curl -X POST http://PUBLIC_IP/api/v1/auth/demo \
  -H 'Content-Type: application/json' \
  -d '{"handle":"demo_guest"}'
docker compose logs --tail=100 stone
```

## Auto-deploy details

The deploy workflow uses these GitHub secrets:

```text
LIGHTSAIL_HOST
LIGHTSAIL_USER
LIGHTSAIL_SSH_KEY
PULSE_ORIGIN
PULSE_POSTGRES_PASSWORD
PULSE_JWT_SECRET
SEED_DEMO_DATA
```

`provision.sh` sets them automatically when `gh` is authenticated. Secrets are also persisted locally in `.lightsail/secrets.env` so rerunning the script does not rotate the DB/JWT secrets accidentally.

The workflow runs on every push to `main` and can also be started manually from GitHub Actions.

The VM only pulls prebuilt images:

```text
ghcr.io/miguelcsx/pulse/stone:<commit-sha>
ghcr.io/miguelcsx/pulse/web:<commit-sha>
```

No registry token is stored on disk as a repo secret. The workflow logs the remote Docker daemon into GHCR temporarily using the job `GITHUB_TOKEN`.

## Optional HTTPS with a domain

If you have a domain or subdomain:

1. Point an A record to the Lightsail public IP.
2. Set these in `.env`:

```sh
PULSE_HOST=pulse.example.com
ACME_EMAIL=you@example.com
CORS_ORIGINS=https://pulse.example.com
WS_ORIGINS=https://pulse.example.com
AUTH_COOKIE_SECURE=true
```

3. Replace the Caddyfile:

```sh
cp Caddyfile.https Caddyfile
docker compose up -d --build web stone
```

## Shutdown after the demo

```sh
docker compose down
```

To stop all AWS charges, delete the Lightsail instance from the AWS console.
