#!/usr/bin/env bash
set -euo pipefail

matrix=${1:-/etc/kocao/runtime-matrix.json}

if [[ ! -f "${matrix}" ]]; then
  echo "runtime matrix not found: ${matrix}" >&2
  exit 1
fi

want_go=$(jq -r '.go' "${matrix}")
want_node=$(jq -r '.node' "${matrix}")
want_py=$(jq -r '.python' "${matrix}")

got_go=$(go version | awk '{print $3}' | sed 's/^go//')
if [[ "${got_go}" != "${want_go}" ]]; then
  echo "go version mismatch: got=${got_go} want=${want_go}" >&2
  exit 1
fi

got_node=$(node --version | sed 's/^v//')
if [[ "${got_node}" != "${want_node}" ]]; then
  echo "node version mismatch: got=${got_node} want=${want_node}" >&2
  exit 1
fi

got_py=$(python3 --version | awk '{print $2}' | cut -d. -f1-2)
if [[ "${got_py}" != "${want_py}" ]]; then
  echo "python version mismatch: got=${got_py} want=${want_py}" >&2
  exit 1
fi

missing=0
while IFS= read -r bin; do
  if ! command -v "${bin}" >/dev/null 2>&1; then
    echo "missing required bin: ${bin}" >&2
    missing=1
  fi
done < <(jq -r '.required_bins[]' "${matrix}")

if [[ "${missing}" -ne 0 ]]; then
  exit 1
fi

echo "ok"
