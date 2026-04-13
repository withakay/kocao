#!/usr/bin/env bash
set -euo pipefail

# install-zoekt.sh — Self-bootstrapping installer for zoekt binaries.
#
# Installs zoekt-index and zoekt into the skill's bin/ directory.
# Since sourcegraph/zoekt publishes no pre-built release binaries,
# this script builds from source using `go install`.
#
# Environment variables:
#   ZOEKT_VERSION  — Go module version to install (default: "latest")
#   BIN_DIR        — Destination directory for binaries (default: ../bin/ relative to this script)

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${BIN_DIR:-"${SCRIPT_DIR}/../bin"}"
ZOEKT_VERSION="${ZOEKT_VERSION:-latest}"

ZOEKT_MODULE="github.com/sourcegraph/zoekt"
ZOEKT_CMDS=("zoekt-index" "zoekt")

log() {
    printf '[install-zoekt] %s\n' "$*"
}

die() {
    printf '[install-zoekt] ERROR: %s\n' "$*" >&2
    exit 1
}

detect_platform() {
    local os arch
    os="$(uname -s)"
    arch="$(uname -m)"

    case "${os}" in
        Darwin) os="darwin" ;;
        Linux)  os="linux" ;;
        *)      die "Unsupported OS: ${os}" ;;
    esac

    case "${arch}" in
        arm64 | aarch64) arch="arm64" ;;
        x86_64)          arch="amd64" ;;
        *)               die "Unsupported architecture: ${arch}" ;;
    esac

    log "Detected platform: ${os}-${arch}"
    # Export for potential use by downstream scripts or future pre-built download logic
    export PLATFORM_OS="${os}"
    export PLATFORM_ARCH="${arch}"
}

check_already_installed() {
    local all_present=true
    for cmd in "${ZOEKT_CMDS[@]}"; do
        if [[ ! -x "${BIN_DIR}/${cmd}" ]]; then
            all_present=false
            break
        fi
    done

    if [[ "${all_present}" == "true" ]]; then
        # Binaries exist and are executable — check if they work
        if "${BIN_DIR}/zoekt-index" --help >/dev/null 2>&1; then
            log "zoekt binaries already installed and working in ${BIN_DIR}"
            if [[ "${ZOEKT_VERSION}" == "latest" ]]; then
                log "Version pinning not requested; skipping re-install."
                return 0
            fi
            log "Version pin requested (${ZOEKT_VERSION}); will re-install to ensure correct version."
            return 1
        fi
        log "Existing binaries found but not functional; will re-install."
        return 1
    fi
    return 1
}

build_from_source() {
    if ! command -v go >/dev/null 2>&1; then
        die "Go is not installed. Install Go (https://go.dev/dl/) and try again."
    fi

    local go_version
    go_version="$(go version)"
    log "Found Go: ${go_version}"

    mkdir -p "${BIN_DIR}"

    local version_suffix="@${ZOEKT_VERSION}"

    for cmd in "${ZOEKT_CMDS[@]}"; do
        log "Installing ${cmd}${version_suffix} ..."
        GOBIN="${BIN_DIR}" go install "${ZOEKT_MODULE}/cmd/${cmd}${version_suffix}"
    done

    log "Built zoekt binaries from source."
}

verify_install() {
    local failed=false
    for cmd in "${ZOEKT_CMDS[@]}"; do
        if [[ ! -x "${BIN_DIR}/${cmd}" ]]; then
            log "FAIL: ${BIN_DIR}/${cmd} not found or not executable"
            failed=true
        fi
    done

    if [[ "${failed}" == "true" ]]; then
        die "Verification failed: one or more binaries missing."
    fi

    # Smoke test
    if "${BIN_DIR}/zoekt-index" --help >/dev/null 2>&1; then
        log "Verification passed: zoekt-index --help succeeded."
    else
        die "Verification failed: zoekt-index --help returned non-zero."
    fi

    log "Installed zoekt to ${BIN_DIR}/"
}

main() {
    log "Starting zoekt installation (version=${ZOEKT_VERSION})"
    detect_platform

    if check_already_installed; then
        exit 0
    fi

    log "Building from source (sourcegraph/zoekt publishes no pre-built binaries)..."
    build_from_source

    verify_install
    log "Done."
}

main "$@"
