#!/usr/bin/env bash
set -euo pipefail

KIND_BIN=${KIND_BIN:-kind}
KIND_CLUSTER_NAME=${KIND_CLUSTER_NAME:-kocao-dev}

IMAGE=${1:-}
if [ -z "$IMAGE" ]; then
	echo "usage: $0 <image:tag>" >&2
	exit 2
fi

if ! command -v "$KIND_BIN" >/dev/null 2>&1; then
	echo "kind not found: $KIND_BIN" >&2
	exit 1
fi

echo "loading image into kind (${KIND_CLUSTER_NAME}): ${IMAGE}"
"$KIND_BIN" load docker-image --name "$KIND_CLUSTER_NAME" "$IMAGE"
