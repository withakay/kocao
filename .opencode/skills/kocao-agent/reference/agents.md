# Kocao Agent Reference

## What This Skill Manages

The `kocao-agent` skill manages **agent sessions** in the Kocao control plane.
Each agent session is attached to a harness run and is addressed by the harness
run ID in the CLI.

`runId` is the canonical public identifier. `sessionId` is stable session
metadata returned once the sandbox-agent session exists.

## Supported Agents

| Agent | Typical workflow |
|---|---|
| `opencode` | OpenCode-led coding or review work |
| `codex` | Codex CLI work inside the session |
| `claude` | Claude Code work inside the session |
| `pi` | Pi-Agent orchestration work inside the session |

Whether a given CLI is actually usable inside the pod still depends on the
deployed harness image and injected credentials.

## Architecture

```text
User / AI assistant
    |
    v
Skill scripts / kocao agent CLI
    |
    v
Control-plane API
    |
    v
Harness run + agent session
    |
    v
Harness pod
  - Git clone / working copy
  - Agent CLI processes
  - Session artifacts and logs
```

## Environment Variables

| Variable | Description | Example |
|---|---|---|
| `KOCAO_API_URL` | Control-plane API base URL | `https://kocao.example.com` |
| `KOCAO_TOKEN` | Bearer token for API authentication | `kocao_tok_...` |

### Optional configuration file

Instead of environment variables, the CLI can read `~/.config/kocao/settings.json`:

```json
{
  "api_url": "https://kocao.example.com",
  "token": "kocao_tok_...",
  "timeout": "30s",
  "verbose": false
}
```

## Prerequisites

1. **kocao CLI installed**
   ```bash
   go install github.com/withakay/kocao/cmd/kocao@latest
   ```

2. **Cluster deployed**
   The Kocao control-plane and operator must be reachable.

3. **Agent secrets seeded**
   Run `seed-agent-secrets` so harness pods get the credentials they need.

4. **Network access**
   If running locally, you may need port-forwarding:
   ```bash
   kubectl port-forward -n kocao-system pod/<control-plane-api-pod> 18080:8080
   export KOCAO_API_URL=http://127.0.0.1:18080
   ```

## Wrapped CLI Commands

```bash
kocao agent list [--workspace ID] [--output table|json|yaml]
kocao agent start --repo URL --agent NAME [--workspace ID] [--revision REF] [--image IMAGE] [--image-pull-secret NAME] [--egress-mode MODE] [--timeout DURATION] [--output table|json]
kocao agent status <run-id> [--output table|json]
kocao agent logs <run-id> [--tail N] [--follow] [--output table|json]
kocao agent exec <run-id> --prompt TEXT [--output json]
kocao agent stop <run-id> [--json]
```

## Lifecycle contract

The current lifecycle phases are:

- `Provisioning`
- `Ready`
- `Running`
- `Stopping`
- `Completed`
- `Failed`

The same public fields are used across the main session endpoints and CLI views:

- `runId`
- `sessionId`
- `phase`
- `agent`
- `runtime`
- `workspaceSessionId`
- `displayName`
- `createdAt`
- `diagnostic` when a blocker is present

`diagnostic.class` can currently be one of:

- `provisioning`
- `image-pull`
- `sandbox-agent-readiness`
- `auth`
- `repo-access`
- `network`

## Behavior Notes

### `agent-start.sh`

- requires `--repo` and `--agent`
- waits for the agent session to become ready unless the command times out
- `--quiet` prints only the run ID
- supports remote-cluster flags like `--image-pull-secret` and `--egress-mode`
- repeated underlying create/status polling is intentional and safe because session creation is idempotent

### `agent-exec.sh`

- wraps `kocao agent exec`
- defaults to JSON output for machine consumption
- accepts `--no-json` to preserve the CLI's formatted event output

### `agent-logs.sh`

- defaults to JSON for one-shot fetches
- accepts `--follow` and `--tail`
- `--no-json` switches back to the CLI's human-readable event table

### `agent-stop.sh`

- returns the terminal lifecycle state when the CLI can fetch it after stop
- repeated stop calls are safe and should keep returning a terminal view

## Troubleshooting

### `required command not found: kocao`

Install the CLI:
```bash
go install github.com/withakay/kocao/cmd/kocao@latest
```

### `KOCAO_TOKEN` or `KOCAO_API_URL` is missing

Export the variables or create `~/.config/kocao/settings.json`.

### `harness run not found`

The run ID is wrong, expired, or belongs to another cluster/environment.

### Session stuck in `Provisioning`

First inspect the CLI blocker output:

```bash
kocao agent status <run-id>
kocao agent list
```

If the blocker points to Kubernetes provisioning, check whether the cluster can schedule and start the harness pod:

```bash
kubectl -n kocao-system get pods
kubectl -n kocao-system describe pod <pod-name>
kubectl -n kocao-system logs <pod-name> -c harness
```

Map common blockers like this:

- `provisioning`: pod not scheduled or not visible yet
- `image-pull`: image or registry access failure
- `sandbox-agent-readiness`: pod is running but sandbox-agent is not healthy yet
- `auth`: missing or invalid credentials/secret wiring
- `repo-access`: checkout or remote repository access failure
- `network`: DNS, egress, or connection failure
