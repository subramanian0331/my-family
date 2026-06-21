#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "Missing .env — copy .env.example and configure secrets on the server first." >&2
  exit 1
fi

export IMAGE_TAG="${IMAGE_TAG:-latest}"

echo "==> Pulling images (tag: ${IMAGE_TAG})"
sudo docker compose pull api frontend

echo "==> Starting containers"
sudo docker compose up -d

echo "==> Pruning unused images"
sudo docker image prune -f

echo "==> Status"
sudo docker compose ps