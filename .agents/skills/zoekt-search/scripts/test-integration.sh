#!/usr/bin/env bash
set -euo pipefail

# test-integration.sh — Integration test for zoekt-search skill.
#
# Exercises the full flow: install → index → search → cleanup.
# Exit 0 if all checks pass, 1 if any fail.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="${SCRIPT_DIR}/.."

PASS_COUNT=0
FAIL_COUNT=0
TMPDIR_ROOT=""

pass() {
    PASS_COUNT=$((PASS_COUNT + 1))
    printf 'PASS: %s\n' "$1"
}

fail() {
    FAIL_COUNT=$((FAIL_COUNT + 1))
    printf 'FAIL: %s\n' "$1" >&2
}

cleanup() {
    if [[ -n "${TMPDIR_ROOT}" && -d "${TMPDIR_ROOT}" ]]; then
        rm -rf "${TMPDIR_ROOT}"
    fi
}
trap cleanup EXIT

# --- Setup: create temp directory with known source files --------------------

TMPDIR_ROOT="$(mktemp -d)"
SRC_DIR="${TMPDIR_ROOT}/src"
INDEX_DIR="${TMPDIR_ROOT}/index"

mkdir -p "${SRC_DIR}" "${INDEX_DIR}"

# Initialize a git repo in src so the scripts don't complain
git init -q "${SRC_DIR}"

# Create known Go files
cat > "${SRC_DIR}/main.go" <<'GOEOF'
package main

import "fmt"

// AgentSession represents a running agent session.
type AgentSession struct {
	ID   string
	Name string
}

func StartAgent(name string) *AgentSession {
	return &AgentSession{ID: "a1", Name: name}
}

func main() {
	s := StartAgent("test")
	fmt.Println(s.Name)
}
GOEOF

cat > "${SRC_DIR}/handler.go" <<'GOEOF'
package main

import "net/http"

func handleRequest(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
GOEOF

# Create known TypeScript files
cat > "${SRC_DIR}/session.ts" <<'TSEOF'
export interface AgentSession {
  id: string;
  name: string;
  status: "running" | "stopped";
}

export function createSession(name: string): AgentSession {
  return { id: crypto.randomUUID(), name, status: "running" };
}

export function stopSession(session: AgentSession): AgentSession {
  return { ...session, status: "stopped" };
}
TSEOF

cat > "${SRC_DIR}/utils.ts" <<'TSEOF'
export function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000);
  const minutes = Math.floor(seconds / 60);
  return `${minutes}m ${seconds % 60}s`;
}

export function uniqueId(): string {
  return Math.random().toString(36).substring(2, 10);
}
TSEOF

# Commit files so zoekt-index sees them properly
git -C "${SRC_DIR}" add -A
git -C "${SRC_DIR}" commit -q -m "test fixtures"

# --- Test 1: zoekt-index indexes the temp directory -------------------------

printf '\n--- Test 1: Index temp directory ---\n'

if bash "${SCRIPT_DIR}/zoekt-index.sh" --index-dir "${INDEX_DIR}" "${SRC_DIR}" >/dev/null 2>&1; then
    pass "zoekt-index.sh completed successfully"
else
    fail "zoekt-index.sh returned non-zero"
fi

# Verify shards were created
shard_count=$(find "${INDEX_DIR}" -maxdepth 1 -name '*.zoekt' -type f 2>/dev/null | wc -l | tr -d ' ')
if [[ "${shard_count}" -gt 0 ]]; then
    pass "Index contains ${shard_count} shard file(s)"
else
    fail "No .zoekt shard files found in ${INDEX_DIR}"
fi

# --- Test 2: Search for known pattern (AgentSession) — JSON output ----------
#
# NOTE: In JSONL mode, zoekt base64-encodes the Line field. We check FileName
# fields (which are plain text) and also decode Line content for verification.

printf '\n--- Test 2: Search for "AgentSession" (JSONL) ---\n'

search_output=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" "AgentSession" 2>/dev/null) || true

if [[ -n "${search_output}" ]]; then
    pass "Search for 'AgentSession' returned non-empty results"
else
    fail "Search for 'AgentSession' returned empty results"
