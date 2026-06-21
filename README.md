# Family Tree

A collaborative family history app: interactive tree view, Google sign-in, photos, GEDCOM import/export, and multi-family support.

**Live site:** https://myfamily.hopto.org

## Stack

| Layer | Tech |
|-------|------|
| API | Go, PostgreSQL |
| UI | React, TypeScript, Tailwind, Vite |
| Auth | Google OAuth 2.0, JWT |
| Deploy | Docker Compose, Caddy (HTTPS) |

## Local development

### Prerequisites

- Docker Desktop
- Node.js 22+ (frontend only)
- Go 1.26+ (backend only)

### Quick start

```bash
cp .env.example .env
# Edit .env — set GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, JWT_SECRET

make up
```

Open http://localhost

### Useful commands

```bash
make up          # start all services
make down        # stop services
make test        # run Go tests
make frontend    # build frontend locally
make wire        # regenerate Wire DI
```

### Google OAuth (local)

1. [Google Cloud Console](https://console.cloud.google.com/apis/credentials) → OAuth client (Web)
2. **Authorized redirect URI:** `http://localhost/api/auth/google/callback`
3. Set `FRONTEND_URL=http://localhost` in `.env`

## Production (Oracle Cloud)

Single VM deployment with Docker Compose. Caddy terminates HTTPS and proxies to the API and static frontend.

```
Internet → Caddy (:443) → frontend (static) + api (:8080) → postgres
```

### First-time server setup

```bash
ssh ubuntu@YOUR_HOST

# Docker
sudo apt-get update && sudo apt-get install -y docker.io docker-compose-plugin git
sudo systemctl enable --now docker
sudo usermod -aG docker ubuntu

# App
git clone git@github.com:subramanian0331/my-family.git family_tree
cd family_tree
cp .env.example .env
nano .env   # production values — never commit this file

# Oracle Ubuntu images block ports except 22 — open 80/443 in iptables too:
sudo sed -i '/--dport 22 -j ACCEPT/a -A INPUT -p tcp -m state --state NEW -m tcp --dport 80 -j ACCEPT\n-A INPUT -p tcp -m state --state NEW -m tcp --dport 443 -j ACCEPT' /etc/iptables/rules.v4
sudo iptables-restore < /etc/iptables/rules.v4

./scripts/deploy.sh
```

Also open **TCP 80 and 443** in the Oracle **Security List** and any attached **Network Security Group**.

### Environment variables

| Variable | Description |
|----------|-------------|
| `POSTGRES_PASSWORD` | Database password |
| `JWT_SECRET` | Long random string for session tokens |
| `FRONTEND_URL` | Public site URL, e.g. `https://myfamily.hopto.org` |
| `GOOGLE_CLIENT_ID` | OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | OAuth client secret |
| `SITE_ADMIN_EMAIL` | Google email that becomes site admin |

## CI/CD

### Approach: Docker Hub + SSH pull

Images are built on fast GitHub Actions runners and pushed to [Docker Hub](https://hub.docker.com/repositories/subni9). The OCI server only **pulls** images — deploys take ~1–3 minutes instead of building on a 1 GB VM.

```
push to main → tests → build & push images → SSH → git pull → docker compose pull → up -d
```

| Image | Repository |
|-------|------------|
| API | `subni9/family-tree-api:latest` |
| Frontend | `subni9/family-tree-frontend:latest` |

**Local dev** still uses `docker compose up --build` (builds from source).

### GitHub secrets

**Settings → Secrets and variables → Actions:**

| Secret | Value |
|--------|--------|
| `DOCKERHUB_USERNAME` | `subni9` |
| `DOCKERHUB_TOKEN` | [Access token](https://hub.docker.com/settings/security) (not your password) |
| `OCI_HOST` | Server IP or hostname, e.g. `144.24.34.65` |
| `OCI_USER` | `ubuntu` |
| `OCI_SSH_KEY` | Private key contents (`~/.ssh/oracle-cloud`) |

Create empty repos on Docker Hub named `family-tree-api` and `family-tree-frontend` under **subni9**, or let the first CI push create them.

### Manual deploy & rollback (Makefile)

```bash
make deploy                              # latest main + :latest images
make tags                                # list recent SHAs
make rollback TAG=a11bd98                # deploy a previous image tag
make remote-status                       # check server containers
```

Override host/key if needed:

```bash
make deploy OCI_HOST=144.24.34.65 OCI_SSH_KEY=~/.ssh/oracle-cloud
```

### On the server directly

```bash
./scripts/deploy.sh
IMAGE_TAG=a11bd98 ./scripts/rollback.sh a11bd98
```

## Features

- **Tree view** — pan, zoom, drag-to-link spouse/child relationships
- **Add members in tree view** — `+ Add member` with optional parent/spouse/child link
- **People tab** — bulk add, search, person details
- **Families** — multiple family trees, invites, roles (owner/editor/viewer)
- **Photos** — per-person uploads
- **GEDCOM** — import and export
- **Married-in** — people who married into a family (shown with distinct styling)

## Project layout

```
backend/          Go API
frontend/         React UI
migrations/       Postgres schema (applied on first DB start)
deploy/Caddyfile  Reverse proxy + HTTPS
scripts/deploy.sh Production deploy script
docker-compose.yml
```

## License

Private family project.