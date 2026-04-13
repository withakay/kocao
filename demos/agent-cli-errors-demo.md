# Kocao Agent CLI Current Gaps Demo

*2026-04-13T20:05:09Z by Showboat 0.6.1*
<!-- showboat-id: 1e96f0a4-8a8a-4abb-8f82-b4ad4ed1f53b -->

This document captures the current known gaps in the live MicroK8s agent workflow so they are documented with real output instead of implied happy paths.

The run ID comes from the successful live demo and the control-plane API is the refreshed build on localhost:18081.

```bash
export KOCAO_API_URL=http://127.0.0.1:18081 KOCAO_TOKEN=dev-bootstrap; go run ./cmd/kocao agent list --output json || true
```

```output
[]
```

Even with a ready agent session, list currently comes back empty. That is an API/aggregation gap, not a demo fabrication.

```bash
export KOCAO_API_URL=http://127.0.0.1:18081 KOCAO_TOKEN=dev-bootstrap KOCAO_TIMEOUT=2m; RUN_ID=$(cat /tmp/kocao-agent-runid); go run ./cmd/kocao agent exec "$RUN_ID" --prompt "List the top-level files in the repo" --output json || true
```

```output
error: send prompt: execute request: Post "http://127.0.0.1:18081/api/v1/harness-runs/df3efa59bdef4294721868af310b9794/agent-session/prompt": EOF
exit status 1
```

Prompt submission still fails against the live control-plane even though the pod and sandbox-agent are healthy.

```bash
export KOCAO_API_URL=http://127.0.0.1:18081 KOCAO_TOKEN=dev-bootstrap; RUN_ID=$(cat /tmp/kocao-agent-runid); go run ./cmd/kocao agent logs "$RUN_ID" --tail 3 --output json || true
```

```output
{"seq":0,"timestamp":"0001-01-01T00:00:00Z","data":null}
{"seq":0,"timestamp":"0001-01-01T00:00:00Z","data":null}
{"seq":0,"timestamp":"0001-01-01T00:00:00Z","data":null}
```

The logs endpoint currently returns zero-value events instead of meaningful payloads.

```bash
export KOCAO_API_URL=http://127.0.0.1:18081 KOCAO_TOKEN=dev-bootstrap; RUN_ID=$(cat /tmp/kocao-agent-runid); go run ./cmd/kocao agent stop "$RUN_ID" --json || true
```

```output
error: execute request: Post "http://127.0.0.1:18081/api/v1/harness-runs/df3efa59bdef4294721868af310b9794/agent-session/stop": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
exit status 1
```
