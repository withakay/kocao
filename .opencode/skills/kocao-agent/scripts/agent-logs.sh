#!/usr/bin/env bash
# agent-logs.sh — Stream or fetch logs from a workspace session's pod.
# Wraps: kocao sessions logs <id> [--tail N] [--container NAME] [--follow] [--json]
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-logs.sh <session-id> [--tail N] [--container NAME] [--follow] [--json|--no-json]

Fetch or stream logs from a workspace session.

Arguments:
  session-id       Workspace session ID

Options:
  --tail N         Number of log lines to fetch
  --container NAME Container name within the pod
  --follow, -f     Stream logs continuously
  --json           Output JSON (default for one-shot fetches)
  --no-json        Output plain text
  --help           Show this help

Notes:
  --follow and --json cannot be combined.
EOF
}

require_commands kocao

session_id=""
tail_lines=""
container_name=""
follow=false
json_out=true
while (($#)); do
  case "$1" in
    --tail)
      require_flag_value "$1" "$#"
      tail_lines="$2"
      require_positive_integer "$tail_lines" "--tail"
      shift 2
      ;;
    --container)
      require_flag_value "$1" "$#"
      container_name="$2"
      shift 2
      ;;
    --follow|-f)
      follow=true
      shift
      ;;
    --json)
      json_out=true
      shift
      ;;
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
if [[ "$follow" == true && "$json_out" == true ]]; then
  usage_error "--follow cannot be combined with --json"
fi

cmd=(kocao sessions logs "$session_id")
if [[ -n "$tail_lines" ]]; then
  cmd+=(--tail "$tail_lines")
fi
if [[ -n "$container_name" ]]; then
  cmd+=(--container "$container_name")
fi
if [[ "$follow" == true ]]; then
  cmd+=(--follow)
elif [[ "$json_out" == true ]]; then
  cmd+=(--json)
fi

exec "${cmd[@]}"
