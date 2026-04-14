#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(git rev-parse --show-toplevel)

LOCALBIN=${LOCALBIN:-"${ROOT_DIR}/.local/bin"}
KIND=${KIND:-"${LOCALBIN}/kind"}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-kocao-live-ci}
K8S_NAMESPACE=${K8S_NAMESPACE:-kocao-system}

API_IMAGE=${API_IMAGE:-kocao/control-plane-api}
OPERATOR_IMAGE=${OPERATOR_IMAGE:-kocao/control-plane-operator}
HARNESS_IMAGE=${HARNESS_IMAGE:-kocao/harness-runtime}
SIDECAR_IMAGE=${SIDECAR_IMAGE:-kocao/kocao-sidecar}
IMAGE_TAG=${IMAGE_TAG:-dev}

KOCAO_API_URL=${KOCAO_API_URL:-http://127.0.0.1:18080}
KOCAO_TOKEN=${KOCAO_TOKEN:-dev-bootstrap}
KOCAO_LIVE_TEST_REPO_URL=${KOCAO_LIVE_TEST_REPO_URL:-https://github.com/withakay/kocao}
KOCAO_LIVE_TEST_AGENT=${KOCAO_LIVE_TEST_AGENT:-mock}
KOCAO_LIVE_TEST_BAD_IMAGE=${KOCAO_LIVE_TEST_BAD_IMAGE:-kocao/harness-runtime:missing-live-ci}

PORT_FORWARD_PID=""
PORT_FORWARD_LOG=""
START_STDERR=""
TMPDIR_PATH=""

cleanup() {
	set +e
	if [[ -n "${PORT_FORWARD_PID}" ]] && kill -0 "${PORT_FORWARD_PID}" >/dev/null 2>&1; then
		kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
		wait "${PORT_FORWARD_PID}" 2>/dev/null || true
	fi
	if [[ -n "${PORT_FORWARD_LOG}" ]]; then
		rm -f "${PORT_FORWARD_LOG}"
	fi
	if [[ -n "${START_STDERR}" ]]; then
		rm -f "${START_STDERR}"
	fi
	if [[ -n "${TMPDIR_PATH}" ]]; then
		rm -rf "${TMPDIR_PATH}"
	fi
	if [[ "${KOCAO_LIVE_TEST_KEEP_CLUSTER:-0}" != "1" ]]; then
		make -C "${ROOT_DIR}" kind-down KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME}" >/dev/null 2>&1 || true
	fi
}
trap cleanup EXIT

log() {
	printf '[agent-live-kind] %s\n' "$*"
}

fail() {
	printf '[agent-live-kind] ERROR: %s\n' "$*" >&2
	exit 1
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

json_field() {
	local file=$1
	local expr=$2
	jq -er "${expr}" "${file}"
}

assert_jq() {
	local file=$1
	shift
	local message="${@: -1}"
	local jq_args=("${@:1:$(($#-1))}")
	jq -e "${jq_args[@]}" "${file}" >/dev/null || fail "${message}: $(tr -d '\n' <"${file}")"
}

wait_for_http_ok() {
	local url=$1
	local attempts=${2:-60}
	for _ in $(seq 1 "${attempts}"); do
		if curl -fsS "${url}" >/dev/null 2>&1; then
			return 0
		fi
		sleep 2
	done
	return 1
}

start_port_forward() {
	stop_port_forward
	local pod_name
	pod_name=$(kubectl -n "${K8S_NAMESPACE}" get pods -l app=control-plane-api -o jsonpath='{.items[0].metadata.name}')
	[[ -n "${pod_name}" ]] || fail "control-plane-api pod not found"
	PORT_FORWARD_LOG=$(mktemp -t kocao-port-forward.XXXXXX.log)
	kubectl -n "${K8S_NAMESPACE}" port-forward "pod/${pod_name}" 18080:8080 >"${PORT_FORWARD_LOG}" 2>&1 &
	PORT_FORWARD_PID=$!
	if ! wait_for_http_ok "${KOCAO_API_URL}/healthz" 60; then
		cat "${PORT_FORWARD_LOG}" >&2 || true
		fail "control-plane API did not become reachable via port-forward"
	fi
}

stop_port_forward() {
	if [[ -n "${PORT_FORWARD_PID}" ]] && kill -0 "${PORT_FORWARD_PID}" >/dev/null 2>&1; then
		kill "${PORT_FORWARD_PID}" >/dev/null 2>&1 || true
		wait "${PORT_FORWARD_PID}" 2>/dev/null || true
	fi
	PORT_FORWARD_PID=""
	if [[ -n "${PORT_FORWARD_LOG}" ]]; then
		rm -f "${PORT_FORWARD_LOG}"
		PORT_FORWARD_LOG=""
	fi
}

api_request() {
	local method=$1
	local path=$2
	local body=${3:-}
	if [[ -n "${body}" ]]; then
		curl -fsS -X "${method}" "${KOCAO_API_URL}${path}" \
			-H "Authorization: Bearer ${KOCAO_TOKEN}" \
			-H 'Content-Type: application/json' \
			-d "${body}"
	else
		curl -fsS -X "${method}" "${KOCAO_API_URL}${path}" \
			-H "Authorization: Bearer ${KOCAO_TOKEN}"
	fi
}

trigger_agent_session_create() {
	local run_id=$1
	curl -sS -o /dev/null -X POST "${KOCAO_API_URL}/api/v1/harness-runs/${run_id}/agent-session" \
		-H "Authorization: Bearer ${KOCAO_TOKEN}" || true
}

wait_for_image_pull_diagnostic() {
	local run_id=$1
	local status_file=$2
	for _ in $(seq 1 90); do
		if "${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent status "${run_id}" --output json >"${status_file}" 2>/dev/null; then
			if jq -e '.phase == "Provisioning" and .diagnostic.class == "image-pull" and (.diagnostic.summary | length > 0)' "${status_file}" >/dev/null; then
				return 0
			fi
		fi
		sleep 2
	done
	return 1
}

wait_for_failed_session() {
	local run_id=$1
	local status_file=$2
	for _ in $(seq 1 90); do
		trigger_agent_session_create "${run_id}"
		if "${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent status "${run_id}" --output json >"${status_file}" 2>/dev/null; then
			if jq -e '.phase == "Failed"' "${status_file}" >/dev/null; then
				return 0
			fi
		fi
		sleep 2
	done
	return 1
}

require_command docker
require_command curl
require_command jq
require_command kubectl
require_command go

TMPDIR_PATH=$(mktemp -d -t kocao-agent-live.XXXXXX)
kind_config="${TMPDIR_PATH}/kind-config.yaml"
cat >"${kind_config}" <<'EOF'
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
EOF

log "building CLI binary"
make -C "${ROOT_DIR}" build-cli

log "creating kind cluster"
CONFIG_FILE="${kind_config}" make -C "${ROOT_DIR}" kind-up KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME}"

log "building live-agent images"
make -C "${ROOT_DIR}" images-live-agent IMAGE_TAG="${IMAGE_TAG}"

log "loading images into kind"
make -C "${ROOT_DIR}" kind-load-images-live-agent KIND_CLUSTER_NAME="${KIND_CLUSTER_NAME}" IMAGE_TAG="${IMAGE_TAG}"

log "deploying control plane"
make -C "${ROOT_DIR}" deploy
make -C "${ROOT_DIR}" deploy-wait

log "starting API port-forward"
start_port_forward

tmpdir=${TMPDIR_PATH}
status_json="${tmpdir}/status.json"
list_json="${tmpdir}/list.json"
logs_jsonl="${tmpdir}/logs.jsonl"
stop_json="${tmpdir}/stop.json"
repeat_stop_json="${tmpdir}/stop-repeat.json"
diag_status_json="${tmpdir}/diag-status.json"
diag_list_json="${tmpdir}/diag-list.json"
diag_api_json="${tmpdir}/diag-api.json"
workspace_json="${tmpdir}/workspace.json"
run_json="${tmpdir}/run.json"

START_STDERR="${tmpdir}/start.stderr"

log "creating live mock-backed harness run"
api_request POST /api/v1/workspace-sessions "$(jq -nc --arg repo "${KOCAO_LIVE_TEST_REPO_URL}" '{displayName:"live-ci-bootstrap",repoURL:$repo}')" >"${workspace_json}"
workspace_id=$(json_field "${workspace_json}" '.id')

api_request POST "/api/v1/workspace-sessions/${workspace_id}/harness-runs" "$(jq -nc --arg repo "${KOCAO_LIVE_TEST_REPO_URL}" --arg image "${HARNESS_IMAGE}:${IMAGE_TAG}" --arg agent "${KOCAO_LIVE_TEST_AGENT}" '{repoURL:$repo,repoRevision:"main",image:$image,egressMode:"full",agentSession:{agent:$agent}}')" >"${run_json}"
run_id=$(json_field "${run_json}" '.id')

log "waiting for bootstrap session lifecycle to surface"
if ! wait_for_failed_session "${run_id}" "${status_json}"; then
	api_request GET "/api/v1/harness-runs/${run_id}/agent-session" >"${diag_api_json}" || true
	kubectl -n "${K8S_NAMESPACE}" get pods -o wide >&2 || true
	kubectl -n "${K8S_NAMESPACE}" logs deploy/control-plane-api -c api --tail=200 >&2 || true
	fail "timed out waiting for failed bootstrap lifecycle"
fi

assert_jq "${status_json}" --arg run_id "${run_id}" '.runId == $run_id and .phase == "Failed" and .agent == "mock"' 'status did not surface failed bootstrap session'

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent list --output json >"${list_json}"
assert_jq "${list_json}" --arg run_id "${run_id}" 'map(select(.runId == $run_id and .phase == "Failed" and .agent == "mock")) | length == 1' 'agent list did not include failed bootstrap session'

log "verifying live ACP event logs"
"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent logs "${run_id}" --output json >"${logs_jsonl}"
grep -q 'session/new' "${logs_jsonl}" || fail 'agent logs did not include bootstrap session/new traffic'

log "restarting control-plane deployments to verify failed lifecycle durability"
kubectl -n "${K8S_NAMESPACE}" rollout restart deploy/control-plane-api deploy/control-plane-operator
make -C "${ROOT_DIR}" deploy-wait
start_port_forward

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent status "${run_id}" --output json >"${status_json}"
assert_jq "${status_json}" --arg run_id "${run_id}" '.runId == $run_id and .phase == "Failed"' 'failed bootstrap status did not survive control-plane restart'

log "stopping session and asserting terminal lifecycle"
"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent stop "${run_id}" --json >"${stop_json}"
assert_jq "${stop_json}" '.status == "stopped" and .session.phase == "Failed"' 'stop did not preserve failed lifecycle view'

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent stop "${run_id}" --json >"${repeat_stop_json}"
assert_jq "${repeat_stop_json}" '.status == "stopped" and .session.phase == "Failed"' 'repeat stop was not idempotent'

log "creating bad-image run for provisioning diagnostics"
api_request POST /api/v1/workspace-sessions "$(jq -nc --arg repo "${KOCAO_LIVE_TEST_REPO_URL}" '{displayName:"live-ci-diagnostics",repoURL:$repo}')" >"${workspace_json}"
workspace_id=$(json_field "${workspace_json}" '.id')

api_request POST "/api/v1/workspace-sessions/${workspace_id}/harness-runs" "$(jq -nc --arg repo "${KOCAO_LIVE_TEST_REPO_URL}" --arg image "${KOCAO_LIVE_TEST_BAD_IMAGE}" '{repoURL:$repo,repoRevision:"main",image:$image,egressMode:"full",agentSession:{agent:"mock"}}')" >"${run_json}"
diag_run_id=$(json_field "${run_json}" '.id')

if ! wait_for_image_pull_diagnostic "${diag_run_id}" "${diag_status_json}"; then
	api_request GET "/api/v1/harness-runs/${diag_run_id}/agent-session" >"${diag_api_json}" || true
	kubectl -n "${K8S_NAMESPACE}" get pods -o wide >&2 || true
	kubectl -n "${K8S_NAMESPACE}" describe harnessrun "${diag_run_id}" >&2 || true
	fail "timed out waiting for image-pull diagnostic"
fi

assert_jq "${diag_status_json}" '.diagnostic.class == "image-pull" and (.diagnostic.summary | length > 0) and (.diagnostic.detail | length > 0)' 'status did not expose image-pull diagnostic'
api_request GET "/api/v1/harness-runs/${diag_run_id}/agent-session" >"${diag_api_json}"
assert_jq "${diag_api_json}" '.phase == "Provisioning" and .diagnostic.class == "image-pull"' 'API did not expose provisioning diagnostic'

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent list --output json >"${diag_list_json}"
assert_jq "${diag_list_json}" --arg run_id "${diag_run_id}" 'map(select(.runId == $run_id and .phase == "Provisioning" and .diagnostic.class == "image-pull")) | length == 1' 'agent list did not expose provisioning diagnostic'

log "live kind lifecycle verification passed"
