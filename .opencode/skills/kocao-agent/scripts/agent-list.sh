#!/usr/bin/env bash
# agent-list.sh — List all workspace sessions via kocao CLI.
# Wraps: kocao sessions ls --json
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-list.sh [--no-json] [extra kocao flags]

List all workspace sessions.

Options:
  --no-json   Output the default kocao table instead of JSON
  --help      Show this help

Any additional arguments are passed through to `kocao sessions ls`.
EOF
}

require_commands kocao

json_out=true
extra_args=()
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
    *)
      extra_args+=("$1")
      shift
      ;;
  esac
done

cmd=(kocao sessions ls)
if [[ "$json_out" == true ]]; then
  cmd+=(--json)
fi
cmd+=("${extra_args[@]}")

exec "${cmd[@]}"
