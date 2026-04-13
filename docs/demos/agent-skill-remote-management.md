# Demo: AI Assistant Managing Remote Agents

This document shows how an AI assistant uses the `kocao-agent` skill to manage
remote Kocao agent sessions. For real captured output, see the showboat demos in
`demos/agent-cli-live-demo.md` and `demos/agent-cli-errors-demo.md`.

## Scene 1: Start a remote agent

**User:** Start a remote codex agent for the kocao repo.

**Assistant:** Starting a remote agent using the `kocao agent start` wrapper.

```bash
.opencode/skills/kocao-agent/scripts/agent-start.sh \
  --repo https://github.com/withakay/kocao \
  --agent codex \
  --timeout 5m
```

The wrapper returns JSON by default so the assistant can capture the `runId` for
follow-up actions.

## Scene 2: Send a task to the agent

**User:** Ask it to review the README and suggest improvements.

**Assistant:** Sending the task through `kocao agent exec`.

```bash
.opencode/skills/kocao-agent/scripts/agent-exec.sh <run-id> \
  --prompt "Review README.md and suggest improvements for clarity and onboarding."
```

## Scene 3: Check agent status

**User:** How is it doing?

**Assistant:** Checking the current agent session state.

```bash
.opencode/skills/kocao-agent/scripts/agent-status.sh <run-id>
```

## Scene 4: View logs

**User:** Show me what it has been doing.

**Assistant:** Fetching recent agent session events.

```bash
.opencode/skills/kocao-agent/scripts/agent-logs.sh <run-id> --tail 50 --no-json
```

## Scene 5: List active agents

**User:** What other remote agents are running?

**Assistant:** Listing active agent sessions.

```bash
.opencode/skills/kocao-agent/scripts/agent-list.sh
```

## Scene 6: Stop the agent

**User:** That agent is done. Stop it.

**Assistant:** Stopping the agent session.

```bash
.opencode/skills/kocao-agent/scripts/agent-stop.sh <run-id>
```

## Summary

This skill wraps the actual `kocao agent` commands so assistants can:

1. start a remote agent run
2. send a task through `kocao agent exec`
3. inspect agent status
4. review event logs
5. list active agent sessions
6. stop the finished agent run
