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

### Approach: GitHub Actions → SSH deploy

For a **single OCI VM**, deploying over SSH is the simplest reliable option:

- Push to `main` → GitHub Actions runs tests → SSH into the server → `git pull` → `docker compose up --build`
- No Docker Hub account required
- Builds happen on the server (fine for a small app on a 1–4 GB VM)

**Docker Hub** is worth adding later if you want pre-built images, rollbacks, or multiple servers. It is not required today.

### GitHub secrets

In **Settings → Secrets and variables → Actions**, add:

| Secret | Value |
|--------|--------|
| `OCI_HOST` | Server IP or hostname, e.g. `144.24.34.65` |
| `OCI_USER` | `ubuntu` |
| `OCI_SSH_KEY` | Private key contents (`~/.ssh/oracle-cloud`) |

### Manual deploy

```bash
ssh ubuntu@YOUR_HOST 'cd ~/family_tree && git pull && ./scripts/deploy.sh'
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