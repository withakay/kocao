#!/usr/bin/env bash
set -euo pipefail

workspace_dir=${KOCAO_WORKSPACE_DIR:-/workspace}
repo_dir=${KOCAO_REPO_DIR:-"${workspace_dir}/repo"}

mkdir -p "${workspace_dir}" "${workspace_dir}/home"
export HOME="${HOME:-${workspace_dir}/home}"

# Disable interactive git prompts.
export GIT_TERMINAL_PROMPT=${GIT_TERMINAL_PROMPT:-0}

# If a token file is mounted, configure a non-echoing askpass helper.
if [[ -n "${KOCAO_GIT_TOKEN_FILE:-}" && -f "${KOCAO_GIT_TOKEN_FILE}" ]]; then
  export GIT_ASKPASS=${GIT_ASKPASS:-/usr/local/bin/kocao-git-askpass}
fi

if [[ -n "${KOCAO_REPO_URL:-}" ]]; then
  if [[ ! -d "${repo_dir}/.git" ]]; then
    rm -rf "${repo_dir}"
    git clone "${KOCAO_REPO_URL}" "${repo_dir}"
  fi
  if [[ -n "${KOCAO_REPO_REVISION:-}" ]]; then
    git -C "${repo_dir}" fetch --all --tags --prune
    git -C "${repo_dir}" checkout --force "${KOCAO_REPO_REVISION}"
  fi
fi

cd "${repo_dir}" 2>/dev/null || cd "${workspace_dir}"

# Default behavior: keep the pod alive for interactive exec unless a command is provided.
if [[ "$#" -eq 0 ]]; then
  exec sleep infinity
fi

exec "$@"
