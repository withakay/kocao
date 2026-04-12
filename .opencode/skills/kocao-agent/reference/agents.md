# Kocao Agent Reference

## What This Skill Manages

The `kocao-agent` skill manages **workspace sessions** in the Kocao control
plane. A workspace session is the lifecycle wrapper around a harness pod.

Important limitation: the current session-creation API is generic. It does not
choose between different runtime images or agent implementations. The
`agent-start.sh --agent ...` flag is a **labeling convention** that keeps
session names readable for humans.

## Common Session Labels

| Label | Typical intended workflow |
|---|---|
| `opencode` | OpenCode-led coding or review work |
| `codex` | Codex CLI work inside the session |
| `claude` | Claude Code work inside the session |
| `pi` | Pi-Agent orchestration work inside the session |

Use the label to communicate intent. Whether a given CLI is actually usable in
the pod still depends on the deployed harness image and injected credentials.

## Architecture

```
User / AI assistant
    |
    v
Skill scripts / kocao CLI
    |
    v
Control-plane API
    |
    v
Workspace session
    |
    v
Harness pod
  - Git clone / working copy
  - Agent CLI processes
  - Session artifacts and logs
```

## Environment Variables

### Required for API-backed scripts

| Variable | Description | Example |
|---|---|---|
| `KOCAO_API_URL` | Control-plane API base URL | `https://kocao.example.com` |
| `KOCAO_TOKEN` | Bearer token for API authentication | `kocao_tok_...` |

### Optional

| Variable | Description | Default |
|---|---|---|
| `KOCAO_TIMEOUT` | CLI HTTP timeout | `15s` |
| `KOCAO_VERBOSE` | Enable CLI debug output | `false` |

### Configuration File

Instead of environment variables, the CLI can read a JSON config file from
`~/.config/kocao/settings.json`:

```json
{
  "api_url": "https://kocao.example.com",
  "token": "kocao_tok_...",
  "timeout": "30s",
  "verbose": false
}
```

Priority order: env vars > explicit `--config` flag > default config files.

## Prerequisites

1. **kocao CLI installed**
   ```bash
   go install github.com/withakay/kocao/cmd/kocao@latest
   ```

2. **Cluster deployed**
   The Kocao control-plane and operator must be reachable.

3. **Agent secrets seeded**
   Run `seed-agent-secrets` so the harness pods get the credentials they need.

4. **Network access**
   If you are running locally, you may need port-forwarding:
   ```bash
   kubectl port-forward svc/control-plane-api 8080:8080
   ```

## Commands Used by This Skill

### CLI-backed wrappers

```bash
kocao sessions ls [--json]
kocao sessions status <session-id> [--json]
kocao sessions logs <session-id> [--tail N] [--container NAME] [--follow] [--json]
kocao sessions attach <session-id> [--driver] [--collab]
```

### API-backed wrappers

```text
POST   /api/v1/workspace-sessions
DELETE /api/v1/workspace-sessions/<session-id>
POST   /api/v1/workspace-sessions/<session-id>/exec   # optional / experimental
```

## Behavior Notes

### `agent-start.sh`

- requires `--repo`
- accepts `--agent`, but only to help generate a readable display name
- waits for `Running` by default
- `--quiet` prints only the session ID

### `agent-exec.sh`

- depends on a control-plane that implements `/exec`
- if that endpoint is missing, the script exits with guidance to use
  `kocao sessions attach <id> --driver`

### `agent-logs.sh`

- defaults to JSON for one-shot fetches
- requires `--no-json` when you use `--follow`

## Troubleshooting

### `required command not found: kocao`

Install the CLI:
```bash
go install github.com/withakay/kocao/cmd/kocao@latest
```

### `KOCAO_TOKEN is not set`

Export the token or create the config file:
```bash
export KOCAO_TOKEN="your-token-here"
```

### `API returned HTTP 400`

The most common causes are:
- invalid JSON sent to the API
- `repoURL` missing on session creation
- `repoURL` not using `https://`

### `API returned HTTP 401`

The bearer token is missing, invalid, or expired.

### `API returned HTTP 404`

Usually one of these:
- the session ID is wrong
- the session has already been deleted
- you tried `agent-exec.sh` against a control-plane that does not implement the optional `/exec` endpoint

### `attach requires an interactive terminal (TTY)`

`kocao sessions attach` needs a real terminal. Use it from an interactive shell.

### Session stuck in `Pending`

Check whether Kubernetes can schedule the harness pod:
```bash
kubectl get pods -l workspace-session-id=<session-id>
kubectl describe pod <pod-name>
```

Look for scheduling failures, image pull errors, or missing secrets.
