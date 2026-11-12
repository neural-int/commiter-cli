#!/usr/bin/env bash
set -euo pipefail
umask 077
if [[ $# != 2 ]]; then
  echo "usage: test-fail-closed.sh <llama.cpp source> <build directory>" >&2
  exit 2
fi
source_dir="$1"
build_dir="$2"
script_dir="$(cd "$(dirname "$0")" && pwd)"
[[ "$(uname -s)" == Darwin && "$(uname -m)" == arm64 ]]
[[ "$source_dir" == /* && "$build_dir" == /* ]]
llguidance_dir="$build_dir/llguidance/source/target/release"
/usr/bin/c++ -std=c++17 -O2 -DLLAMA_USE_LLGUIDANCE \
  -I"$source_dir/common" -I"$source_dir/include" -I"$source_dir/ggml/include" \
  -I"$source_dir/vendor" -I"$llguidance_dir" \
  "$script_dir/test-fail-closed.cpp" -o "$build_dir/bin/test-llguidance-fail-closed" \
  "$build_dir/common/libllama-common.a" "$build_dir/common/libllama-common-base.a" \
  "$llguidance_dir/libllguidance.a" "$build_dir/src/libllama.a" \
  "$build_dir/ggml/src/libggml.a" "$build_dir/ggml/src/libggml-cpu.a" \
  "$build_dir/ggml/src/ggml-blas/libggml-blas.a" \
  "$build_dir/ggml/src/ggml-metal/libggml-metal.a" "$build_dir/ggml/src/libggml-base.a" \
  "$build_dir/vendor/hash/libvendor-hash.a" -framework Accelerate \
  -framework Foundation -framework Metal -framework MetalKit
"$build_dir/bin/test-llguidance-fail-closed" normal
"$build_dir/bin/test-llguidance-fail-closed" normal-terminal
for mode in compile mask consume reset clone sticky terminal-reject; do
  status=0
  "$build_dir/bin/test-llguidance-fail-closed" "$mode" || status=$?
  if [[ "$status" != 1 ]]; then
    echo "FAIL: $mode must exit 1 before generation resumes; status=$status" >&2
    exit 1
  fi
  echo "PASS: $mode stopped the process before returning to generation"
done
