#!/usr/bin/env bash
# agent-exec.sh — Send a prompt to a running agent session.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-exec.sh <run-id> [--prompt <text> | <text>] [--no-json]

Send a prompt to a running agent session.

Arguments:
  run-id       Harness run ID

Options:
  --prompt, -p Prompt text to send
  --no-json    Use the CLI's default formatted output instead of JSON
  --help       Show this help
EOF
}

require_commands kocao

run_id=""
prompt=""
json_out=true
while (($#)); do
  case "$1" in
    --prompt|-p)
      require_flag_value "$1" "$#"
      prompt="$2"
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
      if [[ -z "$run_id" ]]; then
        run_id="$1"
      elif [[ -z "$prompt" ]]; then
        prompt="$1"
      else
        prompt+=" $1"
      fi
      shift
      ;;
  esac
done

require_nonempty "$run_id" "run-id"
require_nonempty "$prompt" "--prompt"

cmd=(kocao agent exec "$run_id" --prompt "$prompt")
if [[ "$json_out" == true ]]; then
  cmd+=(--output json)
fi

exec "${cmd[@]}"
