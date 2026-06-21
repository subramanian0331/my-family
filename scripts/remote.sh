#!/usr/bin/env bash
# Run a command on the OCI host. Override via env: OCI_HOST, OCI_USER, OCI_SSH_KEY, OCI_APP_DIR
set -euo pipefail

OCI_HOST="${OCI_HOST:-144.24.34.65}"
OCI_USER="${OCI_USER:-ubuntu}"
OCI_SSH_KEY="${OCI_SSH_KEY:-$HOME/.ssh/oracle-cloud}"
OCI_APP_DIR="${OCI_APP_DIR:-family_tree}"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 <remote command>" >&2
  exit 1
fi

if [[ ! -f "$OCI_SSH_KEY" ]]; then
  echo "SSH key not found: $OCI_SSH_KEY" >&2
  exit 1
fi

exec ssh \
  -i "$OCI_SSH_KEY" \
  -o StrictHostKeyChecking=accept-new \
  -o ConnectTimeout=15 \
  "${OCI_USER}@${OCI_HOST}" \
  "$@"