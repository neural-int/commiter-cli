#!/usr/bin/env bash
set -euo pipefail
umask 077

# Build the approved fixed revision with only the reviewed LLGuidance patch.
# CMake and Rust/Cargo must already be installed/approved by the operator.
if [[ $# != 3 ]]; then
  echo "usage: build.sh <llama.cpp source> <build directory> <cmake executable>" >&2
  exit 2
fi
source_dir="$1"
build_dir="$2"
cmake_executable="$3"
revision=80d2c0c2c56e4a1bbf74be98f64b99927bf230bc
llguidance_revision=d795912fedc7d393de740177ea9ea761e7905774
patch_sha256=68aa603b9c43e96975f95a0ef506972cc7be35d7eba35f245207ea268942661a
patched_blob=0c4ea16817eff73702ca508760bb6113a0d1005c
build_commit="$revision+commiter-llg-fail-closed-v1"
script_dir="$(cd "$(dirname "$0")" && pwd)"
patch_file="$script_dir/patches/0001-llguidance-fail-closed.patch"

[[ "$(uname -s)" == Darwin && "$(uname -m)" == arm64 ]]
[[ "$(git -C "$source_dir" rev-parse HEAD)" == "$revision" ]]
[[ "$source_dir" == /* && "$build_dir" == /* && "$cmake_executable" == /* ]]
[[ "$(shasum -a 256 "$patch_file" | cut -d ' ' -f 1)" == "$patch_sha256" ]]
if [[ -z "$(git -C "$source_dir" status --porcelain --untracked-files=all)" ]]; then
  git -C "$source_dir" apply --check --whitespace=error "$patch_file"
  git -C "$source_dir" apply --whitespace=error "$patch_file"
fi
verify_source() {
  [[ "$(git -C "$source_dir" status --porcelain --untracked-files=all)" == ' M common/llguidance.cpp' ]]
  [[ "$(git -C "$source_dir" hash-object --no-filters common/llguidance.cpp)" == "$patched_blob" ]]
}
verify_source
export CARGO_HOME="$build_dir/cargo"
export CARGO_BUILD_JOBS=4

"$cmake_executable" -S "$source_dir" -B "$build_dir" \
  -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF \
  -DLLAMA_BUILD_NUMBER=1 -DLLAMA_BUILD_COMMIT="$build_commit" \
  -DLLAMA_LLGUIDANCE=ON -DGGML_METAL=ON -DGGML_METAL_EMBED_LIBRARY=ON \
  -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_UI=OFF \
  -DLLAMA_OPENSSL=OFF
"$cmake_executable" --build "$build_dir" --target llama-server -j 4
[[ "$(git -C "$build_dir/llguidance/source" rev-parse HEAD)" == "$llguidance_revision" ]]
verify_source
[[ -z "$(git -C "$build_dir/llguidance/source" status --porcelain)" ]]
"$build_dir/bin/llama-server" --version
shasum -a 256 "$build_dir/bin/llama-server" "$build_dir/llguidance/source/Cargo.lock"
echo "base_revision=$revision patch_sha256=$patch_sha256"
