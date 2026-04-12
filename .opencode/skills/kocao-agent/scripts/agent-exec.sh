#!/usr/bin/env bash
# agent-exec.sh — Send a prompt/command to a running workspace session.
# Uses the attach mechanism to deliver input in driver mode.
# For full interactive sessions, use `kocao sessions attach <id> --driver` directly.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

# --- Preflight ---
if ! command -v kocao &>/dev/null; then
  echo "error: kocao binary not found in PATH" >&2
  echo "Install: go install github.com/withakay/kocao/cmd/kocao@latest" >&2
  exit 2
fi

if ! command -v curl &>/dev/null; then
  echo "error: curl is required but not found in PATH" >&2
  exit 2
fi

if ! command -v jq &>/dev/null; then
  echo "error: jq is required but not found in PATH" >&2
  exit 2
fi

# --- Defaults ---
SESSION_ID=""
PROMPT=""
API_URL="${KOCAO_API_URL:-http://127.0.0.1:8080}"
TOKEN="${KOCAO_TOKEN:-}"

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --prompt|-p)
      PROMPT="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: agent-exec.sh <session-id> --prompt <text>"
      echo ""
      echo "Send a prompt to a running workspace session."
      echo ""
      echo "Arguments:"
      echo "  session-id       Workspace session ID"
      echo ""
      echo "Options:"
      echo "  --prompt TEXT    The prompt/command to send (required)"
      echo "  --help           Show this help"
      echo ""
      echo "Note: This sends the prompt via the control-plane exec API."
      echo "For interactive terminal sessions, use:"
      echo "  kocao sessions attach <session-id> --driver"
      exit 0
      ;;
    -*)
      echo "error: unknown flag: $1" >&2
      exit 2
      ;;
    *)
      if [[ -z "$SESSION_ID" ]]; then
        SESSION_ID="$1"
      else
        echo "error: unexpected argument: $1" >&2
        exit 2
      fi
      shift
      ;;
  esac
done

if [[ -z "$SESSION_ID" ]]; then
  echo "error: session-id is required" >&2
  echo "Usage: agent-exec.sh <session-id> --prompt <text>" >&2
  exit 2
fi

if [[ -z "$PROMPT" ]]; then
  echo "error: --prompt is required" >&2
  exit 2
fi

if [[ -z "$TOKEN" ]]; then
  echo "error: KOCAO_TOKEN is not set" >&2
  exit 2
fi

# --- Send prompt via exec API ---
# The exec endpoint sends a prompt to the running agent session.
# If the API does not have a dedicated exec endpoint, fall back to
# delivering input via the attach-token + websocket mechanism.
API_URL="${API_URL%/}"
ENCODED_ID=$(printf '%s' "$SESSION_ID" | jq -sRr @uri)

PAYLOAD=$(jq -n --arg prompt "$PROMPT" '{"prompt": $prompt}')

RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d "$PAYLOAD" \
  "${API_URL}/api/v1/workspace-sessions/${ENCODED_ID}/exec" \
  2>&1) || {
  echo "error: failed to call control-plane API" >&2
  exit 1
}

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" -lt 200 || "$HTTP_CODE" -ge 300 ]]; then
  # If exec endpoint is not available (404), report with guidance
  if [[ "$HTTP_CODE" == "404" ]]; then
    echo "error: exec endpoint not available (HTTP 404)" >&2
    echo "The control-plane may not support the exec API yet." >&2
    echo "Use interactive attach instead: kocao sessions attach ${SESSION_ID} --driver" >&2
    exit 1
  fi
  echo "error: API returned HTTP ${HTTP_CODE}" >&2
  echo "$BODY" >&2
  exit 1
fi

# --- Output ---
if [[ -n "$BODY" ]]; then
  echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
else
  jq -n --arg sid "$SESSION_ID" '{status:"sent",sessionId:$sid}'
fi
