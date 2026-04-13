#!/usr/bin/env bash
# agent-status.sh — Get detailed status of an agent session.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-status.sh <run-id> [--no-json]

Get detailed status for an agent session.

Arguments:
  run-id      Harness run ID

Options:
  --no-json   Output the default kocao text format instead of JSON
  --help      Show this help
EOF
}

require_commands kocao

run_id=""
json_out=true
while (($#)); do
  case "$1" in
    --no-json)
      json_out=false
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    -* )
      usage_error "unknown flag: $1"
      ;;
    *)
      if [[ -n "$run_id" ]]; then
        usage_error "unexpected argument: $1"
      fi
      run_id="$1"
      shift
      ;;
  esac
done

require_nonempty "$run_id" "run-id"

cmd=(kocao agent status "$run_id")
if [[ "$json_out" == true ]]; then
  cmd+=(--output json)
fi

exec "${cmd[@]}"
