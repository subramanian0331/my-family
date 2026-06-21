#!/usr/bin/env bash
# Fetch SMTP_PASSWORD from OCI Vault when OCI_SMTP_SECRET_OCID is set in .env.
# Docker Compose reads .env for all other variables — we never put SMTP_PASSWORD there.
set -euo pipefail

read_env_var() {
  local key="$1"
  local file="${2:-.env}"
  local line value
  line="$(grep -E "^${key}=" "$file" 2>/dev/null | tail -1 || true)"
  [[ -n "$line" ]] || return 1
  value="${line#*=}"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  printf '%s' "$value"
}

instance_region() {
  local region
  region="$(read_env_var OCI_REGION 2>/dev/null || true)"
  if [[ -n "$region" ]]; then
    echo "$region"
    return 0
  fi
  curl -sf -H "Authorization: Bearer Oracle" -L http://169.254.169.254/opc/v2/instance/ \
    | python3 -c 'import json,sys; print(json.load(sys.stdin)["canonicalRegionName"])'
}

fetch_smtp_password_from_oci() {
  unset SMTP_PASSWORD

  local secret_ocid
  secret_ocid="$(read_env_var OCI_SMTP_SECRET_OCID 2>/dev/null || true)"
  if [[ -z "$secret_ocid" ]]; then
    return 0
  fi

  if ! command -v oci &>/dev/null; then
    echo "WARN: OCI_SMTP_SECRET_OCID is set but oci CLI is missing; invite email disabled" >&2
    return 0
  fi

  local region
  if ! region="$(instance_region 2>/dev/null)"; then
    echo "WARN: could not detect OCI region; invite email disabled" >&2
    return 0
  fi

  local encoded
  if ! encoded="$(oci secrets secret-bundle get-secret-bundle \
    --secret-id "$secret_ocid" \
    --auth instance_principal \
    --region "$region" \
    --query 'data."secret-bundle-content".content' \
    --raw-output 2>/dev/null)"; then
    echo "WARN: failed to read SMTP secret from OCI Vault (check IAM dynamic group + policy)" >&2
    return 0
  fi

  SMTP_PASSWORD="$(printf '%s' "$encoded" | base64 -d)"
  if [[ -z "$SMTP_PASSWORD" ]]; then
    echo "WARN: OCI SMTP secret is empty; invite email disabled" >&2
    unset SMTP_PASSWORD
    return 0
  fi

  export SMTP_PASSWORD
}

prepare_runtime_env() {
  local root="${1:-.}"
  (cd "$root" && fetch_smtp_password_from_oci)
}