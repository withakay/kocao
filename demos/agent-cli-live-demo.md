# Kocao Agent CLI Live Demo

*2026-04-13T20:55:00Z — full happy-path verified after transport & list fixes*

This demo captures the complete agent CLI workflow against the MicroK8s cluster: start a session, list active sessions, inspect status, view logs, execute a prompt, and stop the session.

## 1. Start an agent session

```bash
export KOCAO_API_URL=http://127.0.0.1:8080 KOCAO_TOKEN=dev-bootstrap
kocao agent start \
  --repo https://github.com/withakay/kocao \
  --agent opencode --revision main \
  --image ghcr.io/withakay/kocao/harness-runtime:dev-microk8s-amd64fix \
  --image-pull-secret ghcr-pull --egress-mode full \
  --timeout 7m --output json
```

```output
Creating workspace session... done
Starting harness run... done
Waiting for agent session... ready
{
  "sessionId": "ses_2775f78e1ffeMh4b3ookLBRjsW",
  "runId": "7d5108e3a844b192c8f754cf5afed99b",
  "agent": "opencode",
  "phase": "Ready"
}
```

## 2. Check agent status

```bash
kocao agent status 7d5108e3a844b192c8f754cf5afed99b --output json
```

```output
{
  "sessionId": "ses_2775f78e1ffeMh4b3ookLBRjsW",
  "runId": "7d5108e3a844b192c8f754cf5afed99b",
  "displayName": "",
  "runtime": "sandbox-agent",
  "agent": "opencode",
  "phase": "Ready",
  "workspaceSessionId": "",
  "createdAt": "0001-01-01T00:00:00Z"
}
```

## 3. List active agent sessions

```bash
kocao agent list --output json | jq '.[0:2]'
```

```output
[
  {
    "sessionId": "ses_2775f78e1ffeMh4b3ookLBRjsW",
    "runId": "7d5108e3a844b192c8f754cf5afed99b",
    "displayName": "cool-sutherland-ed99b",
    "runtime": "sandbox-agent",
    "agent": "opencode",
    "phase": "Ready",
    "workspaceSessionId": "ea2e5a6636ca5e1de047b07b38b171cf",
    "createdAt": "2026-04-13T20:54:59Z"
  }
]
```

## 4. View agent logs

Logs now return meaningful JSON-RPC event payloads with proper sequence numbers and timestamps.

```bash
kocao agent logs 7d5108e3a844b192c8f754cf5afed99b --tail 1 --output json
```

```output
{"seq":3,"timestamp":"2026-04-13T20:55:15.0613341Z","data":{"jsonrpc":"2.0","method":"session/update","params":{"sessionId":"ses_2775f78e1ffeMh4b3ookLBRjsW","update":{"availableCommands":[...],"sessionUpdate":"available_commands_update"}}}}
```

## 5. Execute a prompt

```bash
kocao agent exec 7d5108e3a844b192c8f754cf5afed99b \
  --prompt "What is 2+2? Reply with just the number." --output json
```

```output
{
  "events": [
    {
      "seq": 9,
      "timestamp": "2026-04-13T20:55:34.655309752Z",
      "data": {
        "id": 1,
        "jsonrpc": "2.0",
        "result": {
          "_meta": {},
          "stopReason": "end_turn",
          "usage": {
            "cachedWriteTokens": 14040,
            "inputTokens": 61,
            "outputTokens": 22,
            "totalTokens": 14123
          }
        }
      }
    }
  ]
}
```

## 6. Stop the agent session

```bash
kocao agent stop 7d5108e3a844b192c8f754cf5afed99b --json
```

```output
{
  "session": {
    "sessionId": "ses_2775f78e1ffeMh4b3ookLBRjsW",
    "runId": "7d5108e3a844b192c8f754cf5afed99b",
    "displayName": "",
    "runtime": "sandbox-agent",
    "agent": "opencode",
    "phase": "Completed",
    "workspaceSessionId": "",
    "createdAt": "0001-01-01T00:00:00Z"
  },
  "status": "stopped"
}
```

## 7. Verify pod health (infrastructure check)

```bash
POD=$(kubectl --context microk8s -n kocao-system get harnessruns \
  --sort-by=.metadata.creationTimestamp \
  -o jsonpath="{.items[-1:].status.podName}")
echo pod=$POD
kubectl --context microk8s -n kocao-system get pod "$POD"
kubectl --context microk8s -n kocao-system exec "$POD" -c harness -- \
  curl -fsS http://127.0.0.1:2468/v1/health
```

```output
pod=cool-sutherland-ed99b
NAME                     READY   STATUS    RESTARTS   AGE
cool-sutherland-ed99b    2/2     Running   0          45s
{"status":"ok"}
```