fi

# Should match both .go and .ts files (FileName is plain text in JSONL)
if echo "${search_output}" | grep -q '"FileName":"main.go"'; then
    pass "JSONL results include main.go"
else
    fail "JSONL results missing main.go"
fi

if echo "${search_output}" | grep -q '"FileName":"session.ts"'; then
    pass "JSONL results include session.ts"
else
    fail "JSONL results missing session.ts"
fi

# Verify Line content by base64 decoding
if echo "${search_output}" | python3 -c "
import sys, json, base64
for line in sys.stdin:
    line = line.strip()
    if not line: continue
    obj = json.loads(line)
    for m in obj.get('LineMatches', []):
        decoded = base64.b64decode(m['Line']).decode('utf-8', errors='replace')
        if 'AgentSession' in decoded:
            sys.exit(0)
sys.exit(1)
" 2>/dev/null; then
    pass "Decoded JSONL Line content contains 'AgentSession'"
else
    fail "Decoded JSONL Line content does not contain 'AgentSession'"
fi

# --- Test 3: Search for regex pattern (func.*Start) -------------------------

printf '\n--- Test 3: Search for "func.*Start" (regex) ---\n'

regex_output=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" "func.*Start" 2>/dev/null) || true

# Check using --no-json for content verification (plain text contains the match directly)
regex_plain=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" --no-json "func.*Start" 2>/dev/null) || true

if echo "${regex_plain}" | grep -q "StartAgent"; then
    pass "Regex search 'func.*Start' found StartAgent (plain text)"
else
    fail "Regex search 'func.*Start' did not find StartAgent"
fi

if echo "${regex_output}" | grep -q '"FileName":"main.go"'; then
    pass "Regex JSONL results include main.go"
else
    fail "Regex JSONL results missing main.go"
fi

# --- Test 4: Search for pattern that should NOT match -----------------------

printf '\n--- Test 4: Search for non-existent pattern ---\n'

nomatch_output=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" "xyzzy_does_not_exist_12345" 2>/dev/null) || true

if [[ -z "${nomatch_output}" ]]; then
    pass "Search for non-existent pattern returned empty results"
else
    fail "Search for non-existent pattern returned unexpected output: ${nomatch_output}"
fi

# --- Test 5: Search with --no-json (plain text output) ----------------------

printf '\n--- Test 5: Search with --no-json ---\n'

nojson_output=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" --no-json "handleRequest" 2>/dev/null) || true

if echo "${nojson_output}" | grep -q "handleRequest"; then
    pass "--no-json search returned results containing handleRequest"
else
    fail "--no-json search did not return expected results"
fi

# Verify it is NOT JSON (should not start with '{')
first_char=$(echo "${nojson_output}" | head -c1)
if [[ "${first_char}" != "{" ]]; then
    pass "--no-json output is not JSON formatted"
else
    fail "--no-json output appears to be JSON (starts with '{')"
fi

# --- Test 6: JSON output format validation ----------------------------------

printf '\n--- Test 6: Verify JSONL output format ---\n'

json_output=$(bash "${SCRIPT_DIR}/zoekt-search.sh" --index-dir "${INDEX_DIR}" "formatDuration" 2>/dev/null) || true

if echo "${json_output}" | head -1 | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null; then
    pass "JSONL output is valid JSON"
else
    # zoekt JSONL may have multiple lines; check if any line is valid JSON
    if echo "${json_output}" | while IFS= read -r line; do
        echo "${line}" | python3 -c "import sys,json; json.load(sys.stdin)" 2>/dev/null && exit 0
    done; then
        pass "JSONL output contains valid JSON lines"
    else
        fail "JSONL output is not valid JSON"
    fi
fi

# --- Summary -----------------------------------------------------------------

printf '\n=== Results ===\n'
printf 'Passed: %d\n' "${PASS_COUNT}"
printf 'Failed: %d\n' "${FAIL_COUNT}"

if [[ "${FAIL_COUNT}" -gt 0 ]]; then
    printf '\nFAILED: %d test(s) failed.\n' "${FAIL_COUNT}"
    exit 1
fi

printf '\nAll tests passed.\n'
exit 0
