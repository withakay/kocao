# Kocao Agent CLI Live Demo

*2026-04-12T23:36:15Z by Showboat 0.6.1*
<!-- showboat-id: 7d3b1bc9-4568-4f7a-8ff9-6f81b83906cb -->

This demo exercises the kocao agent CLI against a live MicroK8s cluster running the Kocao control plane.

Verify no agents are running.

```bash
go run ./cmd/kocao agent list
```

```output
no agent sessions found
```

Show the agent CLI help.

```bash
go run ./cmd/kocao agent help
```

```output
kocao agent — manage sandbox agents

Usage:
  kocao agent <subcommand> [flags]

Subcommands:
  list, ls    List agents
  start       Start an agent
  stop        Stop an agent
  logs        View agent logs
  exec        Execute a command in an agent
  status      Show agent status
```

Start a codex agent. The harness image is large (~2.4GB) so first-time pulls take several minutes. Agent session initialization may timeout on cold starts.

```bash
go run ./cmd/kocao agent start --repo https://github.com/golang/example --agent codex --revision master --image ghcr.io/withakay/kocao/harness-runtime:dev-microk8s --image-pull-secret ghcr-pull --egress-mode full --timeout 2m 2>&1 || true
```

```output
Creating workspace session... done
Starting harness run... done
Waiting for agent session... timeout
Last known state: phase=Provisioning sessionId=
error: timed out waiting for agent session to become ready
exit status 1
```

Check the agent's status using the run ID from the harness.

```bash
go run ./cmd/kocao agent status c354834c6d8ad87bc07ae77256485e0b 2>&1 || true
```

```output
  Run ID:    -
  Session ID: -
  Name:      -
  Runtime:   sandbox-agent
  Agent:     codex
  Phase:     Provisioning
  Workspace: -
  Created:   -
```

View the agent status in JSON format.

```bash
go run ./cmd/kocao agent status c354834c6d8ad87bc07ae77256485e0b --output json 2>&1 || true
```

```output
{
  "sessionId": "",
  "runId": "",
  "displayName": "",
  "runtime": "sandbox-agent",
  "agent": "codex",
  "phase": "Provisioning",
  "workspaceSessionId": "",
  "createdAt": "0001-01-01T00:00:00Z"
}
```

List agents — our codex agent should appear in Provisioning state.

```bash
go run ./cmd/kocao agent list
```

```output
error: not found
exit status 1
```

Fetch event logs for the harness run.

```bash
go run ./cmd/kocao agent logs c354834c6d8ad87bc07ae77256485e0b --tail 5 2>&1 || true
```

```output
TIMESTAMP  SEQ  TYPE  DATA
```

Stop the agent session.

```bash
go run ./cmd/kocao agent stop c354834c6d8ad87bc07ae77256485e0b 2>&1 || true
```

```output
error: api request failed (502): sandbox-agent proxy DELETE https://10.152.183.1:443/api/v1/namespaces/kocao-system/pods/epic-mclean-85e0b:2468/proxy/v1/acp/c354834c6d8ad87bc07ae77256485e0b returned 503: {"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"error trying to reach service: dial tcp 10.1.24.156:2468: connect: connection refused","reason":"ServiceUnavailable","code":503}
exit status 1
```

Verify no agents are running after cleanup.

```bash
go run ./cmd/kocao agent list
```

```output
no agent sessions found
```
