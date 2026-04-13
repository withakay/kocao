#!/usr/bin/env bash
# agent-list.sh — List agent sessions via kocao CLI.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-list.sh [--workspace <id>] [--no-json]

List remote agent sessions.

Options:
  --workspace ID  Filter by workspace session ID
  --no-json       Output the default kocao table instead of JSON
  --help          Show this help
EOF
}

require_commands kocao

workspace_id=""
json_out=true
while (($#)); do
  case "$1" in
    --workspace)
      require_flag_value "$1" "$#"
      workspace_id="$2"
      shift 2
      ;;
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
      usage_error "unexpected argument: $1"
      ;;
  esac
done

cmd=(kocao agent list)
if [[ -n "$workspace_id" ]]; then
  cmd+=(--workspace "$workspace_id")
fi
if [[ "$json_out" == true ]]; then
  cmd+=(--output json)
fi

exec "${cmd[@]}"
