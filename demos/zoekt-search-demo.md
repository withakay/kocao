# Zoekt Code Search Skill Demo

*2026-04-13T20:06:35Z by Showboat 0.6.1*
<!-- showboat-id: 59149b48-d8e2-4481-95c8-4be5f3ee3235 -->

This demo proves the current zoekt-search skill with concise, deterministic output: install verification, indexing, narrow JSONL extraction, plain-text search, plugin presence, and the integration test.

```bash
ls -1 .agents/skills/zoekt-search/bin && echo === && bash .agents/skills/zoekt-search/scripts/install-zoekt.sh
```

```output
===
[install-zoekt] Starting zoekt installation (version=latest)
[install-zoekt] Detected platform: darwin-arm64
[install-zoekt] Building from source (sourcegraph/zoekt publishes no pre-built binaries)...
[install-zoekt] Found Go: go version go1.25.1 darwin/arm64
[install-zoekt] Installing zoekt-index@latest ...
[install-zoekt] Installing zoekt@latest ...
[install-zoekt] Built zoekt binaries from source.
[install-zoekt] Verification passed: zoekt-index --help succeeded.
[install-zoekt] Installed zoekt to /Users/jack/Code/withakay/kocao/main/.agents/skills/zoekt-search/scripts/../bin/
[install-zoekt] Done.
```

Index the current repository into the worktree-safe git index directory.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-index.sh
```

```output
[zoekt-index] Using binary: /Users/jack/Code/withakay/kocao/main/.agents/skills/zoekt-search/scripts/../bin/zoekt-index
[zoekt-index] Indexing: /Users/jack/Code/withakay/kocao/main
[zoekt-index] Index dir: /Users/jack/Code/withakay/kocao/.bare/worktrees/main/zoekt
2026/04/13 21:06:45 finished shard /Users/jack/Code/withakay/kocao/.bare/worktrees/main/zoekt/main_v16.00000.zoekt: 283034230 index bytes (overhead 2.7), 18124 files processed 
2026/04/13 21:06:47 finished shard /Users/jack/Code/withakay/kocao/.bare/worktrees/main/zoekt/main_v16.00001.zoekt: 236066974 index bytes (overhead 2.7), 12330 files processed 
[zoekt-index] Indexed /Users/jack/Code/withakay/kocao/main → /Users/jack/Code/withakay/kocao/.bare/worktrees/main/zoekt
[zoekt-index]   Shards: 2
[zoekt-index]   Index size: 495M
```

Use JSONL mode, but extract only file and line data so the output stays readable.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "ZOEKT_REINDEX_DEBOUNCE_MS" | jq -r .' .FileName as $f | .LineMatches[] | "\($f):\(.LineNumber)" ' | head -10
```

```output
.opencode/plugins/zoekt-reindex.js:14
.opencode/plugins/zoekt-reindex.js:31
```

Plain-text mode is useful when you want grep-like output without decoding JSONL fields.

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh --no-json "zoekt-reindex" | head -12
```

```output
.opencode/plugins/zoekt-reindex.js:65:        body: { service: 'zoekt-reindex', level, message },
.ito/changes/008-01_zoekt-search-skill/tasks.md:91:- **Files**: `.opencode/plugins/zoekt-reindex/index.js`
.ito/changes/008-01_zoekt-search-skill/proposal.md:11:- New OpenCode plugin at `.opencode/plugins/zoekt-reindex/` that auto-reindexes on file changes or session idle via debounced hooks.
.ito/changes/008-01_zoekt-search-skill/proposal.md:20:- `zoekt-opencode-plugin`: OpenCode plugin (`.opencode/plugins/zoekt-reindex/`) that auto-reindexes on file changes via debounced hooks, keeping the zoekt index fresh without manual intervention.
.ito/changes/008-01_zoekt-search-skill/proposal.md:30:- New plugin: `.opencode/plugins/zoekt-reindex/`
.ito/changes/008-01_zoekt-search-skill/specs/zoekt-opencode-plugin/spec.md:54:The plugin SHALL be located at `.opencode/plugins/zoekt-reindex/` and SHALL follow the OpenCode plugin contract (ESM module with hook exports).
.ito/changes/008-01_zoekt-search-skill/specs/zoekt-opencode-plugin/spec.md:60:- **WHEN** OpenCode starts in a repository containing `.opencode/plugins/zoekt-reindex/`
```

The OpenCode plugin is part of the feature surface and points back at the zoekt index script.

```bash
grep -n "zoekt-index.sh\|ZOEKT_REINDEX_DEBOUNCE_MS\|session.idle\|file.watcher.updated" .opencode/plugins/zoekt-reindex.js
```

```output
9: *   - session.idle — agent finished processing, good time to refresh the index
10: *   - file.watcher.updated — files changed on disk (git checkout, editor saves)
14: *   ZOEKT_REINDEX_DEBOUNCE_MS      — override debounce window (default 30000)
15: *   ZOEKT_INDEX_DIR                — override index directory (forwarded to zoekt-index.sh)
31:    const raw = Number.parseInt(process.env.ZOEKT_REINDEX_DEBOUNCE_MS || '', 10);
40:      worktree && path.join(worktree, '.agents/skills/zoekt-search/scripts/zoekt-index.sh'),
41:      path.join(directory, '.agents/skills/zoekt-search/scripts/zoekt-index.sh'),
142:      if (event.type === 'session.idle' || event.type === 'file.watcher.updated') {
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
