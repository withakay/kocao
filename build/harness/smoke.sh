#!/usr/bin/env bash
set -euo pipefail

# Smoke test for the harness runtime image.
# Reads runtime-matrix.json and validates every baked runtime and tool.
# Exits non-zero with a clear report on any missing or unreachable tool.

matrix=${1:-/etc/kocao/runtime-matrix.json}

if [[ ! -f "${matrix}" ]]; then
  echo "FAIL: runtime matrix not found: ${matrix}" >&2
  exit 1
fi

fail=0

echo "=== Validating runtimes ==="

# Check each runtime is installed and reachable.
for runtime in $(jq -r '.runtimes | keys[]' "${matrix}"); do
  check_cmd=$(jq -r ".runtimes[\"${runtime}\"].check" "${matrix}")
  versions=$(jq -r ".runtimes[\"${runtime}\"].versions[]" "${matrix}")

  # Verify the default version is reachable.
  if ! eval "${check_cmd}" >/dev/null 2>&1; then
    echo "FAIL: ${runtime} — default not reachable (${check_cmd})" >&2
    fail=1
  else
    echo "  ok: ${runtime} default — $(eval "${check_cmd}" 2>&1 | head -1)"
  fi

  # Verify each pinned version is installed in mise.
  for ver in ${versions}; do
    if ! mise where "${runtime}@${ver}" >/dev/null 2>&1; then
      echo "FAIL: ${runtime}@${ver} — not installed in mise" >&2
      fail=1
    else
      echo "  ok: ${runtime}@${ver} — installed at $(mise where "${runtime}@${ver}")"
    fi
  done
done

echo ""
echo "=== Validating tools ==="

# Check each CLI tool is on PATH and executable.
for tool in $(jq -r '.tools | keys[]' "${matrix}"); do
  check_cmd=$(jq -r ".tools[\"${tool}\"]" "${matrix}")

  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "FAIL: ${tool} — not found on PATH" >&2
    fail=1
  elif ! eval "${check_cmd}" >/dev/null 2>&1; then
    echo "FAIL: ${tool} — found but check failed (${check_cmd})" >&2
    fail=1
  else
    echo "  ok: ${tool} — $(eval "${check_cmd}" 2>&1 | head -1)"
  fi
done

echo ""
if [[ "${fail}" -ne 0 ]]; then
  echo "FAIL: one or more checks failed" >&2
  exit 1
fi

echo "ok: all checks passed"
