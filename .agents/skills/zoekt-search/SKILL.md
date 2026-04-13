---
name: zoekt-search
description: "Fast trigram-indexed code search across the entire repository using Zoekt (the engine behind GitHub and Sourcegraph code search). Use INSTEAD OF grep/ripgrep for broad codebase searches. Supports regex, literal strings, file filters, and symbol queries. Auto-installs zoekt binaries and auto-indexes the repo on first use."
---

# Zoekt Code Search

Fast, trigram-indexed code search powered by [Zoekt](https://github.com/sourcegraph/zoekt) — the same engine behind GitHub and Sourcegraph code search. Use this instead of grep or ripgrep for searching across a codebase.

## When to Use This Skill

- Search the codebase for a string, pattern, or symbol
- Find all references to a function, type, or variable
- Find where something is defined
- Find implementations of an interface or method
- Broad code search across the project (faster than grep for indexed repos)

## When NOT to Use This Skill

- Searching within a single known file (use Read or Grep instead)
- Searching non-code content (logs, binary files)
- The query is already answered by your current context

## Quick Start

```bash
# Search for a string
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "AgentSession"

# First run auto-installs zoekt and auto-indexes the repo
```

## Search

```bash
# Literal string search
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "AgentSession"

# Regex search
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "func.*Start"

# Symbol / type definitions
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "type AgentSession struct"

# Filter by filename pattern
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "handleRequest file:\.go$"

# Filter by file path
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "TODO file:internal/"

# Exclude files
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh "Session -file:test"

# Plain text output (default is JSONL)
bash .agents/skills/zoekt-search/scripts/zoekt-search.sh --no-json "AgentSession"
```

### Zoekt Query Syntax

| Syntax | Meaning |
|--------|---------|
| `foo` | Literal search (case-insensitive by default) |
| `foo bar` | Both terms must appear in the same file |
| `"foo bar"` | Exact phrase |
| `foo.*bar` | Regex pattern |
| `file:\.go$` | Restrict to files matching pattern |
| `file:internal/` | Restrict to files under a path |
| `-file:test` | Exclude files matching pattern |
| `case:yes foo` | Case-sensitive search |
| `sym:Start` | Symbol/definition search |
| `lang:go` | Restrict to a language |

## Reindex

The index is built automatically on first search. Reindex manually after significant code changes:

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-index.sh
```

Index a specific directory:

```bash
bash .agents/skills/zoekt-search/scripts/zoekt-index.sh /path/to/source
```

## Reading Results

### JSONL Output (default)

Each line is a JSON object. Key fields:

- `FileName` — file path relative to repo root
- `LineNum` — 1-based line number of the match
- `Line` — matched line content

Multiple matches in the same file are returned as separate lines. Results are ranked by relevance.

### Plain Text Output (`--no-json`)

Human-readable format with file paths, line numbers, and matched content. Useful for quick visual inspection.

## Configuration

| Setting | Default | Override |
|---------|---------|----------|
| Index directory | `<git-dir>/zoekt/` | `--index-dir <path>` or `ZOEKT_INDEX_DIR` env var |
| Binary location | `.agents/skills/zoekt-search/bin/` | Binaries on `PATH` are also detected |

## Prerequisites

- **Go toolchain** — required for building zoekt from source (no pre-built binaries available from sourcegraph/zoekt)
- Binaries are auto-installed to `.agents/skills/zoekt-search/bin/` on first use

## Index Location

The index lives at `<git-dir>/zoekt/` and is automatically gitignored. In worktrees, the index is stored in the worktree's git directory, not the shared bare repo.

## Tips

- Prefer zoekt over grep/ripgrep for broad searches — the trigram index makes it significantly faster on large codebases
- If results seem stale after major changes, reindex with `zoekt-index.sh`
- Combine `file:` filters with search terms to narrow results efficiently
- Use `sym:` prefix to find symbol definitions rather than all occurrences
- The `lang:` filter uses zoekt's built-in language detection, not file extensions

## Error Handling

| Error | Resolution |
|-------|------------|
| "No index found" | Run `bash .agents/skills/zoekt-search/scripts/zoekt-index.sh` |
| "Go is not installed" | Install Go from https://go.dev/dl/ |
| "zoekt binary not found after auto-install" | Check Go installation, ensure `go install` works |
| "Not inside a git repository" | Run from within a git repo or pass `--index-dir` |
