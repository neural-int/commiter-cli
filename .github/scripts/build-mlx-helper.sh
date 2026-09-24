#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
build_root="${RUNNER_TEMP:?}/commiter-mlx-xcode"
products="$build_root/Build/Products/Release"

(
  cd "$repo_root/mlx-helper"
  xcodebuild \
    -scheme CommiterMLXHelper \
    -configuration Release \
    -destination 'platform=macOS,arch=arm64' \
    -derivedDataPath "$build_root" \
    -clonedSourcePackagesDirPath "$build_root/SourcePackages" \
    -skipPackagePluginValidation \
    -skipMacroValidation \
    MACOSX_DEPLOYMENT_TARGET=14.0 \
    ARCHS=arm64 \
    ONLY_ACTIVE_ARCH=YES \
    CODE_SIGNING_ALLOWED=NO \
    build
)

helper="$products/commiter-mlx-helper"
test -x "$helper"
archs="$(lipo -archs "$helper")"
if [[ "$archs" != arm64 ]]; then
  echo "::error::Expected arm64 MLX helper, got: $archs"
  exit 1
fi

deployment_target="$(otool -l "$helper" | awk '
  $1 == "cmd" && $2 == "LC_BUILD_VERSION" { in_build = 1; next }
  in_build && $1 == "minos" { print $2; exit }
')"
if [[ "$deployment_target" != 14.0 ]]; then
  echo "::error::Expected MLX helper deployment target 14.0, got: ${deployment_target:-unknown}"
  exit 1
fi

unexpected_dependencies="$(otool -L "$helper" | awk '
  NR == 1 { next }
  {
    dependency = $1
    if (dependency !~ /^\/usr\/lib\// && dependency !~ /^\/System\/Library\//) {
      print dependency
    }
  }
')"
if [[ -n "$unexpected_dependencies" ]]; then
  echo "::error::MLX helper has dependencies outside macOS system libraries: $unexpected_dependencies"
  exit 1
fi

metallib="$(find "$products" -path '*/mlx-swift_Cmlx.bundle/*' -name default.metallib -type f -print -quit)"
if [[ -z "$metallib" || ! -s "$metallib" ]]; then
  echo "::error::Xcode did not produce mlx-swift_Cmlx.bundle/default.metallib"
  exit 1
fi

mkdir -p "$repo_root/dist"
cp "$helper" "$repo_root/dist/commiter-mlx-helper"
# mlx-swift 0.31.6 first looks for mlx.metallib beside the executable.
cp "$metallib" "$repo_root/dist/mlx.metallib"
