#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<'EOF'
Usage: record-terminal.sh --output <path.cast> [OPTIONS] (--command <shell-command> | --script <path>)

Record a terminal workflow into an asciinema cast using repo-local defaults.

Options:
  --output <path.cast>       Output cast path (required)
  --title <text>             Cast title (default: basename of output)
  --workdir <path>           Working directory inside the recording (default: repo root)
  --window-size <COLSxROWS>  Terminal size for recording (default: 100x28)
  --idle-time-limit <secs>   Cap idle playback gaps (default: 1.25)
  --command <shell-command>  Record a single shell command with prompt echoing
  --script <path>            Source a recording script that calls run "..."
  --help                     Show this help

Script mode helpers:
  run "cmd"                 Print "$ cmd" then eval it
  note "text"               Print an unprefixed note line
  pause <seconds>            Sleep for pacing

Examples:
  ./demos/record-terminal.sh \
    --output demos/example.cast \
    --title "Example" \
    --command 'showboat verify demos/zoekt-search-demo.md'

  ./demos/record-terminal.sh \
    --output demos/zoekt-search-demo.cast \
    --title "Zoekt demo terminal walkthrough" \
    --script demos/zoekt-search-demo-recording.sh
EOF
}

die() {
    printf '[record-terminal] ERROR: %s\n' "$*" >&2
    exit 1
}

main() {
    local output=""
    local title=""
    local workdir
    local window_size="100x28"
    local idle_time_limit="1.25"
    local command_text=""
    local script_path=""
    local repo_root
    local runner

    repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
    workdir="${repo_root}"

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                [[ $# -ge 2 ]] || die "--output requires a value"
                output="$2"
                shift 2
                ;;
            --title)
                [[ $# -ge 2 ]] || die "--title requires a value"
                title="$2"
                shift 2
                ;;
            --workdir)
                [[ $# -ge 2 ]] || die "--workdir requires a value"
                workdir="$2"
                shift 2
                ;;
            --window-size)
                [[ $# -ge 2 ]] || die "--window-size requires a value"
                window_size="$2"
                shift 2
                ;;
            --idle-time-limit)
                [[ $# -ge 2 ]] || die "--idle-time-limit requires a value"
                idle_time_limit="$2"
                shift 2
                ;;
            --command)
                [[ $# -ge 2 ]] || die "--command requires a value"
                command_text="$2"
                shift 2
                ;;
            --script)
                [[ $# -ge 2 ]] || die "--script requires a value"
                script_path="$2"
                shift 2
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            *)
                die "Unknown argument: $1"
                ;;
        esac
    done

    [[ -n "${output}" ]] || die "--output is required"
    [[ -n "${command_text}" || -n "${script_path}" ]] || die "Provide --command or --script"
    [[ -z "${command_text}" || -z "${script_path}" ]] || die "Use only one of --command or --script"
    [[ "${output}" == *.cast ]] || die "Output path must end with .cast"

    if [[ -n "${script_path}" && ! -f "${script_path}" ]]; then
        die "Script not found: ${script_path}"
    fi

    if [[ ! -d "${workdir}" ]]; then
        die "Workdir does not exist: ${workdir}"
    fi

    if ! command -v asciinema >/dev/null 2>&1; then
        die "asciinema is not installed"
    fi

    mkdir -p "$(dirname "${output}")"

    if [[ -z "${title}" ]]; then
        title="$(basename "${output}" .cast)"
    fi

    runner="$(mktemp "${TMPDIR:-/tmp}/record-terminal.XXXXXX")"
    cat >"${runner}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

run() {
    local cmd="$1"
    printf '\n$ %s\n' "${cmd}"
    eval "${cmd}"
}

note() {
    printf '\n%s\n' "$*"
}

pause() {
    sleep "$1"
}

cd "${KOCAO_RECORD_WORKDIR}"

if [[ -n "${KOCAO_RECORD_SCRIPT:-}" ]]; then
    source "${KOCAO_RECORD_SCRIPT}"
else
    run "${KOCAO_RECORD_COMMAND}"
fi
EOF
    chmod +x "${runner}"
    trap 'rm -f '"'"'${runner}'"'"'' EXIT

    KOCAO_RECORD_WORKDIR="$(cd "${workdir}" && pwd)" \
    KOCAO_RECORD_SCRIPT="${script_path}" \
    KOCAO_RECORD_COMMAND="${command_text}" \
    asciinema record \
        --headless \
        --overwrite \
        --return \
        --idle-time-limit "${idle_time_limit}" \
        --window-size "${window_size}" \
        --title "${title}" \
        --command "bash \"${runner}\"" \
        "${output}"
}

main "$@"
