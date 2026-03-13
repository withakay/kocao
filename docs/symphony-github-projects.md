# Symphony GitHub Projects Guide

Kocao's Symphony integration lets you configure a GitHub Projects v2 board as an orchestration queue. Eligible issue-backed items are normalized, matched to an allowlisted repository, and materialized as Kocao `Session` plus `HarnessRun` resources.

## What the MVP Does

- Polls one GitHub Projects v2 board per `SymphonyProject`
- Considers only issue-backed items from configured repositories
- Creates or reuses a durable `Session` per issue
- Launches `HarnessRun` attempts with repository-specific auth and egress settings
- Exposes status, retries, skips, recent errors, and control actions through API, CLI, and UI

## Required Inputs

- A GitHub Projects v2 board owner and project number
- A Kubernetes `Secret` containing the GitHub token used for board polling
- One or more allowlisted repositories
- A runtime image for Symphony-created harness runs

## Example Resource

```yaml
apiVersion: kocao.withakay.github.com/v1alpha1
kind: SymphonyProject
metadata:
  name: demo
  namespace: default
spec:
  source:
    project:
      owner: withakay
      number: 42
    tokenSecretRef:
      name: github-token
    fieldName: Status
    activeStates:
      - Todo
      - In Progress
    terminalStates:
      - Done
      - Cancelled
    pollIntervalSeconds: 60
  repositories:
    - owner: withakay
      name: kocao
      repoURL: https://github.com/withakay/kocao
      branch: main
      gitAuth:
        secretName: repo-git-auth
      agentAuth:
        apiKeySecretName: llm-api-keys
      egressMode: restricted
  runtime:
    image: ghcr.io/withakay/kocao-harness:latest
    defaultRepoRevision: main
    maxConcurrentItems: 1
    retryBaseDelaySeconds: 60
    retryMaxDelaySeconds: 300
    recentSkipLimit: 20
    recentErrorLimit: 20
    activeStatusItemLimit: 20
    defaultEgressMode: restricted
```

## Operator Flow

1. Create the GitHub polling `Secret` in the same namespace as the control plane.
2. Create a `SymphonyProject` with the target board and repository allowlist.
3. Watch runtime state in `/#/symphony`, `kocao symphony get <name>`, or `GET /api/v1/symphony-projects/<name>`.
4. Use pause, resume, and refresh controls to manage orchestration.

## Validation Commands

These are the commands currently used to validate the Symphony MVP paths in this repository:

```bash
go test ./internal/operator/... ./internal/controlplaneapi/... ./internal/controlplanecli/... ./internal/auditlog/... ./cmd/control-plane-operator/...
pnpm -C web test
pnpm -C web lint
kubectl kustomize deploy/base >/dev/null
```

## Security Notes

- The board polling secret stays in the operator path and is not mounted into Symphony worker pods.
- Worker pods receive only explicitly configured per-repository `gitAuth` and `agentAuth` refs.
- Symphony audit events redact secret-shaped metadata keys before persistence.
- Restricted egress remains the default unless a repository or runtime override opts into a broader mode.

## Workflow Engine

- Workflow-driven execution details live in `docs/symphony-workflow-engine.md`.
- Use `localPath` and optional `workflowPath` only in tightly controlled environments where the controller can read repository contents directly.
