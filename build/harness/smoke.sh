#!/usr/bin/env bash
set -euo pipefail

# Smoke test for a harness runtime image profile.
# Reads stable in-image contract files and validates the preserved sandbox-agent
# surface plus the profile-specific runtime matrix.

matrix=${1:-/etc/kocao/runtime-matrix.json}
profile=${2:-/etc/kocao/harness-profile.json}

if [[ ! -f "${matrix}" ]]; then
  echo "FAIL: runtime matrix not found: ${matrix}" >&2
  exit 1
fi

if [[ ! -f "${profile}" ]]; then
  echo "FAIL: harness profile metadata not found: ${profile}" >&2
  exit 1
fi

fail=0

echo "=== Validating harness contract ==="

workspace_dir=$(jq -r '.workspaceDir' "${profile}")
required_uid=$(jq -r '.runAsUser.uid' "${profile}")
required_gid=$(jq -r '.runAsUser.gid' "${profile}")

if [[ "$(id -u)" != "${required_uid}" ]]; then
  echo "FAIL: expected uid ${required_uid}, got $(id -u)" >&2
  fail=1
else
  echo "  ok: running as uid ${required_uid}"
fi

if [[ "$(id -g)" != "${required_gid}" ]]; then
  echo "FAIL: expected gid ${required_gid}, got $(id -g)" >&2
  fail=1
else
  echo "  ok: running as gid ${required_gid}"
fi

if [[ "$(pwd)" != "${workspace_dir}" ]]; then
  echo "FAIL: expected workdir ${workspace_dir}, got $(pwd)" >&2
  fail=1
else
  echo "  ok: workdir is ${workspace_dir}"
fi

for required_file in $(jq -r '.requiredFiles[]' "${profile}"); do
  if [[ ! -e "${required_file}" ]]; then
    echo "FAIL: required file missing: ${required_file}" >&2
    fail=1
  else
    echo "  ok: required file present: ${required_file}"
  fi
done

for required_tool in $(jq -r '.requiredTools[]' "${profile}"); do
  if ! command -v "${required_tool}" >/dev/null 2>&1; then
    echo "FAIL: required tool missing: ${required_tool}" >&2
    fail=1
  else
    echo "  ok: required tool present: ${required_tool}"
  fi
done

echo ""
echo "=== Validating runtimes ==="

for runtime in $(jq -r '.runtimes | keys[]' "${matrix}"); do
  check_cmd=$(jq -r ".runtimes[\"${runtime}\"].check" "${matrix}")
  versions=$(jq -r ".runtimes[\"${runtime}\"].versions[]" "${matrix}")

  if ! eval "${check_cmd}" >/dev/null 2>&1; then
    echo "FAIL: ${runtime} - default not reachable (${check_cmd})" >&2
    fail=1
  else
    echo "  ok: ${runtime} default - $(eval "${check_cmd}" 2>&1 | head -1)"
  fi

  for ver in ${versions}; do
    if ! mise where "${runtime}@${ver}" >/dev/null 2>&1; then
      echo "FAIL: ${runtime}@${ver} - not installed in mise" >&2
      fail=1
    else
      echo "  ok: ${runtime}@${ver} - installed at $(mise where "${runtime}@${ver}")"
    fi
  done
done

echo ""
echo "=== Validating tools ==="

for tool in $(jq -r '.tools | keys[]' "${matrix}"); do
  check_cmd=$(jq -r ".tools[\"${tool}\"]" "${matrix}")

  if ! command -v "${tool}" >/dev/null 2>&1; then
    echo "FAIL: ${tool} - not found on PATH" >&2
    fail=1
  elif ! eval "${check_cmd}" >/dev/null 2>&1; then
    echo "FAIL: ${tool} - found but check failed (${check_cmd})" >&2
    fail=1
  else
    echo "  ok: ${tool} - $(eval "${check_cmd}" 2>&1 | head -1)"
  fi
done

echo ""
echo "=== Validating sandbox-agent API contract ==="

sandbox_host=127.0.0.1
sandbox_port=2468
sandbox_health_endpoint=$(jq -r '.healthEndpoint' "${profile}")
sandbox_catalog_endpoint=$(jq -r '.agentCatalogEndpoint' "${profile}")
sandbox_report_agents=$(jq -r '.requiredAgents[]' "${profile}")
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
  if curl -fsS "http://${sandbox_host}:${sandbox_port}${sandbox_health_endpoint}" >/dev/null 2>&1; then
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
  agents_json="$(curl -fsS "http://${sandbox_host}:${sandbox_port}${sandbox_catalog_endpoint}" 2>/dev/null || true)"
  for required_agent in $(jq -r '.requiredAgents[]' "${profile}"); do
    if [[ -z "${agents_json}" ]] || ! jq -e --arg agent "${required_agent}" '.agents[]? | select(.id == $agent)' >/dev/null <<<"${agents_json}"; then
      echo "FAIL: sandbox-agent catalog missing ${required_agent}" >&2
      fail=1
    else
      echo "  ok: sandbox-agent catalog includes ${required_agent}"
    fi
  done

  report_json="$(sandbox-agent api agents report --endpoint "http://${sandbox_host}:${sandbox_port}" 2>/dev/null || true)"
  for report_agent in ${sandbox_report_agents}; do
    if [[ -z "${report_json}" ]] || ! jq -e --arg agent "${report_agent}" '.agents[]? | select(.id == $agent and .installed == true)' >/dev/null <<<"${report_json}"; then
      echo "FAIL: sandbox-agent ${report_agent} agent is not installed" >&2
      fail=1
    else
      echo "  ok: sandbox-agent ${report_agent} agent is installed"
    fi
  done
fi

echo ""
if [[ "${fail}" -ne 0 ]]; then
  echo "FAIL: one or more checks failed" >&2
  exit 1
fi

echo "ok: all checks passed"
