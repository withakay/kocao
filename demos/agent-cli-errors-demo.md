# Kocao Agent CLI — Resolved Gaps

*2026-04-13T20:55:00Z — all four transport gaps resolved*

This document records the four live gaps that were identified and fixed in the agent CLI workflow. All gaps are now resolved; see `agent-cli-live-demo.md` for the full working happy path.

## Gap 1: `agent list` returned empty `[]`

**Root cause:** The API server had no `GET /workspace-sessions/{id}/agent-sessions` endpoint. The CLI client called it for each workspace, got 404 (silently tolerated), and returned an empty list.

**Fix:** Added `handleWorkspaceAgentSessionsList` handler that aggregates agent sessions across harness runs for a workspace. Also set the workspace-session label on harness runs at creation time.

**Commit:** `9a4df71` — `fix(api): add GET /workspace-sessions/{id}/agent-sessions endpoint`

## Gap 2: `agent exec` returned EOF

**Root cause:** The API prompt handler returned `{session, result}` but the CLI expected `{events: [...]}`. The response shape mismatch caused the CLI to fail parsing.

**Fix:** The prompt response now includes an `events` array wrapping the JSON-RPC result so the CLI can display it uniformly.

**Commit:** `f62727f` — `fix: align agent session API wire format with CLI and prevent stop timeout`

## Gap 3: `agent logs` returned zero-value events

**Root cause:** The API serialized `agentSessionEvent` with JSON tags `sequence/at/envelope` but the CLI deserialized with `seq/timestamp/data`. The field name mismatch caused all values to deserialize as zero.

**Fix:** Aligned the API struct tags to `seq/timestamp/data` to match the CLI's expectations.

**Commit:** `f62727f` — `fix: align agent session API wire format with CLI and prevent stop timeout`

## Gap 4: `agent stop` timed out

**Root cause:** The sandbox-agent serializes requests per server ID, so the DELETE blocked while the SSE GET stream was still open. The K8s REST client's HTTP/2 connection pool also held stale connections after stream closure.

**Fix:** Stop now: (1) explicitly closes the SSE response body, (2) cancels the stream context, (3) waits for the stream goroutine to exit, (4) sends DELETE, and (5) marks the session as Completed even if the DELETE times out through the K8s pod proxy (the operator handles pod cleanup).

**Commits:** `f62727f` (initial ordering fix) + reconciliation commit (resilient stop with streamDone/streamBody)

## Remaining notes

- The K8s API server pod proxy DELETE can still time out due to HTTP/2 connection pool staleness after closing a long-lived SSE stream. The stop handler now tolerates this gracefully — the session is marked Completed regardless, and the operator cleans up the pod. A follow-on improvement would be to use a separate HTTP client (or disable HTTP/2) for the pod proxy transport.
