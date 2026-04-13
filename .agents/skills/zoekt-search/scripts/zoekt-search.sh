#!/usr/bin/env bash
set -euo pipefail

# zoekt-search.sh — Search a zoekt index for a given query.
#
# Resolves the zoekt binary from the skill's bin/ directory or PATH,
# auto-installs if missing, and searches the repo's local index.
#
# Usage:
#   zoekt-search.sh [--index-dir <path>] [--no-json] <query...>
#
# Environment variables:
#   ZOEKT_INDEX_DIR — override the index directory (default: <git-dir>/zoekt)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../bin"

usage() {
    cat <<'EOF'
Usage: zoekt-search.sh [OPTIONS] QUERY...

Search a zoekt index.

Options:
  --index-dir <path>   Override index directory (default: <git-dir>/zoekt)
  --no-json            Plain text output instead of JSONL
  --help               Show this help

Environment:
  ZOEKT_INDEX_DIR      Override index directory (same as --index-dir)

Examples:
  zoekt-search.sh "AgentSession"
  zoekt-search.sh --no-json "func main"
  zoekt-search.sh --index-dir /tmp/my-index "byte file:java"
EOF
    exit "${1:-0}"
}

die() {
    printf '[zoekt-search] ERROR: %s\n' "$*" >&2
    exit 1
}

log() {
    printf '[zoekt-search] %s\n' "$*" >&2
}

# --- Resolve zoekt binary ---------------------------------------------------

resolve_zoekt() {
    # 1. Skill-local bin/
    if [[ -x "${BIN_DIR}/zoekt" ]]; then
        ZOEKT_BIN="${BIN_DIR}/zoekt"
        return 0
    fi

    # 2. PATH
    if command -v zoekt >/dev/null 2>&1; then
        ZOEKT_BIN="$(command -v zoekt)"
        return 0
    fi

    return 1
}

ensure_zoekt() {
    if resolve_zoekt; then
        return 0
    fi

    # Auto-install
    log "zoekt binary not found. Running install-zoekt.sh..."
    if [[ -x "${SCRIPT_DIR}/install-zoekt.sh" ]]; then
        bash "${SCRIPT_DIR}/install-zoekt.sh"
    else
        die "install-zoekt.sh not found at ${SCRIPT_DIR}/install-zoekt.sh"
    fi

    # Retry after install
    if resolve_zoekt; then
        return 0
    fi

    die "zoekt binary still not found after auto-install."
}

# --- Determine index directory -----------------------------------------------

resolve_index_dir() {
    local index_dir="${1:-}"

    # 1. Explicit flag takes precedence (already set by caller)
    if [[ -n "${index_dir}" ]]; then
        echo "${index_dir}"
        return 0
    fi

    # 2. Environment variable
    if [[ -n "${ZOEKT_INDEX_DIR:-}" ]]; then
        echo "${ZOEKT_INDEX_DIR}"
        return 0
    fi

    # 3. Default: <git-dir>/zoekt in current repo
    #    Uses git rev-parse --git-dir which works in both normal repos and worktrees.
    local git_dir
    if git_dir="$(git rev-parse --git-dir 2>/dev/null)"; then
        # Resolve to absolute path
        if [[ "${git_dir}" != /* ]]; then
            git_dir="$(cd "${git_dir}" && pwd)"
        fi
        echo "${git_dir}/zoekt"
        return 0
    fi

    die "Not inside a git repository and no --index-dir or ZOEKT_INDEX_DIR set."
}

# --- Main --------------------------------------------------------------------

main() {
    local index_dir_override=""
    local use_json=true
    local query_args=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --index-dir)
                [[ $# -ge 2 ]] || die "--index-dir requires an argument"
                index_dir_override="$2"
                shift 2
                ;;
            --no-json)
                use_json=false
                shift
                ;;
            --help|-h)
                usage 0
                ;;
            --)
                shift
                query_args+=("$@")
                break
                ;;
            *)
                query_args+=("$1")
                shift
                ;;
        esac
    done

    # Must have a query
    if [[ ${#query_args[@]} -eq 0 ]]; then
        printf 'Error: no query provided.\n\n' >&2
        usage 1
    fi

    # Resolve binary
    ensure_zoekt

    # Resolve index directory
    local index_dir
    index_dir="$(resolve_index_dir "${index_dir_override}")"

    # Check index exists
    if [[ ! -d "${index_dir}" ]] || ! compgen -G "${index_dir}/*.zoekt" >/dev/null 2>&1; then
        die "No index found at ${index_dir}. Run zoekt-index.sh first."
    fi

    # Build zoekt command
    local cmd=("${ZOEKT_BIN}" "-index_dir" "${index_dir}")
    if [[ "${use_json}" == "true" ]]; then
        cmd+=("-jsonl")
    fi
    cmd+=("${query_args[@]}")

    exec "${cmd[@]}"
}

main "$@"
