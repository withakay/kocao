#!/usr/bin/env bash
# agent-logs.sh — Stream or fetch logs from a workspace session's pod.
# Wraps: kocao sessions logs <id> [--tail N] [--container NAME] [--follow] [--json]
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
TAIL=""
CONTAINER=""
FOLLOW=false
JSON_OUT=true

# --- Parse args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --tail)
      TAIL="$2"
      shift 2
      ;;
    --container)
      CONTAINER="$2"
      shift 2
      ;;
    --follow|-f)
      FOLLOW=true
      JSON_OUT=false  # --follow and --json are incompatible
      shift
      ;;
    --json)
      JSON_OUT=true
      shift
      ;;
    --no-json)
      JSON_OUT=false
      shift
      ;;
    --help|-h)
      echo "Usage: agent-logs.sh <session-id> [--tail N] [--container NAME] [--follow] [--json|--no-json]"
      echo ""
      echo "Fetch or stream logs from a workspace session."
      echo ""
      echo "Arguments:"
      echo "  session-id       Workspace session ID"
      echo ""
      echo "Options:"
      echo "  --tail N         Number of log lines (default: 200)"
      echo "  --container NAME Container name within the pod"
      echo "  --follow, -f     Continuously poll for new logs"
      echo "  --json           Output structured JSON (default; incompatible with --follow)"
      echo "  --no-json        Output plain text"
      echo "  --help           Show this help"
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
  echo "Usage: agent-logs.sh <session-id> [--tail N] [--follow]" >&2
  exit 2
fi

# --- Build command ---
CMD=(kocao sessions logs "$SESSION_ID")

if [[ -n "$TAIL" ]]; then
  CMD+=(--tail "$TAIL")
fi

if [[ -n "$CONTAINER" ]]; then
  CMD+=(--container "$CONTAINER")
fi

if [[ "$FOLLOW" == true ]]; then
  CMD+=(--follow)
elif [[ "$JSON_OUT" == true ]]; then
  CMD+=(--json)
fi

# --- Execute ---
exec "${CMD[@]}"
