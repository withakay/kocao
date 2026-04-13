## 1. Skill Scaffold

- [ ] 1.1 Create skill directory `.opencode/skills/kocao-agent/` with `SKILL.md` skeleton, `scripts/`, and `reference/` directories

## 2. CLI Wrapper Scripts

- [ ] 2.1 Write `scripts/agent-list.sh` — wraps `kocao agent list --output json`, accepts optional `--workspace` filter
- [ ] 2.2 Write `scripts/agent-start.sh` — wraps `kocao agent start`, accepts `--repo`, `--agent`, `--workspace`, `--timeout` arguments
- [ ] 2.3 Write `scripts/agent-stop.sh` — wraps `kocao agent stop <session-id>`
- [ ] 2.4 Write `scripts/agent-logs.sh` — wraps `kocao agent logs <session-id>`, supports `--follow` and `--tail`
- [ ] 2.5 Write `scripts/agent-exec.sh` — wraps `kocao agent exec <session-id> --prompt "..."`, returns JSON events
- [ ] 2.6 Write `scripts/agent-status.sh` — wraps `kocao agent status <session-id> --output json`

## 3. Skill Documentation

- [ ] 3.1 Write `SKILL.md` with trigger descriptions, workflows (start, exec, logs, status, stop, multi-agent), and usage examples
- [ ] 3.2 Write `reference/agents.md` — supported agents catalog (opencode, claude, codex, pi), required env vars, prerequisites, troubleshooting

## 4. Demo

- [ ] 4.1 Write Showboat demo document: AI assistant managing a remote agent lifecycle end-to-end with annotated output
