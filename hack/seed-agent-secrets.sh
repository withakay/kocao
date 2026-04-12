#!/usr/bin/env bash
# seed-agent-secrets.sh — Copy local OAuth auth files into the
# kocao-agent-oauth Kubernetes Secret so harness pods get real credentials.
#
# Usage:  bash hack/seed-agent-secrets.sh
#         make seed-agent-secrets
#
# Safe to run repeatedly; only patches keys for files that exist locally.

set -euo pipefail

NAMESPACE="${K8S_NAMESPACE:-kocao-system}"
SECRET_NAME="kocao-agent-oauth"

# ---------- safety: refuse production cluster ----------
ctx=$(kubectl config current-context 2>/dev/null || true)
if [[ "$ctx" == "hz-arm" ]]; then
  echo "ERROR: kubectl context is 'hz-arm' (PRODUCTION). Refusing to proceed." >&2
  exit 1
fi
echo "kubectl context: ${ctx:-<unknown>}"

# ---------- auth file map: secret-key → local-path ----------
declare -A AUTH_FILES=(
  ["opencode-auth.json"]="${HOME}/.local/share/opencode/auth.json"
  ["codex-auth.json"]="${HOME}/.codex/auth.json"
)

seeded=0

for key in "${!AUTH_FILES[@]}"; do
  src="${AUTH_FILES[$key]}"
  if [[ -f "$src" ]]; then
    echo "seeding ${key} from ${src}"
    contents=$(cat "$src")
    kubectl -n "$NAMESPACE" patch secret "$SECRET_NAME" --type=merge \
      -p "{\"stringData\":{\"${key}\":$(echo "$contents" | jq -Rs .)}}"
    seeded=$((seeded + 1))
  else
    echo "skipping ${key} — ${src} not found"
  fi
done

if [[ $seeded -eq 0 ]]; then
  echo "warning: no auth files found; secret unchanged"
else
  echo "done: seeded ${seeded} key(s) into ${NAMESPACE}/${SECRET_NAME}"
fi
