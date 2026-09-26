#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
version="${1:-0.0.0-preflight}"
archive="${2:-commiter_${version}_darwin_arm64.zip}"

if [[ "$(uname -s)" != Darwin || "$(uname -m)" != arm64 ]]; then
  echo "::error::Release preflight requires macOS on Apple Silicon"
  exit 1
fi

export RUNNER_TEMP="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
export MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-14.0}"
export CGO_ENABLED="${CGO_ENABLED:-1}"
export GOOS="${GOOS:-darwin}"
export GOARCH="${GOARCH:-arm64}"

mkdir -p "$repo_root/dist"
(
  cd "$repo_root"
  go build \
    -trimpath \
    -ldflags "-s -w -X github.com/natsuki0413/commiter-cli/internal/cli.Version=${version}" \
    -o dist/commiter \
    ./cmd/commiter
)

actual_version="$("$repo_root/dist/commiter" version)"
expected_version="commiter ${version}"
if [[ "$actual_version" != "$expected_version" ]]; then
  echo "::error::Version mismatch: expected '$expected_version', got '$actual_version'"
  exit 1
fi

archs="$(lipo -archs "$repo_root/dist/commiter")"
if [[ "$archs" != arm64 ]]; then
  echo "::error::Expected arm64 Mach-O, got: $archs"
  exit 1
fi
file "$repo_root/dist/commiter"

bash "$repo_root/.github/scripts/build-mlx-helper.sh"
bash "$repo_root/.github/scripts/build-release-package.sh" "$archive"
bash "$repo_root/.github/scripts/verify-release-package.sh" "$archive" false

echo "Release preflight passed: dist/$archive"
