# Demo: AI Assistant Managing Remote Agents

This demo shows an AI assistant (OpenCode) using the `kocao-agent` skill to
manage remote coding agents on a Kocao cluster.

## Prerequisites

- Kocao cluster deployed and accessible
- `kocao` CLI installed and configured (`KOCAO_API_URL`, `KOCAO_TOKEN`)
- OpenCode with the `kocao-agent` skill loaded

---

## Scene 1: Start a remote agent

**User:** Start a codex agent on the cluster to work on the kocao repo.

**Assistant:** Starting a codex agent for the kocao repository.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-start.sh \
    --repo https://github.com/withakay/kocao \
    --agent codex
```

```json
{
  "id": "ws-a1b2c3d4",
  "displayName": "codex-1712937600",
  "repoURL": "https://github.com/withakay/kocao",
  "phase": "Running",
  "createdAt": "2026-04-12T10:00:00Z"
}
```

**Assistant:** Session `ws-a1b2c3d4` is running. The codex agent is ready and
has cloned the repository.

---

## Scene 2: Send a task to the agent

**User:** Ask it to review the README and suggest improvements.

**Assistant:** Sending the task to the remote agent.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-exec.sh ws-a1b2c3d4 \
    --prompt "Review the README.md file and suggest improvements for clarity, completeness, and developer experience."
```

```json
{
  "status": "sent",
  "sessionId": "ws-a1b2c3d4"
}
```

**Assistant:** Task sent to agent `ws-a1b2c3d4`. The codex agent is now
reviewing the README. Let me check on its progress.

---

## Scene 3: Check agent status

**User:** How is it doing?

**Assistant:** Checking the agent status.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-status.sh ws-a1b2c3d4
```

```json
{
  "session": {
    "id": "ws-a1b2c3d4",
    "displayName": "codex-1712937600",
    "phase": "Running",
    "repoURL": "https://github.com/withakay/kocao",
    "createdAt": "2026-04-12T10:00:00Z"
  },
  "run": {
    "id": "hr-e5f6g7h8",
    "phase": "Running",
    "podName": "harness-ws-a1b2c3d4-xk9m2"
  }
}
```

**Assistant:** Agent `ws-a1b2c3d4` is still running. The harness run `hr-e5f6g7h8`
is active in pod `harness-ws-a1b2c3d4-xk9m2`.

---

## Scene 4: View agent logs

**User:** Show me what it's been doing.

**Assistant:** Fetching the last 50 lines of logs.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-logs.sh ws-a1b2c3d4 --tail 50 --no-json
```

```
[agent] Cloned https://github.com/withakay/kocao (branch: main)
[agent] Analyzing README.md...
[agent] Found 3 areas for improvement:
[agent]   1. Missing quickstart section
[agent]   2. Installation instructions reference deprecated flag
[agent]   3. API reference section is incomplete
[agent] Generating suggestions...
[agent] Creating branch: readme-improvements
[agent] Committing changes...
[agent] Opening pull request...
[agent] PR created: #47 "Improve README clarity and completeness"
[agent] Task complete.
```

**Assistant:** The codex agent has completed its review. It found 3 areas for
improvement and created PR #47 with the suggested changes.

---

## Scene 5: List all running agents

**User:** What other agents are running?

**Assistant:** Listing all active sessions.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-list.sh
```

```json
{
  "workspaceSessions": [
    {
      "id": "ws-a1b2c3d4",
      "displayName": "codex-1712937600",
      "phase": "Running",
      "createdAt": "2026-04-12T10:00:00Z"
    },
    {
      "id": "ws-z9y8x7w6",
      "displayName": "opencode-api-tests",
      "phase": "Running",
      "createdAt": "2026-04-12T09:30:00Z"
    }
  ]
}
```

**Assistant:** There are 2 active sessions:
1. `ws-a1b2c3d4` (codex) — the one we started for README review
2. `ws-z9y8x7w6` (opencode) — running API tests, started 30 minutes ago

---

## Scene 6: Stop the agent

**User:** The codex agent is done. Stop it.

**Assistant:** Stopping the codex session.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-stop.sh ws-a1b2c3d4
```

```json
{
  "status": "stopped",
  "sessionId": "ws-a1b2c3d4"
}
```

**Assistant:** Session `ws-a1b2c3d4` has been stopped. The codex agent's work
(PR #47) is preserved in GitHub.

---

## Summary

This demo showed the full lifecycle of managing a remote coding agent:

1. **Start** — Created a codex agent session on the cluster
2. **Exec** — Sent a review task to the running agent
3. **Status** — Checked the agent's current phase and run info
4. **Logs** — Viewed the agent's output and progress
5. **List** — Saw all active agent sessions
6. **Stop** — Terminated the completed session

The `kocao-agent` skill provides an AI assistant with full control over remote
coding agents, enabling multi-agent workflows where tasks are distributed
across specialized agents running in isolated Kubernetes pods.
