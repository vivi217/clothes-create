#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$SCRIPT_DIR/common.sh"

FAILED=0

run_check() {
  local label="$1"
  shift
  log_info "Running $label"
  if "$@"; then
    log_ok "$label passed"
  else
    log_error "$label failed"
    FAILED=1
  fi
}

run_check 'system check' bash "$SCRIPT_DIR/check_system.sh"
run_check 'pygarment check' python3 "$SCRIPT_DIR/check_pygarment.py"
run_check 'OBJ to GLB check' python3 "$SCRIPT_DIR/check_obj_to_glb.py"
run_check 'Seamly2D check' bash "$SCRIPT_DIR/check_seamly2d.sh"
run_check 'MinIO check' python3 "$SCRIPT_DIR/check_minio.py"
run_check 'Qwen API check' python3 "$SCRIPT_DIR/check_qwen_api.py"
run_check 'vision stack check' python3 "$SCRIPT_DIR/check_vision_stack.py"

if [[ $FAILED -ne 0 ]]; then
  log_error 'One or more stage 0 checks failed.'
  exit 1
fi

log_ok 'All stage 0 checks passed.'