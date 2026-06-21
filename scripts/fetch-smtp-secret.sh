#!/usr/bin/env bash
# Load .env (except SMTP_PASSWORD) and optionally fetch SMTP_PASSWORD from OCI Vault.
set -euo pipefail

load_env_without_smtp_password() {
  local env_file="${1:-.env}"
  if [[ ! -f "$env_file" ]]; then
    return 0
  fi
  set -a
  # shellcheck disable=SC1090
  source <(grep -Ev '^(SMTP_PASSWORD|#|$)' "$env_file")
  set +a
}

instance_region() {
  if [[ -n "${OCI_REGION:-}" ]]; then
    echo "$OCI_REGION"
    return 0
  fi
  curl -sf -H "Authorization: Bearer Oracle" -L http://169.254.169.254/opc/v2/instance/ \
    | python3 -c 'import json,sys; print(json.load(sys.stdin)["canonicalRegionName"])'
}

fetch_smtp_password_from_oci() {
  unset SMTP_PASSWORD

  if [[ -z "${OCI_SMTP_SECRET_OCID:-}" ]]; then
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
    --secret-id "$OCI_SMTP_SECRET_OCID" \
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
  load_env_without_smtp_password "$root/.env"
  fetch_smtp_password_from_oci
}