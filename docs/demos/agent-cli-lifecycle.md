# Agent CLI Lifecycle Demo

This document walks through the full `kocao agent` CLI lifecycle: starting a
sandbox agent, inspecting its status, sending prompts, streaming logs, and
tearing it down.

## Prerequisites

1. **kocao deployed to Kind** with the control-plane API running:

```bash
make kind-up
make deploy-dev
```

2. **Agent secrets seeded** so the harness can authenticate with upstream providers:

```bash
hack/seed-agent-secrets.sh
```

3. **CLI configured** to talk to the local control-plane:

```bash
export KOCAO_API_URL=http://localhost:8080
export KOCAO_TOKEN=$(kubectl get secret kocao-api-token -o jsonpath='{.data.token}' | base64 -d)
```

---

## Step 1 — Start an agent

Create a new workspace session and launch a Codex agent against a repository.
The CLI creates the workspace, harness run, and agent session in one shot, then
polls until the agent reaches the `Ready` phase.

```bash
$ kocao agent start --repo https://github.com/withakay/kocao --agent codex
```

```
Creating workspace session... done
Starting harness run... done
Waiting for agent session... ready
Session ID:  as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b
Run ID:      hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d
Agent:       codex
Phase:       Ready
```

The `--output json` flag produces machine-readable output:

```bash
$ kocao agent start --repo https://github.com/withakay/kocao --agent codex --output json
```

```json
{
  "sessionId": "as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b",
  "runId": "hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d",
  "agent": "codex",
  "phase": "Ready"
}
```

**What happened:** The CLI called `POST /api/v1/workspace-sessions` to create a
workspace, then `POST .../harness-runs` to schedule the sandbox container, then
`POST .../agent-session` to attach the Codex agent. It polled
`GET .../agent-session` every 2 seconds until the phase flipped to `Ready`.

---

## Step 2 — Check status

Inspect the running agent session by its run ID.

```bash
$ kocao agent status hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d
```

```
Session ID:   as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b
Run ID:       hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d
Agent:        codex
Runtime:      harness-v1
Phase:        Ready
Workspace:    ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890
Created:      2026-04-12T14:23:01Z
```

JSON format:

```bash
$ kocao agent status hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d --output json
```

```json
{
  "sessionId": "as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b",
  "runId": "hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d",
  "displayName": "",
  "runtime": "harness-v1",
  "agent": "codex",
  "phase": "Ready",
  "workspaceSessionId": "ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "createdAt": "2026-04-12T14:23:01Z"
}
```

---

## Step 3 — List running agents

Show all active agent sessions across all workspaces.

```bash
$ kocao agent list
```

```
SESSION ID                                    RUN                                           AGENT   PHASE  WORKSPACE                                     CREATED
as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b      hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d      codex   Ready  ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890      2026-04-12T14:23:01Z
```

Filter by workspace:

```bash
$ kocao agent list --workspace ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890 --output json
```

```json
[
  {
    "sessionId": "as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b",
    "runId": "hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d",
    "displayName": "",
    "runtime": "harness-v1",
    "agent": "codex",
    "phase": "Ready",
    "workspaceSessionId": "ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "createdAt": "2026-04-12T14:23:01Z"
  }
]
```

The `list` command also supports `--output yaml` for Kubernetes-style workflows.

---

## Step 4 — Send a prompt

Execute a task on the running agent. The CLI sends the prompt to the agent
session and streams back the response events.

```bash
$ kocao agent exec hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d --prompt "What files are in the repo?"
```

```
[14:24:12] seq=1  type=message text="I'll look at the repository structure for you."
[14:24:13] seq=2  type=tool_use text="Running: find . -maxdepth 2 -type f | head -30"
[14:24:15] seq=3  type=tool_result text="./go.mod\n./go.sum\n./Makefile\n./README.md\n./cmd/kocao/main.go\n./cmd/..."
[14:24:16] seq=4  type=message text="The repository contains a Go project with the following top-level structure:..."
```

The prompt can also be passed as a positional argument:

```bash
$ kocao agent exec hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d "What files are in the repo?"
```

JSON output includes the full event payloads:

```bash
$ kocao agent exec hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d -p "List Go packages" --output json
```

```json
{
  "events": [
    {
      "seq": 1,
      "timestamp": "2026-04-12T14:24:12Z",
      "data": {"type": "message", "text": "I'll list the Go packages in the repository."}
    },
    {
      "seq": 2,
      "timestamp": "2026-04-12T14:24:14Z",
      "data": {"type": "tool_use", "tool": "bash", "input": "go list ./..."}
    },
    {
      "seq": 3,
      "timestamp": "2026-04-12T14:24:16Z",
      "data": {"type": "tool_result", "output": "github.com/withakay/kocao/cmd/kocao\ngithub.com/withakay/kocao/internal/controlplanecli\n..."}
    }
  ]
}
```

