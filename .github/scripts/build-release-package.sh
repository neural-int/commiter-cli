#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
archive="${1:?usage: build-release-package.sh <archive-name>}"
runner_temp="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"

if [[ "$archive" == */* ]]; then
  echo "::error::Archive name must not contain a path: $archive"
  exit 1
fi

test -x "$repo_root/dist/commiter"
test -x "$repo_root/dist/commiter-mlx-helper"
test -s "$repo_root/dist/mlx.metallib"
test -s "$repo_root/LICENSE"

package_dir="$(mktemp -d "$runner_temp/commiter-package.XXXXXX")"
mkdir -p "$package_dir/bin" "$package_dir/libexec"
cp "$repo_root/dist/commiter" "$package_dir/bin/commiter"
cp "$repo_root/dist/commiter-mlx-helper" "$package_dir/libexec/commiter-mlx-helper"
cp "$repo_root/dist/mlx.metallib" "$package_dir/libexec/mlx.metallib"
cp "$repo_root/LICENSE" "$package_dir/LICENSE"

archive_tmp="$package_dir/$archive"
(
  cd "$package_dir"
  zip -X -q -r "$archive_tmp" bin libexec LICENSE
)
mv "$archive_tmp" "$repo_root/dist/$archive"
