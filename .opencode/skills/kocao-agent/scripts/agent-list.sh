#!/usr/bin/env bash
# agent-list.sh — List all workspace sessions via kocao CLI.
# Wraps: kocao sessions ls --json
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

# --- Preflight ---
if ! command -v kocao &>/dev/null; then
  echo "error: kocao binary not found in PATH" >&2
  echo "Install: go install github.com/withakay/kocao/cmd/kocao@latest" >&2
  exit 2
fi

# --- Parse args ---
OUTPUT_FORMAT="--json"
EXTRA_ARGS=()

while [[ $# -gt 0 ]]; do
  case "$1" in
    --no-json)
      OUTPUT_FORMAT=""
      shift
      ;;
    --help|-h)
      echo "Usage: agent-list.sh [--no-json]"
      echo ""
      echo "List all workspace sessions."
      echo ""
      echo "Options:"
      echo "  --no-json   Output human-readable table instead of JSON"
      echo "  --help      Show this help"
      exit 0
      ;;
    *)
      EXTRA_ARGS+=("$1")
      shift
      ;;
  esac
done

# --- Execute ---
exec kocao sessions ls ${OUTPUT_FORMAT} "${EXTRA_ARGS[@]+"${EXTRA_ARGS[@]}"}"
