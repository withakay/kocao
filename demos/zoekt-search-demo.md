# Zoekt Code Search Skill Demo

*2026-04-13T20:06:35Z by Showboat 0.6.1*
<!-- showboat-id: 59149b48-d8e2-4481-95c8-4be5f3ee3235 -->

This demo proves the current zoekt-search skill with concise, deterministic output: install verification, indexing, narrow JSONL extraction, plain-text search, plugin presence, and the integration test.

Companion terminal recording: [`zoekt-search-demo.cast`](zoekt-search-demo.cast)

The cast is a shorter terminal walkthrough of the same workflow. It shows the install, index, and search flow as live terminal playback, while the markdown below remains the deterministic artifact for `showboat verify`.

```bash
bash .agents/skills/zoekt-search/scripts/install-zoekt.sh >/dev/null && bash .agents/skills/zoekt-search/scripts/install-zoekt.sh && ls -1 .agents/skills/zoekt-search/bin | sort
```

```output
[install-zoekt] Starting zoekt installation (version=latest)
[install-zoekt] Detected platform: darwin-arm64
[install-zoekt] zoekt binaries already installed and working in /Users/jack/Code/withakay/kocao/main/.agents/skills/zoekt-search/scripts/../bin
[install-zoekt] Version pinning not requested; skipping re-install.
zoekt
zoekt-index
```

Index the current repository into the worktree-safe git index directory.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-index.sh 2>&1 | grep -E '^\[zoekt-index\] (Using binary|Indexing:|Index dir:|Indexed )'
```

```output
[zoekt-index] Using binary: /Users/jack/Code/withakay/kocao/main/.agents/skills/zoekt-search/scripts/../bin/zoekt-index
[zoekt-index] Indexing: /Users/jack/Code/withakay/kocao/main
[zoekt-index] Index dir: /Users/jack/Code/withakay/kocao/.bare/worktrees/main/zoekt
[zoekt-index] Indexed /Users/jack/Code/withakay/kocao/main → /Users/jack/Code/withakay/kocao/.bare/worktrees/main/zoekt
```

Use JSONL mode, but extract only the plugin filename so the output stays readable and stable as the repository evolves.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "ZOEKT_REINDEX_DEBOUNCE_MS" | jq -r 'select(.FileName == ".opencode/plugins/zoekt-reindex.js") | .FileName' | sort -u
```

```output
.opencode/plugins/zoekt-reindex.js
```

Plain-text mode is useful when you want grep-like output without decoding JSONL fields. Normalize the hit so line movement does not churn the demo.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh --no-json "zoekt-reindex" | grep '^\.opencode/plugins/zoekt-reindex.js:' | sed -E 's#^([^:]+):[0-9]+:.*#\1: match#' | head -1
```

```output
.opencode/plugins/zoekt-reindex.js: match
```

The OpenCode plugin is part of the feature surface and points back at the zoekt index script. Normalize the grep output to the matched tokens so verification is resilient to line movement.

```bash
grep -o "zoekt-index.sh\|ZOEKT_REINDEX_DEBOUNCE_MS\|session.idle\|file.watcher.updated" .opencode/plugins/zoekt-reindex.js | awk '!seen[$0]++'
```

```output
session.idle
file.watcher.updated
ZOEKT_REINDEX_DEBOUNCE_MS
zoekt-index.sh
```

Finally, run the full integration test script to verify indexing and search behavior end-to-end.

```bash
bash .agents/skills/zoekt-search/scripts/test-integration.sh
```

```output

--- Test 1: Index temp directory ---
PASS: zoekt-index.sh completed successfully
PASS: Index contains 1 shard file(s)

--- Test 2: Search for "AgentSession" (JSONL) ---
PASS: Search for 'AgentSession' returned non-empty results
PASS: JSONL results include main.go
PASS: JSONL results include session.ts
PASS: Decoded JSONL Line content contains 'AgentSession'

--- Test 3: Search for "func.*Start" (regex) ---
PASS: Regex search 'func.*Start' found StartAgent (plain text)
PASS: Regex JSONL results include main.go

--- Test 4: Search for non-existent pattern ---
PASS: Search for non-existent pattern returned empty results

--- Test 5: Search with --no-json ---
PASS: --no-json search returned results containing handleRequest
PASS: --no-json output is not JSON formatted

--- Test 6: Verify JSONL output format ---
PASS: JSONL output is valid JSON

=== Results ===
Passed: 12
Failed: 0

All tests passed.
```
