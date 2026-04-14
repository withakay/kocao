# Remote Agent Multi-Workflow Demo

*2026-04-14T15:00:00Z - reviewer / implementer / researcher orchestration walkthrough*

This demo shows one coordinated remote-agent workflow where three named agents own different tasks:

- `researcher` investigates the bug and produces notes
- `implementer` turns the notes into a patch
- `reviewer` checks the patch and emits review comments

The important contract is that each task keeps its own identity, status, transcript, and artifact list even though the work belongs to one larger workflow.

## 1. Create the logical pool and named agents

```bash
export KOCAO_API_URL=http://127.0.0.1:8080
export KOCAO_TOKEN=dev-bootstrap

curl -fsS -X POST "$KOCAO_API_URL/api/v1/remote-agent-pools" \
  -H "Authorization: Bearer $KOCAO_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"workflow","displayName":"Workflow Agents"}'

for agent in researcher implementer reviewer; do
  curl -fsS -X POST "$KOCAO_API_URL/api/v1/remote-agents" \
    -H "Authorization: Bearer $KOCAO_TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"$agent\",\"poolName\":\"workflow\"}"
done
```

## 2. Dispatch the workflow tasks through the CLI

```bash
kocao remote-agents tasks dispatch \
  --agent researcher --pool workflow \
  --prompt "Research the regression and summarize the likely root cause." \
  --output json

kocao remote-agents tasks dispatch \
  --agent implementer --pool workflow \
  --prompt "Implement the fix described in research-notes.md." \
  --output json

kocao remote-agents tasks dispatch \
  --agent reviewer --pool workflow \
  --prompt "Review fix.patch and confirm whether it is ready to merge." \
  --output json
```

```output
{"id":"task-research","agentName":"researcher","poolName":"workflow","state":"assigned"}
{"id":"task-implement","agentName":"implementer","poolName":"workflow","state":"assigned"}
{"id":"task-review","agentName":"reviewer","poolName":"workflow","state":"assigned"}
```

At this point the workflow has three independent task records instead of one shared opaque job.

## 3. Inspect the coordinated workflow state

```bash
kocao remote-agents tasks list --output json
```

```output
[
  {"id":"task-research","agentName":"researcher","poolName":"workflow","state":"completed","result":{"summary":"Research completed"}},
  {"id":"task-implement","agentName":"implementer","poolName":"workflow","state":"completed","result":{"summary":"Implementation completed"}},
  {"id":"task-review","agentName":"reviewer","poolName":"workflow","state":"completed","result":{"summary":"Review completed"}}
]
```

The orchestration layer keeps per-agent status boundaries intact, which is the core multi-agent coordination guarantee.

## 4. Retrieve transcripts after completion

```bash
kocao remote-agents tasks transcript task-research --output json
kocao remote-agents tasks transcript task-implement --output json
kocao remote-agents tasks transcript task-review --output json
```

```output
{"taskId":"task-research","transcript":[{"role":"user","text":"Research the regression and summarize the likely root cause."},{"role":"assistant","text":"Likely root cause narrowed to orchestration state handling during retries."}]}
{"taskId":"task-implement","transcript":[{"role":"user","text":"Implement the fix described in research-notes.md."},{"role":"assistant","text":"Patched the retry state transition and added validation coverage."}]}
{"taskId":"task-review","transcript":[{"role":"user","text":"Review fix.patch and confirm whether it is ready to merge."},{"role":"assistant","text":"Patch looks correct; add one assertion for transcript retention before merge."}]}
```

This is the persistent transcript contract: transcripts remain available even after the live task is done.

## 5. Retrieve output artifacts for each agent

```bash
kocao remote-agents tasks artifacts task-research --output json
kocao remote-agents tasks artifacts task-implement --output json
kocao remote-agents tasks artifacts task-review --output json
```

```output
{"taskId":"task-research","outputArtifacts":[{"name":"research-notes.md","kind":"report","path":"/workspace/research-notes.md"}]}
{"taskId":"task-implement","outputArtifacts":[{"name":"fix.patch","kind":"patch","path":"/workspace/fix.patch"}]}
{"taskId":"task-review","outputArtifacts":[{"name":"review.md","kind":"report","path":"/workspace/review.md"}]}
```

Each output stays attached to the task that produced it, so operators can trace how the workflow progressed from research to implementation to review.
