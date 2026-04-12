#!/usr/bin/env bash
# agent-start.sh — Create a new workspace session via the kocao control-plane API.
# The API currently creates a generic workspace session; the --agent flag is kept
# as a naming hint so older examples keep working.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-start.sh --repo <url> [--agent <name>] [--name <display-name>] [--quiet] [--no-wait] [--wait-timeout SEC]

Create a new workspace session.

Options:
  --repo URL          Repository URL to clone into the session (required)
  --agent NAME        Session label for the intended agent workflow: opencode, codex, claude, pi
  --name NAME         Explicit display name for the session
  --quiet, -q         Output only the session ID
  --no-wait           Return immediately after creation
  --wait-timeout SEC  Max seconds to wait for the session to become Running
  --help              Show this help

Notes:
  --agent is currently a display-name hint only. The control-plane API creates a
  generic workspace session and does not switch runtime images based on this flag.
EOF
}

require_commands curl jq

repo_url=""
agent_label="opencode"
display_name=""
quiet=false
wait_for_running=true
wait_timeout=120
api_response_file=""
trap '[[ -n "$api_response_file" ]] && rm -f "$api_response_file"' EXIT

while (($#)); do
  case "$1" in
    --repo)
      require_flag_value "$1" "$#"
      repo_url="$2"
      shift 2
      ;;
    --agent)
      require_flag_value "$1" "$#"
      agent_label="$2"
      shift 2
      ;;
    --name)
      require_flag_value "$1" "$#"
      display_name="$2"
      shift 2
      ;;
    --quiet|-q)
      quiet=true
      shift
      ;;
    --no-wait)
      wait_for_running=false
      shift
      ;;
    --wait-timeout)
      require_flag_value "$1" "$#"
      wait_timeout="$2"
      require_positive_integer "$wait_timeout" "--wait-timeout"
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
      usage_error "unexpected argument: $1"
      ;;
  esac
done

if [[ "$wait_for_running" == true ]]; then
  require_command kocao
fi

require_nonempty "$repo_url" "--repo"
case "$agent_label" in
  opencode|codex|claude|pi) ;;
  *)
    usage_error "unsupported --agent value: ${agent_label} (expected: opencode, codex, claude, pi)"
    ;;
esac

if [[ -z "$display_name" ]]; then
  display_name="${agent_label}-$(date +%s)"
fi

payload="$(jq -n --arg displayName "$display_name" --arg repoURL "$repo_url" '{displayName: $displayName, repoURL: $repoURL}')"
api_request POST '/api/v1/workspace-sessions' "$payload"
api_response_file="$API_RESPONSE_FILE"
if ! api_request_ok; then
  print_api_error "$API_RESPONSE_CODE" "$API_RESPONSE_FILE"
  exit 1
fi

session_id="$(jq -r '.id // empty' "$API_RESPONSE_FILE" 2>/dev/null)"
if [[ -z "$session_id" ]]; then
  echo 'error: create response did not include a session id' >&2
  print_json_or_raw "$API_RESPONSE_FILE" >&2 || true
  exit 1
fi

final_output="$API_RESPONSE_FILE"
if [[ "$wait_for_running" == true ]]; then
  elapsed=0
  current_phase=""
  while (( elapsed < wait_timeout )); do
    status_json="$(kocao sessions status "$session_id" --json 2>/dev/null || true)"
    current_phase="$(jq -r '.session.phase // empty' <<<"$status_json" 2>/dev/null || true)"
    case "$current_phase" in
      Running)
        status_file="$(mktemp)"
        printf '%s' "$status_json" > "$status_file"
        final_output="$status_file"
        api_response_file="$status_file"
        break
        ;;
      Failed)
        echo "error: session ${session_id} entered Failed phase" >&2
        if [[ -n "$status_json" ]]; then
          jq . <<<"$status_json" >&2 2>/dev/null || echo "$status_json" >&2
        fi
        exit 1
        ;;
    esac
    sleep 3
    elapsed=$((elapsed + 3))
  done

  if (( elapsed >= wait_timeout )) && [[ "$current_phase" != "Running" ]]; then
    echo "warning: timed out waiting for session ${session_id} to reach Running (current: ${current_phase:-unknown})" >&2
  fi
fi

if [[ "$quiet" == true ]]; then
  echo "$session_id"
else
  print_json_or_raw "$final_output"
fi