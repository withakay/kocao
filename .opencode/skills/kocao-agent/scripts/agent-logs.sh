#!/usr/bin/env bash
# agent-logs.sh — Stream or fetch agent session events.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-logs.sh <run-id> [--tail N] [--follow] [--no-json]

Fetch or stream agent session events.

Arguments:
  run-id         Harness run ID

Options:
  --tail N       Number of event lines to fetch
  --follow, -f   Stream events continuously
  --no-json      Output the default kocao table instead of JSON
  --help         Show this help
EOF
}

require_commands kocao

run_id=""
tail_lines=""
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
    --follow|-f)
      follow=true
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

cmd=(kocao agent logs "$run_id")
if [[ -n "$tail_lines" ]]; then
  cmd+=(--tail "$tail_lines")
fi
if [[ "$follow" == true ]]; then
  cmd+=(--follow)
fi
if [[ "$json_out" == true ]]; then
  cmd+=(--output json)
fi

exec "${cmd[@]}"
