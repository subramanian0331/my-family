#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

TAG="${IMAGE_TAG:-${1:-}}"
if [[ -z "$TAG" ]]; then
  echo "Usage: IMAGE_TAG=<git-sha> $0" >&2
  echo "       $0 <git-sha>" >&2
  exit 1
fi

export IMAGE_TAG="$TAG"

if [[ ! -f .env ]]; then
  echo "Missing .env — copy .env.example and configure secrets on the server first." >&2
  exit 1
fi

echo "==> Rolling back to image tag: ${IMAGE_TAG}"
sudo docker compose pull api frontend
sudo docker compose up -d
sudo docker image prune -f
sudo docker compose ps