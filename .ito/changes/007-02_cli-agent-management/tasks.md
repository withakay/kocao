## 1. Foundation

- [ ] 1.1 Add `AgentSession` struct and client methods to `controlplanecli/client.go` (ListAgentSessions, GetAgentSession, CreateAgentSession, StopAgentSession, SendPrompt, StreamEvents) + unit tests
- [ ] 1.2 Add `agent` parent command to `root.go` command dispatch with usage help + test

## 2. Agent Commands

- [ ] 2.1 Implement `agent list` command with `--workspace`, `--output json|table|yaml` + table-driven tests
- [ ] 2.2 Implement `agent start` command with `--workspace`, `--repo`, `--agent`, `--timeout`, `--output` — full resource chain creation with ready-state polling + tests
- [ ] 2.3 Implement `agent stop` command with `--output` + tests
- [ ] 2.4 Implement `agent logs` command with `--follow`, `--tail`, `--output` — SSE streaming with signal handling + tests
- [ ] 2.5 Implement `agent exec` command with `--prompt`, `--output` — prompt send + streaming response display + tests
- [ ] 2.6 Implement `agent status` command with `--output` — detailed key-value and JSON views + tests

## 3. Integration and Documentation

- [ ] 3.1 Integration test helpers: test harness with mock HTTP server for agent session API endpoints
- [ ] 3.2 Showboat demo document: full lifecycle walkthrough (start -> exec -> logs -> status -> stop) with annotated output
