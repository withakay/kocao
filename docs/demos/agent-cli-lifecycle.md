# Agent CLI Lifecycle Demo

This document is a concise walkthrough of the `kocao agent` command surface.
For real captured output against a live cluster, use `demos/agent-cli-live-demo.md`.

## Prerequisites

1. Deploy Kocao to a dev cluster.
2. Seed agent secrets.
3. Export `KOCAO_API_URL` and `KOCAO_TOKEN` or configure `~/.config/kocao/settings.json`.

## Start an agent

```bash
kocao agent start --repo https://github.com/withakay/kocao --agent codex --timeout 5m
```

This creates or reuses a workspace session, creates a harness run with
`agentSession.agent=<name>`, and polls until the agent session becomes ready or
the timeout expires.

Use `--output json` when you want machine-readable output.

## Check status

```bash
kocao agent status <run-id>
kocao agent status <run-id> --output json
```

The status payload includes run ID, session ID, runtime, agent, phase,
workspace session ID, and created time.

## List active agents

```bash
kocao agent list
kocao agent list --workspace <workspace-id> --output json
```

The `list` command also supports `--output yaml`.

## Send a prompt

```bash
kocao agent exec <run-id> --prompt "What files are in the repo?"
kocao agent exec <run-id> "What files are in the repo?"
```

Use `--output json` to get the full event payloads.

## Stream logs

```bash
kocao agent logs <run-id> --tail 50
kocao agent logs <run-id> --follow --output json
```

## Stop the session

```bash
kocao agent stop <run-id>
kocao agent stop <run-id> --json
```

## Notes

- Use `--image`, `--image-pull-secret`, and `--egress-mode` when targeting
  remote clusters.
- If the session stays in `Provisioning`, inspect the harness pod logs and
  events in Kubernetes.
- `demos/agent-cli-errors-demo.md` contains captured failure modes and error
  output.
