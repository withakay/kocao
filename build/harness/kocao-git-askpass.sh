#!/usr/bin/env bash
set -euo pipefail

prompt=${1:-}

username="${KOCAO_GIT_USERNAME:-x-access-token}"
if [[ -n "${KOCAO_GIT_USERNAME_FILE:-}" && -f "${KOCAO_GIT_USERNAME_FILE}" ]]; then
  username=$(cat "${KOCAO_GIT_USERNAME_FILE}" | tr -d '\n\r')
fi

token=""
if [[ -n "${KOCAO_GIT_TOKEN_FILE:-}" && -f "${KOCAO_GIT_TOKEN_FILE}" ]]; then
  token=$(cat "${KOCAO_GIT_TOKEN_FILE}" | tr -d '\n\r')
fi

shopt -s nocasematch
if [[ "${prompt}" == *"username"* ]]; then
  printf "%s" "${username}"
  exit 0
fi
if [[ "${prompt}" == *"password"* ]]; then
  printf "%s" "${token}"
  exit 0
fi

# Fallback: return token for unknown prompts.
printf "%s" "${token}"
