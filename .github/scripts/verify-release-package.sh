#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
archive="${1:?usage: verify-release-package.sh <archive-name> [verify-signatures]}"
verify_signatures="${2:-false}"
runner_temp="${RUNNER_TEMP:-${TMPDIR:-/tmp}}"
archive_path="$repo_root/dist/$archive"

if [[ "$archive" == */* ]]; then
  echo "::error::Archive name must not contain a path: $archive"
  exit 1
fi
if [[ "$verify_signatures" != true && "$verify_signatures" != false ]]; then
  echo "::error::verify-signatures must be true or false"
  exit 1
fi

package_check="$(mktemp -d "$runner_temp/commiter-package-check.XXXXXX")"
unzip -q "$archive_path" -d "$package_check"
test -x "$package_check/bin/commiter"
test -x "$package_check/libexec/commiter-mlx-helper"
test -s "$package_check/libexec/mlx.metallib"

helper_entries="$(unzip -Z1 "$archive_path" | awk '/(^|\/)commiter-mlx-helper$/')"
if [[ "$helper_entries" != "libexec/commiter-mlx-helper" ]]; then
  echo "::error::Expected only the private libexec helper entry, got: ${helper_entries:-none}"
  exit 1
fi

if [[ "$verify_signatures" == true ]]; then
  codesign --verify --strict "$package_check/bin/commiter"
  codesign --verify --strict "$package_check/libexec/commiter-mlx-helper"
fi

config_home="$package_check/config"
state_home="$package_check/state"
doctor_output="$package_check/doctor.json"
mkdir -p "$config_home/commiter" "$state_home"
cat > "$config_home/commiter/config.toml" <<'EOF'
[llm]
backend = "mlx"
model = "owner/model"
model_revision = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
model_quantization = "4bit"
EOF

set +e
XDG_CONFIG_HOME="$config_home" XDG_STATE_HOME="$state_home" \
  "$package_check/bin/commiter" --json doctor > "$doctor_output"
doctor_status=$?
set -e
if [[ "$doctor_status" -eq 0 ]]; then
  echo "::error::Doctor unexpectedly accepted the absent smoke model"
  exit 1
fi

python3 - "$doctor_output" <<'PY'
import json
import sys

with open(sys.argv[1], encoding="utf-8") as stream:
    checks = json.load(stream)["doctor"]
if checks["mlx_helper"]["ok"] is not True or checks["mlx_model"]["ok"] is not False:
    raise SystemExit("packaged commiter did not resolve its private MLX helper")
PY

smoke_output="$(cd "$package_check" && "$package_check/libexec/commiter-mlx-helper" --smoke-metal)"
if [[ "$smoke_output" != metal_ok ]]; then
  echo "::error::Packaged MLX helper did not initialize Metal"
  exit 1
fi
