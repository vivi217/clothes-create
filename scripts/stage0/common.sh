#!/usr/bin/env bash

set -euo pipefail

log_info() {
  printf '[INFO] %s\n' "$1"
}

log_ok() {
  printf '[OK] %s\n' "$1"
}

log_warn() {
  printf '[WARN] %s\n' "$1"
}

log_error() {
  printf '[ERROR] %s\n' "$1" >&2
}

require_command() {
  local command_name="$1"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    log_error "Missing command: $command_name"
    return 1
  fi
  log_ok "Found command: $command_name"
}

run_python_check() {
  local script_path="$1"
  if command -v python3 >/dev/null 2>&1; then
    python3 "$script_path"
  else
    log_error 'python3 is required to run Python validation scripts.'
    return 1
  fi
}