---

## Step 5 — Stream logs

Follow the agent session's event stream in real time via SSE. Press Ctrl+C to
detach without stopping the agent.

```bash
$ kocao agent logs hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d --follow
```

```
2026-04-12T14:23:01Z  0  session.start    {"type":"session.start","agent":"codex","runtime":"harness-v1"}
2026-04-12T14:23:03Z  1  sandbox.ready    {"type":"sandbox.ready","image":"kocao/harness-runtime:dev"}
2026-04-12T14:24:12Z  2  prompt.received  {"type":"prompt.received","prompt":"What files are in the repo?"}
2026-04-12T14:24:12Z  3  message          {"type":"message","text":"I'll look at the repository structure for you."}
2026-04-12T14:24:13Z  4  tool_use         {"type":"tool_use","tool":"bash","input":"find . -maxdepth 2 -type f | he...
2026-04-12T14:24:15Z  5  tool_result      {"type":"tool_result","output":"./go.mod\n./go.sum\n./Makefile\n./README....
2026-04-12T14:24:16Z  6  message          {"type":"message","text":"The repository contains a Go project with the ...
^C
```

Fetch the last N events without streaming:

```bash
$ kocao agent logs hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d --tail 3
```

```
TIMESTAMP             SEQ  TYPE          DATA
2026-04-12T14:24:15Z  5    tool_result   {"type":"tool_result","output":"./go.mod\n./go.sum\n./Makefile\n./README....
2026-04-12T14:24:16Z  6    message       {"type":"message","text":"The repository contains a Go project with the ...
2026-04-12T14:24:16Z  7    idle          {"type":"idle"}
```

---

## Step 6 — Stop the agent

Terminate the agent session and its sandbox container.

```bash
$ kocao agent stop hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d
```

```
Agent session stopped
  Run ID:      hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d
  Session ID:  as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b
  Name:        -
  Runtime:     harness-v1
  Agent:       codex
  Phase:       Stopped
  Workspace:   ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890
```

JSON confirmation:

```bash
$ kocao agent stop hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d --json
```

```json
{
  "status": "stopped",
  "session": {
    "sessionId": "as-7f2a4b1e-9c3d-4e8f-b6a1-2d5e8f0c3a7b",
    "runId": "hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d",
    "displayName": "",
    "runtime": "harness-v1",
    "agent": "codex",
    "phase": "Stopped",
    "workspaceSessionId": "ws-a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "createdAt": "2026-04-12T14:23:01Z"
  }
}
```

---

## Step 7 — Verify stopped

Confirm no active sessions remain.

```bash
$ kocao agent list
```

```
no agent sessions found
```

---

## Error Handling

### Sending a prompt to a stopped session

```bash
$ kocao agent exec hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d --prompt "hello"
```

```
Error: send prompt: API error 409 POST .../agent-session/prompt: agent session is not running
```

### Stopping an already-stopped session

```bash
$ kocao agent stop hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d
```

```
Error: agent session already stopped (run hr-3e8b2c1a-4d5f-6a7b-8c9d-0e1f2a3b4c5d)
```

### Missing required flags

```bash
$ kocao agent start --repo https://github.com/withakay/kocao
```

```
Error: usage: kocao agent start --repo <url> --agent <name>: missing required flag --agent
```

### Invalid run ID

```bash
$ kocao agent status nonexistent-run-id
```

```
Error: API error 404 GET .../harness-runs/nonexistent-run-id/agent-session: not found
```

---

## Command Reference

| Command | Description |
|---------|-------------|
| `kocao agent start --repo <url> --agent <name>` | Start a new agent session |
| `kocao agent status <run-id>` | Show agent session details |
| `kocao agent list` | List all agent sessions |
| `kocao agent exec <run-id> --prompt <text>` | Send a prompt to a running agent |
| `kocao agent logs <run-id> [--follow]` | View or stream agent events |
| `kocao agent stop <run-id>` | Stop an agent session |

### Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | `table` | Output format: `table`, `json`, or `yaml` (list only) |

### `agent start` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | *(required)* | Repository URL |
| `--agent` | *(required)* | Agent name (`codex`, `claude`, `opencode`, `pi`) |
| `--workspace` | *(auto-create)* | Reuse an existing workspace session ID |
| `--revision` | `main` | Repository revision/branch |
| `--image` | `kocao/harness-runtime:dev` | Harness runtime container image |
| `--timeout` | `5m` | Timeout waiting for agent readiness |

### `agent logs` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--follow`, `-f` | `false` | Stream events via SSE |
| `--tail` | `0` (all) | Show last N events |

### `agent stop` Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output JSON instead of table |
