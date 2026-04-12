#!/usr/bin/env bash
# agent-status.sh — Get detailed status of a workspace session.
# Wraps: kocao sessions status <id> --json
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

# --- Preflight ---
if ! command -v kocao &>/dev/null; then
  echo "error: kocao binary not found in PATH" >&2
  echo "Install: go install github.com/withakay/kocao/cmd/kocao@latest" >&2
  exit 2
fi

# --- Defaults ---
SESSION_ID=""
JSON_OUT=true

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-json)
      JSON_OUT=false
      shift
      ;;
    --help|-h)
      echo "Usage: agent-status.sh <session-id> [--no-json]"
      echo ""
      echo "Get detailed status of a workspace session."
      echo ""
      echo "Arguments:"
      echo "  session-id    Workspace session ID"
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
  echo "Usage: agent-status.sh <session-id>" >&2
  exit 2
fi

# --- Build command ---
CMD=(kocao sessions status "$SESSION_ID")

if [[ "$JSON_OUT" == true ]]; then
  CMD+=(--json)
fi

# --- Execute ---
exec "${CMD[@]}"
