#!/usr/bin/env bash
set -euo pipefail

KIND_BIN=${KIND_BIN:-kind}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-kocao-dev}
CONFIG_FILE=${CONFIG_FILE:-hack/kind/kind-config.yaml}

if ! command -v "$KIND_BIN" >/dev/null 2>&1; then
	echo "kind not found: $KIND_BIN" >&2
	exit 1
fi

if "$KIND_BIN" get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
	echo "kind cluster already exists: ${KIND_CLUSTER_NAME}"
	exit 0
fi

echo "creating kind cluster: ${KIND_CLUSTER_NAME}"
"$KIND_BIN" create cluster --name "$KIND_CLUSTER_NAME" --config "$CONFIG_FILE"

if command -v kubectl >/dev/null 2>&1; then
	echo "waiting for cluster nodes to be Ready"
	kubectl wait --for=condition=Ready nodes --all --timeout=180s
fi
