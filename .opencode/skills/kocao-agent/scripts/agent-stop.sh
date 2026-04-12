#!/usr/bin/env bash
# agent-stop.sh — Stop/terminate a workspace session via the kocao control-plane API.
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
API_URL="${KOCAO_API_URL:-http://127.0.0.1:8080}"
TOKEN="${KOCAO_TOKEN:-}"
OUTPUT_JSON=true

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-json)
      OUTPUT_JSON=false
      shift
      ;;
    --help|-h)
      echo "Usage: agent-stop.sh <session-id> [--no-json]"
      echo ""
      echo "Stop a workspace session."
      echo ""
      echo "Arguments:"
      echo "  session-id    Workspace session ID to stop"
      echo ""
      echo "Options:"
      echo "  --no-json     Output human-readable text instead of JSON"
      echo "  --help        Show this help"
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
  echo "Usage: agent-stop.sh <session-id>" >&2
  exit 2
fi

if [[ -z "$TOKEN" ]]; then
  echo "error: KOCAO_TOKEN is not set" >&2
  exit 2
fi

# --- Stop session ---
API_URL="${API_URL%/}"
RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X DELETE \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Accept: application/json" \
  "${API_URL}/api/v1/workspace-sessions/$(printf '%s' "$SESSION_ID" | jq -sRr @uri)" \
  2>&1) || {
  echo "error: failed to call control-plane API" >&2
  exit 1
}

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [[ "$HTTP_CODE" -lt 200 || "$HTTP_CODE" -ge 300 ]]; then
  echo "error: API returned HTTP ${HTTP_CODE}" >&2
  echo "$BODY" >&2
  exit 1
fi

# --- Output ---
if [[ "$OUTPUT_JSON" == true ]]; then
  if [[ -n "$BODY" ]]; then
    echo "$BODY" | jq . 2>/dev/null || jq -n --arg sid "$SESSION_ID" '{status:"stopped",sessionId:$sid}'
  else
    jq -n --arg sid "$SESSION_ID" '{status:"stopped",sessionId:$sid}'
  fi
else
  echo "Stopped session: ${SESSION_ID}"
fi
