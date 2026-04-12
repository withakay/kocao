# Demo: AI Assistant Managing Remote Sessions

This demo shows an AI assistant using the `kocao-agent` skill to manage remote
Kocao workspace sessions.

## Prerequisites

- Kocao cluster deployed and reachable
- `kocao` CLI installed and configured (`KOCAO_API_URL`, `KOCAO_TOKEN`)
- OpenCode with the `kocao-agent` skill loaded

---

## Scene 1: Start a remote session

**User:** Start a remote session for the kocao repo and label it for codex work.

**Assistant:** Creating a workspace session for the kocao repository.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-start.sh \
    --repo https://github.com/withakay/kocao \
    --agent codex
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

**Assistant:** Session `ws-a1b2c3d4` is running. The `codex` label makes the
session easy to identify, and the harness pod is ready.

---

## Scene 2: Send a task to the session

**User:** Ask it to review the README and suggest improvements.

**Assistant:** Sending the task through the exec endpoint.

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

**Assistant:** Task sent to session `ws-a1b2c3d4`. If the control-plane did not
support `/exec`, I would fall back to `kocao sessions attach ... --driver`.

---

## Scene 3: Check session status

**User:** How is it doing?

**Assistant:** Checking the session status.

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

**Assistant:** The session is still active and the harness run is healthy.

---

## Scene 4: View logs

**User:** Show me what it's been doing.

**Assistant:** Fetching the last 50 log lines.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-logs.sh ws-a1b2c3d4 --tail 50 --no-json
```

```text
[agent] Cloned https://github.com/withakay/kocao (branch: main)
[agent] Analyzing README.md...
[agent] Found 3 areas for improvement:
[agent]   1. Missing quickstart section
[agent]   2. Installation instructions reference deprecated flag
[agent]   3. API reference section is incomplete
[agent] Generating suggestions...
[agent] Task complete.
```

---

## Scene 5: List active sessions

**User:** What other sessions are running?

**Assistant:** Listing active sessions.

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

---

## Scene 6: Stop the session

**User:** That session is done. Stop it.

**Assistant:** Stopping the workspace session.

```bash
$ .opencode/skills/kocao-agent/scripts/agent-stop.sh ws-a1b2c3d4
```

```json
{
  "status": "stopped",
  "sessionId": "ws-a1b2c3d4"
}
```

---

## Summary

This demo covered the full lifecycle:

1. create a remote workspace session
2. send a task through the optional exec endpoint
3. inspect session status
4. review logs
5. list active sessions
6. stop the finished session
