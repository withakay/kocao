# Kocao Agent Reference

## Supported Agents

| Agent | Description | Image |
|---|---|---|
| `opencode` | OpenCode AI coding agent (default) | Built from project Dockerfile |
| `codex` | OpenAI Codex CLI agent | Codex harness image |
| `claude` | Anthropic Claude Code agent | Claude Code harness image |
| `pi` | Pi-Agent orchestrator | Pi-Agent harness image |

Each agent type runs in an ephemeral Kubernetes pod ("harness pod") provisioned
by the Kocao control-plane operator. The pod includes a toolchain-heavy base
image with common development tools, language runtimes, and SCM tooling.

## Architecture

```
User/AI Assistant
    |
    v
kocao CLI  --->  Control-Plane API  --->  Kubernetes Operator
                                              |
                                              v
                                        Harness Pod (agent)
                                          - Git clone
                                          - Agent process
                                          - Session artifacts
```

## Environment Variables

### Required

| Variable | Description | Example |
|---|---|---|
| `KOCAO_API_URL` | Control-plane API base URL | `https://kocao.example.com` |
| `KOCAO_TOKEN` | Bearer token for API authentication | `kocao_tok_...` |

### Optional

| Variable | Description | Default |
|---|---|---|
| `KOCAO_TIMEOUT` | HTTP request timeout | `15s` |
| `KOCAO_VERBOSE` | Enable debug/verbose output | `false` |

### Configuration File

Instead of environment variables, you can use a JSON config file at
`~/.config/kocao/settings.json`:

```json
{
  "api_url": "https://kocao.example.com",
  "token": "kocao_tok_...",
  "timeout": "30s",
  "verbose": false
}
```

The CLI also checks for a `settings.json` alongside the binary itself.
Priority order: env vars > explicit `--config` flag > default config files.

## Prerequisites

1. **kocao CLI installed**
   ```bash
   go install github.com/withakay/kocao/cmd/kocao@latest
   ```

2. **Cluster deployed**
   The Kocao control-plane and operator must be running in a Kubernetes cluster.
   See the main project README for deployment instructions.

3. **Agent secrets seeded**
   Run `seed-agent-secrets` to provision the required Kubernetes secrets for
   agent authentication (GitHub tokens, API keys, etc.).

4. **Network access**
   The kocao CLI must be able to reach the control-plane API URL. If running
   locally, you may need port-forwarding:
   ```bash
   kubectl port-forward svc/control-plane-api 8080:8080
   ```

## CLI Command Reference

### Sessions

```
kocao sessions ls [--json]
kocao sessions get <session-id> [--json]
kocao sessions status <session-id> [--json]
kocao sessions logs <session-id> [--tail N] [--container NAME] [--follow] [--json]
kocao sessions attach <session-id> [--driver] [--collab]
```

### Symphony (Multi-Agent Orchestration)

```
kocao symphony ls [--json]
kocao symphony get <project-name> [--json]
kocao symphony create --file <path> [--json]
kocao symphony pause <project-name> [--json]
kocao symphony resume <project-name> [--json]
kocao symphony refresh <project-name> [--json]
```

### Global Flags

```
--config <path>     Config file path (.json)
--api-url <url>     Control-plane base URL
--token <token>     Bearer token
--timeout <dur>     HTTP timeout (e.g. 15s)
--verbose           Print HTTP request/response diagnostics
--debug             Alias for --verbose
```

## Troubleshooting

### "kocao binary not found"

Install the CLI:
```bash
go install github.com/withakay/kocao/cmd/kocao@latest
```
Or build from source in the project root:
```bash
go build -o kocao ./cmd/kocao
```

### "KOCAO_TOKEN is not set"

Set the token environment variable or create a config file:
```bash
export KOCAO_TOKEN="your-token-here"
```

### "connection refused" or timeout errors

The control-plane API is not reachable. Check:
1. Is the control-plane running? `kubectl get pods -l app=control-plane-api`
2. Do you need port-forwarding? `kubectl port-forward svc/control-plane-api 8080:8080`
3. Is `KOCAO_API_URL` set to the correct URL?

### "no active run pod for workspace session"

The session exists but has no running harness pod. This can mean:
- The session is still starting (check `kocao sessions status <id>`)
- The session's run failed (check phase in status output)
- The pod was evicted (check Kubernetes events)

### "API returned HTTP 401"

Your token is invalid or expired. Regenerate it and update `KOCAO_TOKEN`.

### "API returned HTTP 404"

The session ID does not exist, or the API endpoint is not available.
Double-check the session ID with `kocao sessions ls`.

### "attach requires an interactive terminal (TTY)"

The `attach` command needs a real terminal. It cannot run inside a non-TTY
context (like a pipe or background process). Use `agent-exec.sh` for
non-interactive prompt delivery, or run attach from a real terminal.

### Session stuck in "Pending"

The harness pod may be waiting for resources. Check:
```bash
kubectl get pods -l workspace-session-id=<session-id>
kubectl describe pod <pod-name>
```
Look for scheduling issues, resource limits, or image pull errors.
