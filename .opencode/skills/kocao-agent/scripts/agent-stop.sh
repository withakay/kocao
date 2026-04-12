#!/usr/bin/env bash
# agent-stop.sh — Stop/terminate a workspace session via the kocao control-plane API.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-stop.sh <session-id> [--no-json]

Stop a workspace session.

Arguments:
  session-id    Workspace session ID to stop

Options:
  --no-json     Output a short confirmation line instead of JSON
  --help        Show this help
EOF
}

require_commands curl jq

session_id=""
json_out=true
api_response_file=""
trap '[[ -n "$api_response_file" ]] && rm -f "$api_response_file"' EXIT

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

api_request DELETE "/api/v1/workspace-sessions/$(urlencode "$session_id")"
api_response_file="$API_RESPONSE_FILE"
if ! api_request_ok; then
  print_api_error "$API_RESPONSE_CODE" "$API_RESPONSE_FILE"
  exit 1
fi

if [[ "$json_out" == true ]]; then
  if [[ -s "$API_RESPONSE_FILE" ]]; then
    print_json_or_raw "$API_RESPONSE_FILE"
  else
    jq -n --arg sid "$session_id" '{status:"stopped",sessionId:$sid}'
  fi
else
  echo "Stopped session: ${session_id}"
fi
