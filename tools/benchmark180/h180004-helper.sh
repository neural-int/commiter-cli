#!/bin/sh
# H180-004: pin the measured 8-bit KV configuration for benchmark-cache runs.
set -eu
if [ "$#" -ne 1 ] || [ "$1" != "--benchmark-cache" ]; then
    echo "H180-004 requires --benchmark-cache" >&2
    exit 64
fi
native_helper=${COMMITER_180_H180004_NATIVE_HELPER:-}
if [ -z "$native_helper" ]; then
    echo "Set COMMITER_180_H180004_NATIVE_HELPER to the built commiter-mlx-helper binary; see tools/benchmark180/README.md" >&2
    exit 78
fi
if [ ! -x "$native_helper" ]; then
    echo "H180-004 native helper is missing or not executable: $native_helper" >&2
    exit 69
fi
export COMMITER_180_KV_BITS=8
exec "$native_helper" "$@"
