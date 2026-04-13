# Kocao Agent CLI Live Demo

*2026-04-13T20:04:19Z by Showboat 0.6.1*
<!-- showboat-id: 1d855ddc-66a0-4867-93d3-dab1d5e5537f -->

This demo captures the current live agent CLI happy path against the MicroK8s cluster: create an agent session, inspect its status, and verify the backing pod is healthy.

The commands below talk to the port-forwarded control-plane API on localhost:18081.

```bash
export KOCAO_API_URL=http://127.0.0.1:18081 KOCAO_TOKEN=dev-bootstrap; go run ./cmd/kocao agent start --repo https://github.com/withakay/kocao --agent opencode --revision main --image ghcr.io/withakay/kocao/harness-runtime:dev-microk8s-amd64fix --image-pull-secret ghcr-pull --egress-mode full --timeout 3m --output json | tee /tmp/kocao-agent-start.json; jq -r .runId /tmp/kocao-agent-start.json > /tmp/kocao-agent-runid
```

```output
Creating workspace session... done
Starting harness run... done
Waiting for agent session... ready
{
  "sessionId": "ses_2778dcf1fffeMEhbZYKJCD1VL1",
  "runId": "df3efa59bdef4294721868af310b9794",
  "agent": "opencode",
  "phase": "Ready"
}
```

Use the returned run ID to inspect the current agent session state.

```bash
export KOCAO_API_URL=http://127.0.0.1:18081 KOCAO_TOKEN=dev-bootstrap; RUN_ID=$(cat /tmp/kocao-agent-runid); go run ./cmd/kocao agent status "$RUN_ID" --output json
```

```output
{
  "sessionId": "ses_2778dcf1fffeMEhbZYKJCD1VL1",
  "runId": "df3efa59bdef4294721868af310b9794",
  "displayName": "",
  "runtime": "sandbox-agent",
  "agent": "opencode",
  "phase": "Ready",
  "workspaceSessionId": "",
  "createdAt": "0001-01-01T00:00:00Z"
}
```

Verify the backing harness pod is running, the sidecar is present, and sandbox-agent is healthy inside the pod.

```bash
POD=$(kubectl --context microk8s -n kocao-system get harnessruns --sort-by=.metadata.creationTimestamp -o jsonpath="{.items[-1:].status.podName}"); echo pod=$POD; kubectl --context microk8s -n kocao-system get pod "$POD"; echo ===; kubectl --context microk8s -n kocao-system exec "$POD" -c harness -- /bin/sh -lc "curl -fsS http://127.0.0.1:2468/v1/health && echo"
```

```output
pod=confident-hoover-b9794
NAME                     READY   STATUS    RESTARTS   AGE
confident-hoover-b9794   2/2     Running   0          30s
===
{"status":"ok"}
```
