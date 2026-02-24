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

require_safe_paths "${workspace_dir}" "${repo_dir}"
workspace_dir="$(canon_path "${workspace_dir}")"
repo_dir="$(canon_path "${repo_dir}")"

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
    rm -rf -- "${repo_dir}"
    git clone -- "${KOCAO_REPO_URL}" "${repo_dir}"
  fi
  if [[ -n "${KOCAO_REPO_REVISION:-}" ]]; then
    git -C "${repo_dir}" fetch --all --tags --prune

    resolved_commit=$(git -C "${repo_dir}" rev-parse --verify --quiet -- "${KOCAO_REPO_REVISION}^{commit}" || true)
    [[ -n "${resolved_commit}" ]] || die "failed to resolve KOCAO_REPO_REVISION=${KOCAO_REPO_REVISION}"

    git -C "${repo_dir}" checkout --force --detach "${resolved_commit}"
  fi
fi

cd "${repo_dir}" 2>/dev/null || cd "${workspace_dir}"

# Default behavior: keep the pod alive for interactive exec unless a command is provided.
if [[ "$#" -eq 0 ]]; then
  exec sleep infinity
fi

exec "$@"
