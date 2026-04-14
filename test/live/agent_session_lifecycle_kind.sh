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

read -r -d '' ACP_FIXTURE_SCRIPT <<'EOF' || true
import json
import queue
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

SESSION_ID = "live-sas-123"
history = []
subscribers = []
lock = threading.Lock()


def broadcast(event):
    payload = json.dumps(event, separators=(",", ":"))
    with lock:
        history.append(payload)
        current = list(subscribers)
    for sub in current:
        sub.put(payload)


def close_streams():
    with lock:
        current = list(subscribers)
    for sub in current:
        sub.put(False)


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_POST(self):
        length = int(self.headers.get("Content-Length", "0"))
        body = self.rfile.read(length) if length else b"{}"
        env = json.loads(body.decode("utf-8"))
        method = env.get("method")
        req_id = env.get("id")

        if method == "initialize":
            response = {"jsonrpc": "2.0", "id": req_id, "result": {"authMethods": []}}
        elif method == "session/new":
            params = env.get("params") or {}
            broadcast({"jsonrpc": "2.0", "method": "session/new", "params": {"sessionId": SESSION_ID, "cwd": params.get("cwd", "/workspace/repo")}})
            response = {"jsonrpc": "2.0", "id": req_id, "result": {"sessionId": SESSION_ID}}
        elif method == "session/prompt":
            params = env.get("params") or {}
            prompt_parts = params.get("prompt") or []
            prompt_text = ""
            if prompt_parts:
                prompt_text = str(prompt_parts[0].get("text", ""))
            broadcast({"jsonrpc": "2.0", "method": "session/update", "params": {"sessionId": SESSION_ID, "sessionUpdate": "user_message_chunk", "content": {"type": "text", "text": prompt_text}}})
            broadcast({"jsonrpc": "2.0", "method": "session/update", "params": {"sessionId": SESSION_ID, "sessionUpdate": "agent_message_chunk", "content": {"type": "text", "text": f"echo: {prompt_text}"}}})
            response = {"jsonrpc": "2.0", "id": req_id, "result": {"stopReason": "completed"}}
        else:
            response = {"jsonrpc": "2.0", "id": req_id, "result": {}}

        payload = json.dumps(response, separators=(",", ":")).encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def do_GET(self):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "keep-alive")
        self.end_headers()

        sub = queue.Queue()
        with lock:
            snapshot = list(history)
            subscribers.append(sub)
        try:
            for payload in snapshot:
                self.wfile.write(f"data: {payload}\n\n".encode("utf-8"))
            self.wfile.flush()
            while True:
                try:
                    payload = sub.get(timeout=1)
                except queue.Empty:
                    payload = None
                if payload is None:
                    self.wfile.write(b": keepalive\n\n")
                    self.wfile.flush()
                    continue
                if payload is False:
                    break
                self.wfile.write(f"data: {payload}\n\n".encode("utf-8"))
                self.wfile.flush()
        except (BrokenPipeError, ConnectionResetError):
            pass
        finally:
            with lock:
                if sub in subscribers:
                    subscribers.remove(sub)

    def do_DELETE(self):
        close_streams()
        payload = b"{}"
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(payload)))
        self.end_headers()
        self.wfile.write(payload)

    def log_message(self, format, *args):
        return


ThreadingHTTPServer(("0.0.0.0", 2468), Handler).serve_forever()
EOF

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

wait_for_api_service_endpoint() {
	for _ in $(seq 1 60); do
		if kubectl -n "${K8S_NAMESPACE}" get endpoints control-plane-api -o json | jq -e '([.subsets[]?.addresses[]?] | length) > 0' >/dev/null 2>&1; then
			return 0
		fi
		sleep 2
	done
	return 1
}

