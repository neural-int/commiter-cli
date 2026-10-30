#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
metallib="$repo_root/dist/mlx.metallib"
if [[ ! -s "$metallib" ]]; then
  echo "::error::Run build-mlx-helper.sh before the Swift tests (missing dist/mlx.metallib)"
  exit 1
fi

# Build the tests without running them: SwiftPM does not compile MLX's Metal
# kernels. Use the library produced from the locked dependency by Xcode.
swift_flags=(--package-path "$repo_root/mlx-helper" -c release --disable-automatic-resolution -Xswiftc -enable-testing "$@")
swift build "${swift_flags[@]}" --build-tests
test_products="$(swift build "${swift_flags[@]}" --show-bin-path)"
cp "$metallib" "$test_products/mlx.metallib"
swift test "${swift_flags[@]}" --skip-build

# Exercise Swift's real early-error wire responses, without model inference.
COMMITER_MLX_TEST_HELPER="$repo_root/dist/commiter-mlx-helper" \
  go -C "$repo_root" test ./internal/mlx -run '^TestRealHelperPreservesProfileOnEarlyFailure$' -count=1
