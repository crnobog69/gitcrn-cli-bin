#!/usr/bin/env bash
set -euo pipefail

APP_NAME="gitcrn"
VERSION="${VERSION:-latest}"
PREFIX="${PREFIX:-}"
SERVER_URL="${SERVER_URL:-https://100.91.132.35}"
OWNER="${OWNER:-vltc}"
REPO="${REPO:-gitcrn-cli}"
INSECURE=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)
      if [[ $# -lt 2 ]]; then
        echo "Error: --prefix requires a value." >&2
        exit 1
      fi
      PREFIX="$2"
      shift 2
      ;;
    --version)
      if [[ $# -lt 2 ]]; then
        echo "Error: --version requires a value." >&2
        exit 1
      fi
      VERSION="$2"
      shift 2
      ;;
    --server-url)
      if [[ $# -lt 2 ]]; then
        echo "Error: --server-url requires a value." >&2
        exit 1
      fi
      SERVER_URL="$2"
      shift 2
      ;;
    --owner)
      if [[ $# -lt 2 ]]; then
        echo "Error: --owner requires a value." >&2
        exit 1
      fi
      OWNER="$2"
      shift 2
      ;;
    --repo)
      if [[ $# -lt 2 ]]; then
        echo "Error: --repo requires a value." >&2
        exit 1
      fi
      REPO="$2"
      shift 2
      ;;
    --insecure)
      INSECURE=1
      shift
      ;;
    -h|--help)
      cat <<EOF
Usage: ./scripts/install.sh [options]

Options:
  --version <value>      Release tag (default: latest)
  --prefix <path>        Install directory (default: /usr/local/bin or ~/.local/bin)
  --server-url <url>     Gitea base URL (default: https://100.91.132.35)
  --owner <owner>        Repo owner (default: vltc)
  --repo <repo>          Repo name (default: gitcrn-cli)
  --insecure             Disable TLS certificate verification
EOF
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "${SERVER_URL}" || -z "${OWNER}" || -z "${REPO}" ]]; then
  echo "Error: server URL, owner and repo must be set." >&2
  exit 1
fi

if [[ -z "${PREFIX}" ]]; then
  if [[ -w "/usr/local/bin" ]]; then
    PREFIX="/usr/local/bin"
  else
    PREFIX="${HOME}/.local/bin"
  fi
fi

mkdir -p "${PREFIX}"

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *)
      echo "Error: unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

download_file() {
  local url="$1"
  local output="$2"
  if command -v curl >/dev/null 2>&1; then
    local flags=(-fL --retry 2)
    if [[ "${INSECURE}" -eq 1 ]]; then
      flags+=(-k)
    fi
    curl "${flags[@]}" "$url" -o "$output"
    return
  fi
  if command -v wget >/dev/null 2>&1; then
    local flags=()
    if [[ "${INSECURE}" -eq 1 ]]; then
      flags+=(--no-check-certificate)
    fi
    wget "${flags[@]}" -O "$output" "$url"
    return
  fi
  echo "Error: neither curl nor wget is installed." >&2
  exit 1
}

resolve_latest_tag() {
  local api_url="${SERVER_URL%/}/api/v1/repos/${OWNER}/${REPO}/releases/latest"
  local tmp_json
  tmp_json="$(mktemp)"
  download_file "$api_url" "$tmp_json"

  local tag=""
  if command -v jq >/dev/null 2>&1; then
    tag="$(jq -r '.tag_name // empty' "$tmp_json")"
  else
    tag="$(tr -d '\n' < "$tmp_json" | sed -n 's/.*"tag_name":"\([^"]*\)".*/\1/p')"
  fi

  rm -f "$tmp_json"
  if [[ -z "${tag}" || "${tag}" == "null" ]]; then
    echo "Error: could not resolve latest release tag from ${api_url}" >&2
    exit 1
  fi
  echo "$tag"
}

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
if [[ "${OS}" != "linux" ]]; then
  echo "Error: install.sh currently supports Linux only. Use install.ps1 on Windows." >&2
  exit 1
fi

ARCH="$(detect_arch)"
if [[ "${VERSION}" == "latest" ]]; then
  VERSION="$(resolve_latest_tag)"
fi

ASSET_NAME="${APP_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="${SERVER_URL%/}/${OWNER}/${REPO}/releases/download/${VERSION}/${ASSET_NAME}"

TMP_BIN="$(mktemp "${APP_NAME}.XXXXXX")"
cleanup() {
  rm -f "${TMP_BIN}"
}
trap cleanup EXIT

echo "Downloading ${DOWNLOAD_URL}..."
download_file "${DOWNLOAD_URL}" "${TMP_BIN}"

install -m 0755 "${TMP_BIN}" "${PREFIX}/${APP_NAME}"
echo "Installed: ${PREFIX}/${APP_NAME}"

case ":${PATH}:" in
  *":${PREFIX}:"*) ;;
  *)
    echo "Note: ${PREFIX} is not in PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\"${PREFIX}:\$PATH\""
    ;;
esac
