#!/usr/bin/env bash

API_RESPONSE_FILE=""
API_RESPONSE_CODE=""

script_name() {
  basename -- "$0"
}

print_kocao_install_hint() {
  echo "Install: go install github.com/withakay/kocao/cmd/kocao@latest" >&2
}

fail() {
  echo "error: $*" >&2
  exit 1
}

usage_error() {
  echo "error: $*" >&2
  echo "Run $(script_name) --help for usage." >&2
  exit 2
}

require_command() {
  local command_name="$1"
  if command -v "$command_name" >/dev/null 2>&1; then
    return
  fi

  echo "error: required command not found: $command_name" >&2
  if [[ "$command_name" == "kocao" ]]; then
    print_kocao_install_hint
  fi
  exit 2
}

require_commands() {
  local command_name
  for command_name in "$@"; do
    require_command "$command_name"
  done
}

require_flag_value() {
  local flag="$1"
  local argc="$2"
  if (( argc < 2 )); then
    usage_error "missing value for ${flag}"
  fi
}

require_nonempty() {
  local value="$1"
  local label="$2"
  if [[ -z "$value" ]]; then
    usage_error "${label} is required"
  fi
}

require_positive_integer() {
  local value="$1"
  local label="$2"
  if ! [[ "$value" =~ ^[1-9][0-9]*$ ]]; then
    usage_error "${label} must be a positive integer"
  fi
}

require_token() {
  if [[ -z "${KOCAO_TOKEN:-}" ]]; then
    usage_error "KOCAO_TOKEN is not set. Export KOCAO_TOKEN or configure ~/.config/kocao/settings.json."
  fi
}

api_base_url() {
  local url="${KOCAO_API_URL:-http://127.0.0.1:8080}"
  printf '%s\n' "${url%/}"
}

urlencode() {
  jq -nr --arg value "$1" '$value|@uri'
}

json_error_message() {
  local file="$1"
  jq -r '.error // empty' "$file" 2>/dev/null || true
}

print_json_or_raw() {
  local file="$1"
  if [[ ! -s "$file" ]]; then
    return
  fi
  jq . "$file" 2>/dev/null || cat "$file"
}

print_api_error() {
  local status_code="$1"
  local body_file="$2"
  local api_message

  api_message="$(json_error_message "$body_file")"
  echo "error: API returned HTTP ${status_code}" >&2
  if [[ -n "$api_message" ]]; then
    echo "$api_message" >&2
  elif [[ -s "$body_file" ]]; then
    cat "$body_file" >&2
  fi
}

api_request() {
  local method="$1"
  local route="$2"
  local payload="${3-}"
  local body_file url status_code config_file payload_file

  require_token
  url="$(api_base_url)${route}"
  body_file="$(mktemp)"
  config_file="$(mktemp)"
  payload_file=""
  trap 'rm -f "$config_file" "$payload_file"' RETURN

  {
    printf 'url = "%s"\n' "$url"
    printf 'request = "%s"\n' "$method"
    printf 'header = "Authorization: Bearer %s"\n' "$KOCAO_TOKEN"
    printf 'header = "Accept: application/json"\n'
  } >"$config_file"

  if [[ -n "$payload" ]]; then
    payload_file="$(mktemp)"
    printf '%s' "$payload" >"$payload_file"
    {
      printf 'header = "Content-Type: application/json"\n'
      printf 'data = @"%s"\n' "$payload_file"
    } >>"$config_file"
  fi

  status_code="$(curl -sS -o "$body_file" -w '%{http_code}' --config "$config_file")" || {
    rm -f "$body_file"
    fail "failed to reach $(api_base_url). Check KOCAO_API_URL and your network connection."
  }

  # shellcheck disable=SC2034
  API_RESPONSE_FILE="$body_file"
  API_RESPONSE_CODE="$status_code"
}

api_request_ok() {
  [[ "${API_RESPONSE_CODE:-}" =~ ^[0-9]{3}$ ]] || fail "received an invalid HTTP status from the API: ${API_RESPONSE_CODE:-<empty>}"
  (( 10#${API_RESPONSE_CODE} >= 200 && 10#${API_RESPONSE_CODE} < 300 ))
}
