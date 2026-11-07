#!/usr/bin/env bash
set -euo pipefail
umask 077

# Build the explicitly requested experimental revision, without source patches.
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

[[ "$(uname -s)" == Darwin && "$(uname -m)" == arm64 ]]
[[ "$(git -C "$source_dir" rev-parse HEAD)" == "$revision" ]]
[[ -z "$(git -C "$source_dir" status --porcelain)" ]]
[[ "$source_dir" == /* && "$build_dir" == /* && "$cmake_executable" == /* ]]
export CARGO_HOME="$build_dir/cargo"
export CARGO_BUILD_JOBS=4

"$cmake_executable" -S "$source_dir" -B "$build_dir" \
  -DCMAKE_BUILD_TYPE=Release -DBUILD_SHARED_LIBS=OFF \
  -DLLAMA_LLGUIDANCE=ON -DGGML_METAL=ON -DGGML_METAL_EMBED_LIBRARY=ON \
  -DLLAMA_BUILD_TESTS=OFF -DLLAMA_BUILD_EXAMPLES=OFF -DLLAMA_BUILD_UI=OFF \
  -DLLAMA_OPENSSL=OFF
"$cmake_executable" --build "$build_dir" --target llama-server -j 4
[[ "$(git -C "$build_dir/llguidance/source" rev-parse HEAD)" == "$llguidance_revision" ]]
[[ -z "$(git -C "$source_dir" status --porcelain)" ]]
[[ -z "$(git -C "$build_dir/llguidance/source" status --porcelain)" ]]
"$build_dir/bin/llama-server" --version
shasum -a 256 "$build_dir/bin/llama-server" "$build_dir/llguidance/source/Cargo.lock"
