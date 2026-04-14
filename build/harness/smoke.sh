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
echo "=== Validating sandbox-agent API contract ==="

sandbox_host=127.0.0.1
sandbox_port=2468
sandbox_log="$(mktemp -t sandbox-agent-smoke.XXXXXX.log)"
sandbox_pid=""
cleanup() {
  if [[ -n "${sandbox_pid}" ]] && kill -0 "${sandbox_pid}" >/dev/null 2>&1; then
    kill "${sandbox_pid}" >/dev/null 2>&1 || true
    wait "${sandbox_pid}" 2>/dev/null || true
  fi
  rm -f "${sandbox_log}"
}
trap cleanup EXIT

sandbox-agent server --no-token --host 0.0.0.0 --port "${sandbox_port}" >"${sandbox_log}" 2>&1 &
sandbox_pid=$!

health_ok=0
for _ in $(seq 1 30); do
  if curl -fsS "http://${sandbox_host}:${sandbox_port}/v1/health" >/dev/null 2>&1; then
    health_ok=1
    break
  fi
  if ! kill -0 "${sandbox_pid}" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

if [[ "${health_ok}" -ne 1 ]]; then
  echo "FAIL: sandbox-agent health endpoint did not become ready" >&2
  if [[ -f "${sandbox_log}" ]]; then
    sed -n '1,120p' "${sandbox_log}" >&2 || true
  fi
  fail=1
else
  echo "  ok: sandbox-agent health endpoint is reachable"
fi

if [[ "${health_ok}" -eq 1 ]]; then
  agents_json="$(curl -fsS "http://${sandbox_host}:${sandbox_port}/v1/agents" 2>/dev/null || true)"
  for required_agent in claude codex mock opencode pi; do
    if [[ -z "${agents_json}" ]] || ! jq -e --arg agent "${required_agent}" '.agents[]? | select(.id == $agent)' >/dev/null <<<"${agents_json}"; then
      echo "FAIL: sandbox-agent catalog missing ${required_agent}" >&2
      fail=1
    else
      echo "  ok: sandbox-agent catalog includes ${required_agent}"
    fi
  done

  report_json="$(sandbox-agent api agents report --endpoint "http://${sandbox_host}:${sandbox_port}" 2>/dev/null || true)"
  if [[ -z "${report_json}" ]] || ! jq -e '.agents[]? | select(.id == "mock" and .installed == true)' >/dev/null <<<"${report_json}"; then
    echo "FAIL: sandbox-agent mock agent is not installed" >&2
    fail=1
  else
    echo "  ok: sandbox-agent mock agent is installed"
  fi
fi

echo ""
if [[ "${fail}" -ne 0 ]]; then
  echo "FAIL: one or more checks failed" >&2
  exit 1
fi

echo "ok: all checks passed"
