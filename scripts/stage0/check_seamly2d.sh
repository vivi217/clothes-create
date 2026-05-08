#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
. "$SCRIPT_DIR/common.sh"

if ! require_command seamly2d-cli; then
  exit 1
fi

if seamly2d-cli --help >/tmp/seamly2d-help.txt 2>&1; then
  log_ok 'seamly2d-cli --help executed successfully.'
  head -n 5 /tmp/seamly2d-help.txt || true
  exit 0
fi

log_error 'seamly2d-cli exists but failed to execute --help.'
exit 1