#!/usr/bin/env bash
# Ensure the OCI CLI is available (needed to fetch secrets via instance principal).
set -euo pipefail

ensure_oci_cli() {
  if command -v oci &>/dev/null; then
    return 0
  fi

  echo "==> Installing OCI CLI"
  local install_dir="${OCI_CLI_INSTALL_DIR:-$HOME/bin}"
  mkdir -p "$install_dir"

  bash -c "$(curl -fsSL https://raw.githubusercontent.com/oracle/oci-cli/master/scripts/install/install.sh)" -- \
    --accept-all-defaults \
    --exec-dir "$install_dir" \
    --install-dir "${OCI_CLI_HOME:-$HOME/lib/oracle-cli}"

  export PATH="$install_dir:$PATH"
  if ! command -v oci &>/dev/null; then
    echo "OCI CLI install failed — add $install_dir to PATH" >&2
    return 1
  fi
}