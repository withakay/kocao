# Kocao Agent CLI Error Handling Demo

*2026-04-12T21:08:48Z by Showboat 0.6.1*
<!-- showboat-id: 967d0896-2463-4bbe-8f1f-ec655101a6b4 -->

This demo exercises error handling in the kocao agent CLI against a live MicroK8s cluster.

Try to get status for a non-existent run ID.

```bash
go run ./cmd/kocao agent status nonexistent-run-id || true
```

```output
error: harness run not found
exit status 1
```

Try to exec without providing a prompt.

```bash
go run ./cmd/kocao agent exec some-run-id || true
```

```output
error: usage: kocao agent exec <run-id> [--prompt <text> | <text>]
exit status 2
```

Try to start an agent without required flags.

```bash
go run ./cmd/kocao agent start || true
```

```output
error: usage: kocao agent start --repo <url> --agent <name>: missing required flag --repo
exit status 2
```

Try to stop a non-existent run.

```bash
go run ./cmd/kocao agent stop nonexistent-run-id || true
```

```output
error: harness run not found
exit status 1
```

Try agent logs for a non-existent run.

```bash
go run ./cmd/kocao agent logs nonexistent-run-id --tail 5 || true
```

```output
error: harness run not found
exit status 1
```

Try start with only --repo (missing --agent).

```bash
go run ./cmd/kocao agent start --repo https://github.com/withakay/kocao || true
```

```output
error: usage: kocao agent start --repo <url> --agent <name>: missing required flag --agent
exit status 2
```

Try an unknown subcommand.

```bash
go run ./cmd/kocao agent bogus || true
```

```output
error: unknown agent subcommand "bogus"
exit status 2
```
