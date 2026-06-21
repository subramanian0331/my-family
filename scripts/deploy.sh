#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "Missing .env — copy .env.example and configure secrets on the server first." >&2
  exit 1
fi

# shellcheck source=scripts/ensure-oci-cli.sh
source "$ROOT/scripts/ensure-oci-cli.sh"
# shellcheck source=scripts/fetch-smtp-secret.sh
source "$ROOT/scripts/fetch-smtp-secret.sh"

if [[ -n "${OCI_SMTP_SECRET_OCID:-}" ]]; then
  ensure_oci_cli || echo "WARN: OCI CLI install failed; invite email may be disabled" >&2
fi
prepare_runtime_env "$ROOT"

export IMAGE_TAG="${IMAGE_TAG:-latest}"

echo "==> Pulling images (tag: ${IMAGE_TAG})"
timeout 300 sudo docker compose pull api frontend

echo "==> Starting containers"
timeout 120 sudo docker compose up -d --remove-orphans

echo "==> Pruning unused images"
sudo docker image prune -f

echo "==> Status"
sudo docker compose ps