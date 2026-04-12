#!/usr/bin/env bash
# agent-start.sh — Create a new workspace session via the kocao control-plane API.
# This script calls the API directly since the CLI does not yet have a
# "sessions create" subcommand.
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

# --- Defaults ---
REPO_URL=""
AGENT="opencode"
DISPLAY_NAME=""
QUIET=false
API_URL="${KOCAO_API_URL:-http://127.0.0.1:8080}"
TOKEN="${KOCAO_TOKEN:-}"
WAIT=true
WAIT_TIMEOUT=120

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)
      REPO_URL="$2"
      shift 2
      ;;
    --agent)
      AGENT="$2"
      shift 2
      ;;
    --name)
      DISPLAY_NAME="$2"
      shift 2
      ;;
    --quiet|-q)
      QUIET=true
      shift
      ;;
    --no-wait)
      WAIT=false
      shift
      ;;
    --wait-timeout)
      WAIT_TIMEOUT="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: agent-start.sh --repo <url> [--agent <name>] [--name <display-name>] [--quiet] [--no-wait]"
      echo ""
      echo "Create a new workspace session."
      echo ""
      echo "Options:"
      echo "  --repo URL          Repository URL (required)"
      echo "  --agent NAME        Agent type: opencode, codex, claude, pi (default: opencode)"
      echo "  --name NAME         Display name for the session"
      echo "  --quiet             Output only the session ID"
      echo "  --no-wait           Don't wait for Running phase"
      echo "  --wait-timeout SEC  Max seconds to wait for Running (default: 120)"
      echo "  --help              Show this help"
      exit 0
      ;;
    *)
      echo "error: unknown flag: $1" >&2
      echo "Run with --help for usage" >&2
      exit 2
      ;;
  esac
done

if [[ -z "$REPO_URL" ]]; then
  echo "error: --repo is required" >&2
  exit 2
fi

if [[ -z "$TOKEN" ]]; then
  echo "error: KOCAO_TOKEN is not set" >&2
  exit 2
fi

# Validate agent type
case "$AGENT" in
  opencode|codex|claude|pi) ;;
  *)
    echo "error: unsupported agent type: $AGENT (expected: opencode, codex, claude, pi)" >&2
    exit 2
    ;;
esac

# --- Build request ---
if [[ -z "$DISPLAY_NAME" ]]; then
  DISPLAY_NAME="${AGENT}-$(date +%s)"
fi

PAYLOAD=$(cat <<EOF
{
  "displayName": "${DISPLAY_NAME}",
  "repoURL": "${REPO_URL}",
  "agent": "${AGENT}"
}
EOF
)

# --- Create session ---
API_URL="${API_URL%/}"
RESPONSE=$(curl -s -w "\n%{http_code}" \
  -X POST \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d "$PAYLOAD" \
  "${API_URL}/api/v1/workspace-sessions" \
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

SESSION_ID=$(echo "$BODY" | jq -r '.id // empty' 2>/dev/null)
if [[ -z "$SESSION_ID" ]]; then
  echo "error: could not extract session ID from response" >&2
  echo "$BODY" >&2
  exit 1
fi

# --- Wait for Running (optional) ---
if [[ "$WAIT" == true ]]; then
  ELAPSED=0
  while [[ $ELAPSED -lt $WAIT_TIMEOUT ]]; do
    STATUS_JSON=$(kocao sessions status "$SESSION_ID" --json 2>/dev/null) || true
    PHASE=$(echo "$STATUS_JSON" | jq -r '.session.phase // empty' 2>/dev/null)
    case "$PHASE" in
      Running)
        break
        ;;
      Failed)
        echo "error: session entered Failed phase" >&2
        echo "$STATUS_JSON" | jq . >&2 2>/dev/null || echo "$STATUS_JSON" >&2
        exit 1
        ;;
    esac
    sleep 3
    ELAPSED=$((ELAPSED + 3))
  done

  if [[ $ELAPSED -ge $WAIT_TIMEOUT ]]; then
    echo "warning: timed out waiting for Running phase (current: ${PHASE:-unknown})" >&2
  fi
fi

# --- Output ---
if [[ "$QUIET" == true ]]; then
  echo "$SESSION_ID"
else
  echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
fi
