#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "Missing .env — copy .env.example and configure secrets on the server first." >&2
  exit 1
fi

echo "==> Building and starting containers"
sudo docker compose up --build -d

echo "==> Pruning unused images"
sudo docker image prune -f

echo "==> Status"
sudo docker compose ps