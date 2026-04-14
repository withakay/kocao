# Remote Agent Multi-Workflow Contract Demo

*2026-04-14T15:00:00Z - CLI+API contract walkthrough for researcher / implementer / reviewer task chaining*

This demo documents the contract exercised by `TestRemoteAgentOrchestrationContract_MultiAgentWorkflowViaCLIAndAPI`.
It does not claim live harness execution, Kind coverage, or showboat verification.

The workflow used in the test is:

- `researcher` investigates the bug and produces notes
- `implementer` receives `research-notes.md` as an explicit input artifact and turns it into a patch
- `reviewer` receives `fix.patch` as an explicit input artifact and emits review comments

The important contract is that each task keeps its own identity, status, transcript, and input/output artifact list even though the larger workflow is only modeled through API and CLI boundaries.

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

## 2. Dispatch the initial task through the CLI

```bash
kocao remote-agents tasks dispatch \
  --agent researcher --pool workflow \
  --prompt "Research the regression and summarize the likely root cause." \
  --output json
```

```output
{"id":"task-research","agentName":"researcher","poolName":"workflow","state":"assigned"}
```

## 3. Dispatch the linked follow-up tasks through the API

```bash
curl -fsS -X POST "$KOCAO_API_URL/api/v1/remote-agent-tasks" \
  -H "Authorization: Bearer $KOCAO_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "target": {"agentName": "implementer", "poolName": "workflow"},
    "prompt": "Implement the fix described in research-notes.md.",
    "inputArtifacts": [
      {"name": "research-notes.md", "kind": "report", "path": "/workspace/research-notes.md"}
    ]
  }'

curl -fsS -X POST "$KOCAO_API_URL/api/v1/remote-agent-tasks" \
  -H "Authorization: Bearer $KOCAO_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "target": {"agentName": "reviewer", "poolName": "workflow"},
    "prompt": "Review fix.patch and confirm whether it is ready to merge.",
    "inputArtifacts": [
      {"name": "fix.patch", "kind": "patch", "path": "/workspace/fix.patch"}
    ]
  }'
```

```output
{"id":"task-implement","agentName":"implementer","poolName":"workflow","state":"assigned","inputArtifacts":[{"name":"research-notes.md","kind":"report","path":"/workspace/research-notes.md"}]}
{"id":"task-review","agentName":"reviewer","poolName":"workflow","state":"assigned","inputArtifacts":[{"name":"fix.patch","kind":"patch","path":"/workspace/fix.patch"}]}
```

At this point the workflow has three independent task records, and the follow-up steps explicitly declare which prior artifact they consume.

## 4. Inspect the coordinated workflow state

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

The contract guarantee here is narrower than a live orchestration runtime: the API and CLI preserve per-agent status boundaries and linked task metadata.

## 5. Retrieve transcripts after completion

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

This is the persistent transcript contract: transcripts remain available after the modeled task completion.

## 6. Retrieve artifact linkage for each agent

```bash
kocao remote-agents tasks artifacts task-research --output json
kocao remote-agents tasks artifacts task-implement --output json
kocao remote-agents tasks artifacts task-review --output json
```

```output
{"taskId":"task-research","inputArtifacts":[],"outputArtifacts":[{"name":"research-notes.md","kind":"report","path":"/workspace/research-notes.md"}]}
{"taskId":"task-implement","inputArtifacts":[{"name":"research-notes.md","kind":"report","path":"/workspace/research-notes.md"}],"outputArtifacts":[{"name":"fix.patch","kind":"patch","path":"/workspace/fix.patch"}]}
{"taskId":"task-review","inputArtifacts":[{"name":"fix.patch","kind":"patch","path":"/workspace/fix.patch"}],"outputArtifacts":[{"name":"review.md","kind":"report","path":"/workspace/review.md"}]}
```

Each task advertises both the artifact it consumed and the artifact it produced, so the researcher -> implementer -> reviewer chain is validated at the contract level without claiming live runtime execution.
