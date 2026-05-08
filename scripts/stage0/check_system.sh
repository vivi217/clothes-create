#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$SCRIPT_DIR/common.sh"

FAILED=0

check_command() {
  local command_name="$1"
  if ! require_command "$command_name"; then
    FAILED=1
  fi
}

check_version_contains() {
  local label="$1"
  local actual="$2"
  local expected_fragment="$3"
  if [[ "$actual" == *"$expected_fragment"* ]]; then
    log_ok "$label version matches expected fragment: $expected_fragment"
  else
    log_warn "$label version does not contain expected fragment '$expected_fragment': $actual"
  fi
}

log_info 'Checking required system commands.'
check_command git
check_command docker
check_command python3
check_command pip3
check_command node
check_command npm
check_command go
check_command protoc
check_command kubectl

if command -v docker >/dev/null 2>&1; then
  if docker compose version >/dev/null 2>&1; then
    log_ok 'Found Docker Compose via docker compose.'
  elif command -v docker-compose >/dev/null 2>&1; then
    log_ok 'Found Docker Compose via docker-compose.'
  else
    log_error 'Missing Docker Compose.'
    FAILED=1
  fi
fi

if command -v python3 >/dev/null 2>&1; then
  PYTHON_VERSION=$(python3 --version 2>&1)
  check_version_contains 'Python' "$PYTHON_VERSION" '3.11'
fi

if command -v go >/dev/null 2>&1; then
  GO_VERSION=$(go version 2>&1)
  check_version_contains 'Go' "$GO_VERSION" 'go1.22'
fi

if command -v node >/dev/null 2>&1; then
  NODE_VERSION=$(node --version 2>&1)
  log_info "Node version: $NODE_VERSION"
fi

if command -v npm >/dev/null 2>&1; then
  NPM_VERSION=$(npm --version 2>&1)
  log_info "npm version: $NPM_VERSION"
fi

if [[ $FAILED -ne 0 ]]; then
  log_error 'System validation failed.'
  exit 1
fi

log_ok 'System validation finished.'