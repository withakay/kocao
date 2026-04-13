#!/usr/bin/env bash
# agent-start.sh — Start a remote agent session.
# Exit codes: 0=success, 1=runtime error, 2=usage error
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=.opencode/skills/kocao-agent/scripts/common.sh
source "${SCRIPT_DIR}/common.sh"

usage() {
  cat <<'EOF'
Usage: agent-start.sh --repo <url> --agent <name> [--workspace <id>] [--revision <ref>] [--image <image>] [--image-pull-secret <name>] [--egress-mode <mode>] [--timeout <duration>] [--quiet] [--no-json]

Start a remote agent session.

Options:
  --repo URL               Repository URL to clone (required)
  --agent NAME             Agent name: opencode, codex, claude, pi (required)
  --workspace ID           Reuse an existing workspace session
  --revision REF           Repository revision (default: main)
  --image IMAGE            Harness runtime image override
  --image-pull-secret NAME Image pull secret for private registries
  --egress-mode MODE       Harness pod egress mode: restricted or full
  --timeout DURATION       Max wait time for readiness
  --quiet, -q              Output only the run ID
  --no-json                Use the CLI's default formatted output instead of JSON
  --help                   Show this help
EOF
}

require_commands kocao jq

repo_url=""
agent_name=""
workspace_id=""
revision="main"
image=""
image_pull_secret=""
egress_mode=""
timeout_duration=""
quiet=false
json_out=true

while (($#)); do
  case "$1" in
    --repo)
      require_flag_value "$1" "$#"
      repo_url="$2"
      shift 2
      ;;
    --agent)
      require_flag_value "$1" "$#"
      agent_name="$2"
      shift 2
      ;;
    --workspace)
      require_flag_value "$1" "$#"
      workspace_id="$2"
      shift 2
      ;;
    --revision)
      require_flag_value "$1" "$#"
      revision="$2"
      shift 2
      ;;
    --image)
      require_flag_value "$1" "$#"
      image="$2"
      shift 2
      ;;
    --image-pull-secret)
      require_flag_value "$1" "$#"
      image_pull_secret="$2"
      shift 2
      ;;
    --egress-mode)
      require_flag_value "$1" "$#"
      egress_mode="$2"
      shift 2
      ;;
    --timeout)
      require_flag_value "$1" "$#"
      timeout_duration="$2"
      shift 2
      ;;
    --quiet|-q)
      quiet=true
      shift
      ;;
    --no-json)
      json_out=false
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    -* )
      usage_error "unknown flag: $1"
      ;;
    *)
      usage_error "unexpected argument: $1"
      ;;
  esac
done

require_nonempty "$repo_url" "--repo"
require_nonempty "$agent_name" "--agent"

cmd=(kocao agent start --repo "$repo_url" --agent "$agent_name")
if [[ -n "$workspace_id" ]]; then
  cmd+=(--workspace "$workspace_id")
fi
if [[ -n "$revision" ]]; then
  cmd+=(--revision "$revision")
fi
if [[ -n "$image" ]]; then
  cmd+=(--image "$image")
fi
if [[ -n "$image_pull_secret" ]]; then
  cmd+=(--image-pull-secret "$image_pull_secret")
fi
if [[ -n "$egress_mode" ]]; then
  cmd+=(--egress-mode "$egress_mode")
fi
if [[ -n "$timeout_duration" ]]; then
  cmd+=(--timeout "$timeout_duration")
fi

if [[ "$quiet" == true ]]; then
  output="$("${cmd[@]}" --output json)" || exit $?
  jq -r '.runId // .sessionId // empty' <<<"$output"
  exit 0
fi

if [[ "$json_out" == true ]]; then
  cmd+=(--output json)
fi

exec "${cmd[@]}"
