#!/usr/bin/env bash
set -euo pipefail

note "Representative terminal walkthrough for the Zoekt skill demo."
pause 0.2
run "bash .agents/skills/zoekt-search/scripts/install-zoekt.sh >/dev/null && bash .agents/skills/zoekt-search/scripts/install-zoekt.sh && ls -1 .agents/skills/zoekt-search/bin | sort"
pause 0.2
run "bash .agents/skills/zoekt-search/scripts/zoekt-index.sh 2>&1 | grep -E '^\\[zoekt-index\\] (Using binary|Indexing:|Index dir:|Indexed )'"
pause 0.2
run "bash .agents/skills/zoekt-search/scripts/zoekt-search.sh --no-json 'zoekt-reindex' | grep '^\\.opencode/plugins/zoekt-reindex.js:' | sed -E 's#^([^:]+):[0-9]+:.*#\\1: match#' | head -1"
