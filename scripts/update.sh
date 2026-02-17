#!/usr/bin/env bash
set -euo pipefail

INSTALL_SCRIPT_URL="${GITCRN_INSTALL_SCRIPT_URL:-https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/install.sh}"
script_path="${BASH_SOURCE[0]-}"

if [[ -n "${script_path}" && -f "${script_path}" ]]; then
  script_dir="$(cd "$(dirname "${script_path}")" && pwd)"
  if [[ -f "${script_dir}/install.sh" ]]; then
    "${script_dir}/install.sh" --version latest "$@"
    exit 0
  fi
fi

if command -v curl >/dev/null 2>&1; then
  curl -fsSL "${INSTALL_SCRIPT_URL}" | bash -s -- --version latest "$@"
  exit 0
fi

if command -v wget >/dev/null 2>&1; then
  wget -qO- "${INSTALL_SCRIPT_URL}" | bash -s -- --version latest "$@"
  exit 0
fi

echo "Error: neither curl nor wget is installed." >&2
exit 1
