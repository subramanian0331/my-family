#!/usr/bin/env bash
# One-time setup: OCI Vault secret + dynamic group + IAM policy for SMTP password.
# Run from your laptop (with OCI CLI configured as an admin). Password is read locally
# with read -s and sent straight to OCI — never written to .env or git.
set -euo pipefail

INSTANCE_OCID="${INSTANCE_OCID:-ocid1.instance.oc1.phx.anyhqljtvbtb7jqcip7fh4lbdzqtjzy2t7ttk5cpjy4tuywmu6uuv3foqfia}"
TENANCY_OCID="${TENANCY_OCID:-ocid1.tenancy.oc1..aaaaaaaakf6qronl4rkydogjallk2sijqnhontb37ccv7uzswfldcwipuhma}"
REGION="${OCI_REGION:-us-phoenix-1}"
COMPARTMENT_OCID="${COMPARTMENT_OCID:-$TENANCY_OCID}"
DG_NAME="${DG_NAME:-family-tree-instance-dg}"
POLICY_NAME="${POLICY_NAME:-family-tree-smtp-secret-read}"
VAULT_NAME="${VAULT_NAME:-family-tree-vault}"
KEY_NAME="${KEY_NAME:-family-tree-master-key}"
SECRET_NAME="${SECRET_NAME:-family-tree-smtp-password}"

usage() {
  cat <<EOF
Usage: $0 [--print-iam | --create]

  --print-iam   Print Console steps for IAM (dynamic group + policy)
  --create      Create vault/secret via OCI CLI (prompts for SMTP password)

Environment overrides: INSTANCE_OCID TENANCY_OCID OCI_REGION COMPARTMENT_OCID

After --create, add the printed OCI_SMTP_SECRET_OCID to the server .env (not SMTP_PASSWORD).
EOF
}

print_iam_instructions() {
  cat <<EOF
=== OCI Console: IAM (one time) ===

1. Identity & Security → Domains → Default → Dynamic groups → Create
   Name: ${DG_NAME}
   Matching rule:
     instance.id = '${INSTANCE_OCID}'

2. Identity & Security → Policies → Create policy (root compartment / tenancy)
   Name: ${POLICY_NAME}
   Statement:
     Allow dynamic-group ${DG_NAME} to read secret-bundles in tenancy

=== OCI Console: Secret (if not using --create) ===

1. Security → Vault → Create vault (type: Default) in your compartment
2. Create master encryption key (AES, software-protected)
3. Secrets → Create secret
   Name: ${SECRET_NAME}
   Secret type: Base64
   Value: your SMTP password or mail API key (paste in Console only)
4. Copy the secret OCID → set OCI_SMTP_SECRET_OCID in server ~/family_tree/.env

EOF
}

require_oci_admin() {
  if ! command -v oci &>/dev/null; then
    echo "oci CLI not found. Install: https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/cliinstall.htm" >&2
    exit 1
  fi
  if ! oci iam compartment get --compartment-id "$COMPARTMENT_OCID" --region "$REGION" &>/dev/null; then
    echo "OCI CLI is not authenticated or lacks permission. Fix ~/.oci/config (user OCID + API key or session)." >&2
    exit 1
  fi
}

find_or_create_vault() {
  local vault_id
  vault_id="$(oci kms management vault list \
    --compartment-id "$COMPARTMENT_OCID" \
    --region "$REGION" \
    --query "data[?\"display-name\"=='${VAULT_NAME}'].id | [0]" \
    --raw-output 2>/dev/null || true)"

  if [[ -n "$vault_id" && "$vault_id" != "null" ]]; then
    echo "$vault_id"
    return 0
  fi

  echo "==> Creating vault: ${VAULT_NAME}" >&2
  oci kms management vault create \
    --compartment-id "$COMPARTMENT_OCID" \
    --display-name "$VAULT_NAME" \
    --vault-type DEFAULT \
    --region "$REGION" \
    --query 'data.id' \
    --raw-output
}

find_or_create_key() {
  local vault_id="$1"
  local key_id
  key_id="$(oci kms management key list \
    --compartment-id "$COMPARTMENT_OCID" \
    --vault-id "$vault_id" \
    --region "$REGION" \
    --query "data[?\"display-name\"=='${KEY_NAME}'].id | [0]" \
    --raw-output 2>/dev/null || true)"

  if [[ -n "$key_id" && "$key_id" != "null" ]]; then
    echo "$key_id"
    return 0
  fi

  echo "==> Creating master encryption key: ${KEY_NAME}" >&2
  oci kms management key create \
    --compartment-id "$COMPARTMENT_OCID" \
    --display-name "$KEY_NAME" \
    --key-shape '{"algorithm":"AES","length":32}' \
    --protection-mode SOFTWARE \
    --vault-id "$vault_id" \
    --region "$REGION" \
    --query 'data.id' \
    --raw-output
}

create_secret() {
  require_oci_admin

  local vault_id key_id secret_ocid encoded
  vault_id="$(find_or_create_vault)"
  key_id="$(find_or_create_key "$vault_id")"

  echo -n "SMTP password or mail API key (input hidden): "
  read -rs smtp_pass
  echo
  if [[ -z "$smtp_pass" ]]; then
    echo "Empty password — aborting." >&2
    exit 1
  fi

  encoded="$(printf '%s' "$smtp_pass" | base64 | tr -d '\n')"
  unset smtp_pass

  echo "==> Creating secret: ${SECRET_NAME}" >&2
  secret_ocid="$(oci vault secret create-base64 \
    --compartment-id "$COMPARTMENT_OCID" \
    --vault-id "$vault_id" \
    --key-id "$key_id" \
    --secret-name "$SECRET_NAME" \
    --secret-content-content "$encoded" \
    --region "$REGION" \
    --query 'data.id' \
    --raw-output)"
  unset encoded

  cat <<EOF

=== Done ===

Add to server ~/family_tree/.env (SSH in — do not commit this file):

OCI_SMTP_SECRET_OCID=${secret_ocid}
OCI_REGION=${REGION}

Remove any SMTP_PASSWORD= line from .env if present.

Then deploy:
  make deploy

Ensure IAM dynamic group + policy exist (run: $0 --print-iam)
EOF
}

main() {
  local mode="${1:-}"
  case "$mode" in
    --print-iam) print_iam_instructions ;;
    --create) create_secret ;;
    -h|--help|"") usage ;;
    *) echo "Unknown option: $mode" >&2; usage; exit 1 ;;
  esac
}

main "$@"