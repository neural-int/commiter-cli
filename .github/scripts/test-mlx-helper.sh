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
cmp "$metallib" "$test_products/mlx.metallib"
echo "MLX test library: $test_products/mlx.metallib"
# SwiftPM's macOS test runner loads Mach-O bundles from Contents/MacOS.
# MLX uses dladdr(current_binary_dir), so the Products root is not sufficient
# when its code is statically linked into one of these test bundle binaries.
while IFS= read -r -d '' test_binary; do
  library_dir="$(dirname "$test_binary")"
  cp "$metallib" "$library_dir/mlx.metallib"
  cmp "$metallib" "$library_dir/mlx.metallib"
  echo "MLX test library: $library_dir/mlx.metallib"
done < <(find "$test_products" -type f -path '*.xctest/Contents/MacOS/*' -perm -111 -print0)
swift test "${swift_flags[@]}" --skip-build

# Exercise Swift's real early-error wire responses, without model inference.
COMMITER_MLX_TEST_HELPER="$repo_root/dist/commiter-mlx-helper" \
  go -C "$repo_root" test ./internal/mlx -run '^TestRealHelperPreservesProfileOnEarlyFailure$' -count=1
