#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
VERSION="${VERSION:-}"
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/dist}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      if [[ $# -lt 2 ]]; then
        echo "Error: --version requires a value." >&2
        exit 1
      fi
      VERSION="$2"
      shift 2
      ;;
    --out-dir)
      if [[ $# -lt 2 ]]; then
        echo "Error: --out-dir requires a value." >&2
        exit 1
      fi
      OUT_DIR="$2"
      shift 2
      ;;
    -h|--help)
      cat <<EOF
Usage: ./scripts/release.sh --version <tag> [--out-dir <path>]
EOF
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

if [[ -z "${VERSION}" ]]; then
  echo "Error: --version is required (example: v0.1.0)." >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Error: Go is not installed or not in PATH." >&2
  exit 1
fi

mkdir -p "${OUT_DIR}"

build_one() {
  local goos="$1"
  local goarch="$2"
  local suffix=""
  if [[ "${goos}" == "windows" ]]; then
    suffix=".exe"
  fi
  local output="${OUT_DIR}/gitcrn-${goos}-${goarch}${suffix}"

  echo "Building ${output}"
  (
    cd "${REPO_ROOT}"
    GOOS="${goos}" GOARCH="${goarch}" \
      go build -ldflags "-s -w -X main.version=${VERSION}" -o "${output}" ./cmd/gitcrn
  )
}

build_one linux amd64
build_one linux arm64
build_one windows amd64
build_one windows arm64

checksums_file="${OUT_DIR}/checksums.txt"
if command -v sha256sum >/dev/null 2>&1; then
  (
    cd "${OUT_DIR}"
    sha256sum gitcrn-* > "checksums.txt"
  )
elif command -v shasum >/dev/null 2>&1; then
  (
    cd "${OUT_DIR}"
    shasum -a 256 gitcrn-* > "checksums.txt"
  )
else
  echo "Warning: sha256sum/shasum not found, checksums.txt was not generated." >&2
fi

echo "Release assets are ready in: ${OUT_DIR}"
