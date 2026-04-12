#!/usr/bin/env bash
# agent-exec.sh — Send a prompt/command to a running workspace session.
# Uses the experimental /exec API when it is available on the control-plane.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-exec.sh <session-id> --prompt <text>

Send a prompt to a running workspace session through the control-plane exec API.

Arguments:
  session-id       Workspace session ID

Options:
  --prompt, -p     Prompt text to send (required)
  --help           Show this help

Notes:
  This script depends on the optional `/api/v1/workspace-sessions/<id>/exec`
  endpoint. If your control-plane does not expose that endpoint yet, use:
    kocao sessions attach <session-id> --driver
EOF
}

require_commands curl jq

session_id=""
prompt=""
api_response_file=""
trap '[[ -n "$api_response_file" ]] && rm -f "$api_response_file"' EXIT

while (($#)); do
  case "$1" in
    --prompt|-p)
      require_flag_value "$1" "$#"
      prompt="$2"
      shift 2
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
require_nonempty "$prompt" "--prompt"

payload="$(jq -n --arg prompt "$prompt" '{prompt: $prompt}')"
api_request POST "/api/v1/workspace-sessions/$(urlencode "$session_id")/exec" "$payload"
api_response_file="$API_RESPONSE_FILE"
if ! api_request_ok; then
  if [[ "$API_RESPONSE_CODE" == '404' ]]; then
    echo 'error: exec endpoint is not available on this control-plane' >&2
    echo "Use interactive attach instead: kocao sessions attach ${session_id} --driver" >&2
    exit 1
  fi
  print_api_error "$API_RESPONSE_CODE" "$API_RESPONSE_FILE"
  exit 1
fi

if [[ -s "$API_RESPONSE_FILE" ]]; then
  print_json_or_raw "$API_RESPONSE_FILE"
else
  jq -n --arg sid "$session_id" '{status:"sent",sessionId:$sid}'
fi
