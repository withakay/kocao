#!/usr/bin/env bash
set -euo pipefail

die() {
  echo "error: $*" >&2
  exit 1
}

canon_path() {
  local p
  p="$1"

  if command -v realpath >/dev/null 2>&1; then
    realpath -m -- "${p}"
    return
  fi

  if command -v python3 >/dev/null 2>&1; then
    python3 - "${p}" <<'PY'
import os
import sys

p = sys.argv[1]
print(os.path.realpath(os.path.normpath(p)))
PY
    return
  fi

  if command -v readlink >/dev/null 2>&1; then
    readlink -f -- "${p}"
    return
  fi

  printf '%s\n' "${p}"
}

require_safe_paths() {
  local ws raw_repo ws_abs repo_abs
  ws="$1"
  raw_repo="$2"

  [[ -n "${ws}" ]] || die "KOCAO_WORKSPACE_DIR must be non-empty"
  [[ -n "${raw_repo}" ]] || die "KOCAO_REPO_DIR must be non-empty"

  [[ "${ws}" == /* ]] || die "KOCAO_WORKSPACE_DIR must be an absolute path (got: ${ws})"
  [[ "${raw_repo}" == /* ]] || die "KOCAO_REPO_DIR must be an absolute path (got: ${raw_repo})"

  ws_abs="$(canon_path "${ws}")"
  repo_abs="$(canon_path "${raw_repo}")"

  [[ "${ws_abs}" != "/" ]] || die "refusing to use KOCAO_WORKSPACE_DIR=/"
  [[ "${repo_abs}" != "/" ]] || die "refusing to use KOCAO_REPO_DIR=/"

  # Ensure repo dir is strictly inside workspace dir.
  [[ "${repo_abs}" != "${ws_abs}" ]] || die "KOCAO_REPO_DIR must be within KOCAO_WORKSPACE_DIR (got same path: ${repo_abs})"
  [[ "${repo_abs}" == "${ws_abs%/}/"* ]] || die "KOCAO_REPO_DIR must be within KOCAO_WORKSPACE_DIR (got: ${repo_abs} not under ${ws_abs})"
}

workspace_dir=${KOCAO_WORKSPACE_DIR:-/workspace}
repo_dir=${KOCAO_REPO_DIR:-"${workspace_dir}/repo"}
sandbox_agent_pid=""

require_safe_paths "${workspace_dir}" "${repo_dir}"
workspace_dir="$(canon_path "${workspace_dir}")"
repo_dir="$(canon_path "${repo_dir}")"

mkdir -p "${workspace_dir}" "${workspace_dir}/home" "${workspace_dir}/.kocao"
export HOME="${HOME:-${workspace_dir}/home}"

# Disable interactive git prompts.
export GIT_TERMINAL_PROMPT=${GIT_TERMINAL_PROMPT:-0}

# If a token file is mounted, configure a non-echoing askpass helper.
if [[ -n "${KOCAO_GIT_TOKEN_FILE:-}" && -f "${KOCAO_GIT_TOKEN_FILE}" ]]; then
  export GIT_ASKPASS=${GIT_ASKPASS:-/usr/local/bin/kocao-git-askpass}
fi

cleanup() {
  if [[ -n "${sandbox_agent_pid}" ]] && kill -0 "${sandbox_agent_pid}" >/dev/null 2>&1; then
    kill "${sandbox_agent_pid}" >/dev/null 2>&1 || true
    wait "${sandbox_agent_pid}" 2>/dev/null || true
  fi
}

start_sandbox_agent() {
  local host port token log_file
  host=${KOCAO_SANDBOX_AGENT_HOST:-0.0.0.0}
  port=${KOCAO_SANDBOX_AGENT_PORT:-2468}
  token=${KOCAO_SANDBOX_AGENT_TOKEN:-}
  log_file=${KOCAO_SANDBOX_AGENT_LOG:-${workspace_dir}/.kocao/sandbox-agent.log}

  mkdir -p "$(dirname "${log_file}")"

  if [[ -n "${token}" ]]; then
    sandbox-agent server --token "${token}" --host "${host}" --port "${port}" >"${log_file}" 2>&1 &
  else
    sandbox-agent server --no-token --host "${host}" --port "${port}" >"${log_file}" 2>&1 &
  fi
  sandbox_agent_pid=$!

  for _ in $(seq 1 30); do
    if curl -fsS "http://127.0.0.1:${port}/v1/health" >/dev/null 2>&1; then
      export KOCAO_SANDBOX_AGENT_ENDPOINT="http://127.0.0.1:${port}"
      return 0
    fi
    if ! kill -0 "${sandbox_agent_pid}" >/dev/null 2>&1; then
      sed -n '1,120p' "${log_file}" >&2 || true
      die "sandbox-agent exited before becoming healthy"
    fi
    sleep 1
  done

  sed -n '1,120p' "${log_file}" >&2 || true
  die "sandbox-agent health endpoint did not become ready"
}

trap cleanup EXIT

if [[ -n "${KOCAO_REPO_URL:-}" ]]; then
  if [[ ! -d "${repo_dir}/.git" ]]; then
    rm -rf -- "${repo_dir}"
    git clone -- "${KOCAO_REPO_URL}" "${repo_dir}"
  fi
  if [[ -n "${KOCAO_REPO_REVISION:-}" ]]; then
    git -C "${repo_dir}" fetch --all --tags --prune

    resolved_commit=$(git -C "${repo_dir}" rev-parse --verify --quiet "${KOCAO_REPO_REVISION}^{commit}" || true)
    if [[ -z "${resolved_commit}" ]]; then
      resolved_commit=$(git -C "${repo_dir}" rev-parse --verify --quiet "origin/${KOCAO_REPO_REVISION}^{commit}" || true)
    fi
    [[ -n "${resolved_commit}" ]] || die "failed to resolve KOCAO_REPO_REVISION=${KOCAO_REPO_REVISION}"

    git -C "${repo_dir}" checkout --force --detach "${resolved_commit}"
  fi
fi

cd "${repo_dir}" 2>/dev/null || cd "${workspace_dir}"

if [[ "${KOCAO_AGENT_RUNTIME:-}" == "sandbox-agent" ]]; then
  start_sandbox_agent
fi

# Default behavior: keep the pod alive for interactive exec unless a command is provided.
if [[ "$#" -eq 0 ]]; then
  exec sleep infinity
fi

exec "$@"
