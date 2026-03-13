---
codex:
  command: codex app-server
  approval_policy: untrusted
  thread_sandbox: workspace-write
  turn_sandbox_policy: workspace-write
---

You are working on GitHub issue `{{.issue.repository}}#{{.issue.number}}`.

Title: {{.issue.title}}

Follow the repository instructions, make the smallest safe change that resolves the issue, and stop if the workflow contract or local environment prevents safe progress.
