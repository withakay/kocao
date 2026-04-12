#!/usr/bin/env bash
# agent-status.sh — Get detailed status of a workspace session.
# Wraps: kocao sessions status <id> --json
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-status.sh <session-id> [--no-json]

Get detailed status for a workspace session.

Arguments:
  session-id    Workspace session ID

Options:
  --no-json     Output the default kocao text format instead of JSON
  --help        Show this help
EOF
}

require_commands kocao

session_id=""
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
    -*)
      usage_error "unknown flag: $1"
      ;;
    *)
      if [[ -n "$session_id" ]]; then
        usage_error "unexpected argument: $1"
      fi
      session_id="$1"
      shift
      ;;
  esac
done

require_nonempty "$session_id" "session-id"

cmd=(kocao sessions status "$session_id")
if [[ "$json_out" == true ]]; then
  cmd+=(--json)
fi

exec "${cmd[@]}"