start_port_forward() {
	stop_port_forward
	if ! wait_for_api_service_endpoint; then
		kubectl -n "${K8S_NAMESPACE}" get endpoints control-plane-api -o yaml >&2 || true
		fail "control-plane-api service never published a ready endpoint"
	fi
	PORT_FORWARD_LOG=$(mktemp -t kocao-port-forward.XXXXXX.log)
	kubectl -n "${K8S_NAMESPACE}" port-forward service/control-plane-api 18080:80 >"${PORT_FORWARD_LOG}" 2>&1 &
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

wait_for_active_session() {
	local run_id=$1
	local status_file=$2
	for _ in $(seq 1 90); do
		trigger_agent_session_create "${run_id}"
		if "${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent status "${run_id}" --output json >"${status_file}" 2>/dev/null; then
			if jq -e '.phase == "Ready" or .phase == "Running"' "${status_file}" >/dev/null; then
				return 0
			fi
		fi
		sleep 2
	done
	return 1
}

wait_for_run_pod_port() {
	local run_id=$1
	local port=${2:-2468}
	for _ in $(seq 1 90); do
		local pod_name
		pod_name=$(kubectl -n "${K8S_NAMESPACE}" get pods -l "kocao.withakay.github.com/run=${run_id}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
		if [[ -n "${pod_name}" ]] && kubectl -n "${K8S_NAMESPACE}" exec "${pod_name}" -- python3 -c "import socket; s=socket.socket(); s.settimeout(1); raise SystemExit(0 if s.connect_ex(('127.0.0.1', ${port})) == 0 else 1)" >/dev/null 2>&1; then
			return 0
		fi
		sleep 2
	done
	return 1
}

wait_for_logs_contains() {
	local run_id=$1
	local output_file=$2
	local pattern=$3
	for _ in $(seq 1 60); do
		if "${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent logs "${run_id}" --output json >"${output_file}" 2>/dev/null; then
			if grep -q -- "${pattern}" "${output_file}"; then
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
RUN_SUFFIX=$(date +%s)-$$
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
kubectl -n "${K8S_NAMESPACE}" set env deploy/control-plane-api KOCAO_ALLOW_MOCK_AGENT_FIXTURE=1 >/dev/null
kubectl -n "${K8S_NAMESPACE}" rollout restart deploy/control-plane-api deploy/control-plane-operator
make -C "${ROOT_DIR}" deploy-wait

log "starting API port-forward"
start_port_forward

tmpdir=${TMPDIR_PATH}
status_json="${tmpdir}/status.json"
list_json="${tmpdir}/list.json"
logs_jsonl="${tmpdir}/logs.jsonl"
exec_json="${tmpdir}/exec.json"
stop_json="${tmpdir}/stop.json"
repeat_stop_json="${tmpdir}/stop-repeat.json"
diag_status_json="${tmpdir}/diag-status.json"
diag_list_json="${tmpdir}/diag-list.json"
diag_api_json="${tmpdir}/diag-api.json"
workspace_json="${tmpdir}/workspace.json"
run_json="${tmpdir}/run.json"

START_STDERR="${tmpdir}/start.stderr"

log "creating healthy ACP fixture harness run"
api_request POST /api/v1/workspace-sessions "$(jq -nc --arg repo "${KOCAO_LIVE_TEST_REPO_URL}" --arg display_name "live-ci-healthy-${RUN_SUFFIX}" '{displayName:$display_name,repoURL:$repo}')" >"${workspace_json}"
workspace_id=$(json_field "${workspace_json}" '.id')

api_request POST "/api/v1/workspace-sessions/${workspace_id}/harness-runs" "$(jq -nc \
	--arg repo "${KOCAO_LIVE_TEST_REPO_URL}" \
	--arg image "${HARNESS_IMAGE}:${IMAGE_TAG}" \
	--arg agent "${KOCAO_LIVE_TEST_AGENT}" \
	--arg script "${ACP_FIXTURE_SCRIPT}" \
	'{repoURL:$repo,repoRevision:"main",image:$image,egressMode:"full",command:["python3","-u","-c"],args:[$script],agentSession:{agent:$agent}}')" >"${run_json}"
run_id=$(json_field "${run_json}" '.id')

if ! wait_for_run_pod_port "${run_id}" 2468; then
	kubectl -n "${K8S_NAMESPACE}" get pods -o wide >&2 || true
	kubectl -n "${K8S_NAMESPACE}" logs "$(kubectl -n "${K8S_NAMESPACE}" get pods -l "kocao.withakay.github.com/run=${run_id}" -o jsonpath='{.items[0].metadata.name}')" --tail=200 >&2 || true
	fail "fixture-backed run never opened ACP port 2468"
fi

if ! wait_for_active_session "${run_id}" "${status_json}"; then
	api_request GET "/api/v1/harness-runs/${run_id}/agent-session" >"${diag_api_json}" || true
	kubectl -n "${K8S_NAMESPACE}" get pods -o wide >&2 || true
	kubectl -n "${K8S_NAMESPACE}" logs deploy/control-plane-api -c api --tail=200 >&2 || true
	kubectl -n "${K8S_NAMESPACE}" logs "$(kubectl -n "${K8S_NAMESPACE}" get pods -l "kocao.withakay.github.com/run=${run_id}" -o jsonpath='{.items[0].metadata.name}')" --tail=200 >&2 || true
	fail "timed out waiting for healthy agent lifecycle"
fi

assert_jq "${status_json}" --arg run_id "${run_id}" '.runId == $run_id and (.phase == "Ready" or .phase == "Running") and .agent == "mock" and (.sessionId | length > 0)' 'agent status did not surface healthy lifecycle state'

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent list --output json >"${list_json}"
assert_jq "${list_json}" --arg run_id "${run_id}" 'map(select(.runId == $run_id and (.phase == "Ready" or .phase == "Running") and .agent == "mock" and (.sessionId | length > 0))) | length == 1' 'agent list did not include healthy mock session'

log "sending prompt through live agent session"
"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent exec "${run_id}" --prompt 'hello live kind' --output json >"${exec_json}"
assert_jq "${exec_json}" --arg prompt 'hello live kind' '.events | length > 0 and any(.[]; (.data | tostring | contains($prompt)) or (.data | tostring | contains("echo: " + $prompt)) or (.data.stopReason? == "completed"))' 'agent exec did not return prompt completion or response events'

log "verifying live agent logs for session creation and prompt traffic"
if ! wait_for_logs_contains "${run_id}" "${logs_jsonl}" 'session/new'; then
	fail 'agent logs did not include healthy session/new traffic'
fi
if ! wait_for_logs_contains "${run_id}" "${logs_jsonl}" 'hello live kind'; then
	fail 'agent logs did not include prompt traffic from healthy session'
fi

log "restarting control-plane deployments to verify healthy lifecycle durability"
kubectl -n "${K8S_NAMESPACE}" rollout restart deploy/control-plane-api deploy/control-plane-operator
make -C "${ROOT_DIR}" deploy-wait
start_port_forward

if ! wait_for_active_session "${run_id}" "${status_json}"; then
	api_request GET "/api/v1/harness-runs/${run_id}/agent-session" >"${diag_api_json}" || true
	kubectl -n "${K8S_NAMESPACE}" get pods -o wide >&2 || true
	kubectl -n "${K8S_NAMESPACE}" logs deploy/control-plane-api -c api --tail=200 >&2 || true
	fail "healthy agent lifecycle did not survive control-plane restart"
fi
assert_jq "${status_json}" --arg run_id "${run_id}" '.runId == $run_id and (.phase == "Ready" or .phase == "Running") and (.sessionId | length > 0)' 'healthy agent status did not survive control-plane restart'

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent list --output json >"${list_json}"
assert_jq "${list_json}" --arg run_id "${run_id}" 'map(select(.runId == $run_id and (.phase == "Ready" or .phase == "Running") and (.sessionId | length > 0))) | length == 1' 'agent list did not preserve healthy session after restart'

if ! wait_for_logs_contains "${run_id}" "${logs_jsonl}" 'hello live kind'; then
	fail 'agent logs did not preserve healthy prompt traffic after restart'
fi

log "stopping session and asserting terminal lifecycle"
"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent stop "${run_id}" --json >"${stop_json}"
assert_jq "${stop_json}" '.status == "stopped" and .session.phase == "Completed"' 'stop did not complete healthy lifecycle view'

"${ROOT_DIR}/bin/kocao" --api-url "${KOCAO_API_URL}" --token "${KOCAO_TOKEN}" agent stop "${run_id}" --json >"${repeat_stop_json}"
assert_jq "${repeat_stop_json}" '.status == "stopped" and .session.phase == "Completed"' 'repeat stop was not idempotent for healthy session'

log "creating bad-image run for provisioning diagnostics"
api_request POST /api/v1/workspace-sessions "$(jq -nc --arg repo "${KOCAO_LIVE_TEST_REPO_URL}" --arg display_name "live-ci-diagnostics-${RUN_SUFFIX}" '{displayName:$display_name,repoURL:$repo}')" >"${workspace_json}"
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
