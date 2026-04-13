#!/usr/bin/env bash
set -euo pipefail

# zoekt-index.sh — Index a source directory with zoekt-index.
#
# Resolves the zoekt-index binary from the skill's bin/ directory or PATH,
# auto-installs via install-zoekt.sh if missing, then indexes the target
# directory into a zoekt shard directory.
#
# Usage:
#   zoekt-index.sh [--index-dir <path>] [--help] [<source-dir>]
#
# Environment variables:
#   ZOEKT_INDEX_DIR — Override default index directory

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../bin"

log() {
    printf '[zoekt-index] %s\n' "$*"
}

die() {
    printf '[zoekt-index] ERROR: %s\n' "$*" >&2
    exit 1
}

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS] [SOURCE_DIR]

Index a source directory with zoekt-index.

Options:
  --index-dir <path>   Override index directory (default: \$REPO_ROOT/.git/zoekt)
  --help               Show this help message

Arguments:
  SOURCE_DIR           Directory to index (default: git repo root)

Environment:
  ZOEKT_INDEX_DIR      Override index directory (same as --index-dir)
EOF
}

resolve_binary() {
    # 1. Check skill bin/ directory
    if [[ -x "${BIN_DIR}/zoekt-index" ]]; then
        ZOEKT_INDEX_BIN="${BIN_DIR}/zoekt-index"
        return 0
    fi

    # 2. Check PATH
    if command -v zoekt-index >/dev/null 2>&1; then
        ZOEKT_INDEX_BIN="$(command -v zoekt-index)"
        return 0
    fi

    return 1
}

auto_install() {
    local install_script="${SCRIPT_DIR}/install-zoekt.sh"
    if [[ ! -x "${install_script}" ]]; then
        die "zoekt-index not found and install script missing at ${install_script}"
    fi

    log "zoekt-index not found; running install-zoekt.sh ..."
    if ! bash "${install_script}"; then
        die "Auto-install failed. Install zoekt manually or check Go installation."
    fi

    # Retry resolution after install
    if ! resolve_binary; then
        die "zoekt-index still not found after auto-install."
    fi
}

detect_repo_root() {
    if git rev-parse --show-toplevel >/dev/null 2>&1; then
        git rev-parse --show-toplevel
    else
        die "Not inside a git repository. Provide source directory as argument."
    fi
}

main() {
    local index_dir=""
    local source_dir=""
    ZOEKT_INDEX_BIN=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --index-dir)
                [[ $# -ge 2 ]] || die "--index-dir requires an argument"
                index_dir="$2"
                shift 2
                ;;
            --help)
                usage
                exit 0
                ;;
            -*)
                die "Unknown option: $1"
                ;;
            *)
                [[ -z "${source_dir}" ]] || die "Multiple source directories specified"
                source_dir="$1"
                shift
                ;;
        esac
    done

    # Resolve binary (with auto-install fallback)
    if ! resolve_binary; then
        auto_install
    fi

    log "Using binary: ${ZOEKT_INDEX_BIN}"

    # Determine source directory
    if [[ -z "${source_dir}" ]]; then
        source_dir="$(detect_repo_root)"
    fi

    if [[ ! -d "${source_dir}" ]]; then
        die "Source directory does not exist: ${source_dir}"
    fi

    # Resolve to absolute path
    source_dir="$(cd "${source_dir}" && pwd)"

    # Determine index directory: flag > env var > default
    if [[ -z "${index_dir}" ]]; then
        index_dir="${ZOEKT_INDEX_DIR:-""}"
    fi
    if [[ -z "${index_dir}" ]]; then
        # Use git rev-parse --git-dir for worktree-safe resolution
        # (in worktrees, .git is a file, not a directory)
        local git_dir
        git_dir="$(git rev-parse --git-dir)"
        index_dir="${git_dir}/zoekt"
    fi

    mkdir -p "${index_dir}"

    log "Indexing: ${source_dir}"
    log "Index dir: ${index_dir}"

    # Run zoekt-index
    "${ZOEKT_INDEX_BIN}" -index "${index_dir}" "${source_dir}"

    # Report summary
    local shard_count=0
    local index_size=""
    if [[ -d "${index_dir}" ]]; then
        shard_count="$(find "${index_dir}" -maxdepth 1 -name '*.zoekt' -type f 2>/dev/null | wc -l | tr -d ' ')"
        if command -v du >/dev/null 2>&1; then
            index_size="$(du -sh "${index_dir}" 2>/dev/null | cut -f1 | tr -d ' ')"
        fi
    fi

    log "Indexed ${source_dir} → ${index_dir}"
    log "  Shards: ${shard_count}"
    if [[ -n "${index_size}" ]]; then
        log "  Index size: ${index_size}"
    fi
}

main "$@"
