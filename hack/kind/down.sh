#!/usr/bin/env bash
set -euo pipefail

KIND_BIN=${KIND_BIN:-kind}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-kocao-dev}

if ! command -v "$KIND_BIN" >/dev/null 2>&1; then
	echo "kind not found: $KIND_BIN" >&2
	exit 1
fi

if ! "$KIND_BIN" get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
	echo "kind cluster does not exist: ${KIND_CLUSTER_NAME}"
	exit 0
fi

echo "deleting kind cluster: ${KIND_CLUSTER_NAME}"
"$KIND_BIN" delete cluster --name "$KIND_CLUSTER_NAME"
