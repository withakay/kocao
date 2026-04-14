package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoteAgentsListTable(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agents" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"remoteAgents": []map[string]any{{
			"id": "agent-1", "name": "reviewer", "poolName": "backend", "availability": "idle", "workspaceSessionId": "ws-1", "lastActivityAt": "2026-04-14T12:00:00Z",
		}}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "agents", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"AGENT", "reviewer", "backend", "agent-1"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRemoteAgentGetByName(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agents" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"remoteAgents": []map[string]any{{
			"id": "agent-1", "name": "reviewer", "poolName": "backend", "availability": "busy", "currentTaskId": "task-9", "workspaceSessionId": "ws-1",
			"currentSession": map[string]any{"sessionId": "sess-1", "harnessRunId": "run-1"},
		}}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "agents", "get", "reviewer", "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "reviewer"`) {
		t.Fatalf("expected reviewer JSON, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"currentTaskId": "task-9"`) {
		t.Fatalf("expected current task JSON, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskDispatchWithPromptFile(t *testing.T) {
	t.Setenv(EnvToken, "")
	tempDir := t.TempDir()
	promptPath := filepath.Join(tempDir, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte("Review the patch"), 0o600); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/remote-agents" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"remoteAgents": []map[string]any{{
				"id": "agent-1", "name": "reviewer", "poolName": "backend", "workspaceSessionId": "ws-1",
			}}})
		case r.URL.Path == "/api/v1/remote-agent-tasks" && r.Method == http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			target := payload["target"].(map[string]any)
			if target["agentId"] != "agent-1" {
				t.Fatalf("agentId = %#v", target["agentId"])
			}
			if _, ok := target["agentName"]; ok {
				t.Fatalf("unexpected agentName in target: %#v", target["agentName"])
			}
			if payload["prompt"] != "Review the patch" {
				t.Fatalf("prompt = %#v", payload["prompt"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "task-1", "state": "assigned", "agentName": "reviewer", "poolName": "backend", "attempt": 1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "dispatch", "--agent", "reviewer", "--pool", "backend", "--prompt-file", promptPath}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dispatched task task-1 to reviewer (pool backend)") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsAmbiguousNamedAgent(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agents" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"remoteAgents": []map[string]any{
			{"id": "agent-1", "name": "reviewer", "poolName": "backend", "workspaceSessionId": "ws-1"},
			{"id": "agent-2", "name": "reviewer", "poolName": "frontend", "workspaceSessionId": "ws-2"},
		}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "dispatch", "--agent", "reviewer", "--prompt", "Review the patch"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), `remote agent "reviewer" is ambiguous; specify --pool or --workspace`) {
		t.Fatalf("expected ambiguity guidance, got: %s", stderr.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsConflictingAgentFlags(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"remote-agents", "tasks", "dispatch", "--agent", "reviewer", "--agent-id", "agent-1", "--prompt", "Review the patch"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--agent and --agent-id are mutually exclusive") {
		t.Fatalf("expected conflicting flag error, got: %s", stderr.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsMissingSelector(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"remote-agents", "tasks", "dispatch", "--prompt", "Review the patch"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "must specify --agent <name> or --agent-id <id>") {
		t.Fatalf("expected missing selector error, got: %s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsMissingPrompt(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"remote-agents", "tasks", "dispatch", "--agent-id", "agent-1"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "must specify --prompt <text> or --prompt-file <path>") {
		t.Fatalf("expected missing prompt error, got: %s", stderr.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsBlankPromptFile(t *testing.T) {
	t.Setenv(EnvToken, "")
	tempDir := t.TempDir()
	promptPath := filepath.Join(tempDir, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte(" \n\t "), 0o600); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"remote-agents", "tasks", "dispatch", "--agent-id", "agent-1", "--prompt-file", promptPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "task prompt cannot be empty") {
		t.Fatalf("expected blank prompt file error, got: %s", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsPromptFlagConflict(t *testing.T) {
	t.Setenv(EnvToken, "")
	tempDir := t.TempDir()
	promptPath := filepath.Join(tempDir, "prompt.txt")
	if err := os.WriteFile(promptPath, []byte("Review the patch"), 0o600); err != nil {
		t.Fatalf("write prompt file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"remote-agents", "tasks", "dispatch", "--agent-id", "agent-1", "--prompt", "Review the patch", "--prompt-file", promptPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "--prompt and --prompt-file are mutually exclusive") {
		t.Fatalf("expected prompt conflict error, got: %s", stderr.String())
	}
}

func TestRemoteAgentTaskDispatchRejectsUnsupportedOutputFormat(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"remote-agents", "tasks", "dispatch", "--agent-id", "agent-1", "--prompt", "Review the patch", "--output", "yaml"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), `unsupported output format "yaml"`) {
		t.Fatalf("expected unsupported output error, got: %s", stderr.String())
	}
}

func TestRemoteAgentTaskDispatchWithWorkspaceDisambiguation(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/v1/remote-agents" && r.Method == http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"remoteAgents": []map[string]any{
				{"id": "agent-1", "name": "reviewer", "poolName": "backend", "workspaceSessionId": "ws-1"},
				{"id": "agent-2", "name": "reviewer", "poolName": "backend", "workspaceSessionId": "ws-2"},
			}})
		case r.URL.Path == "/api/v1/remote-agent-tasks" && r.Method == http.MethodPost:
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			target := payload["target"].(map[string]any)
			if target["agentId"] != "agent-2" {
				t.Fatalf("agentId = %#v, want agent-2", target["agentId"])
			}
			if payload["prompt"] != "Review the patch" {
				t.Fatalf("prompt = %#v", payload["prompt"])
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "task-2", "state": "assigned", "agentName": "reviewer", "poolName": "backend", "attempt": 1,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "dispatch", "--agent", "reviewer", "--workspace", "ws-2", "--prompt", "Review the patch"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dispatched task task-2 to reviewer (pool backend)") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestRemoteAgentTasksListActiveFilter(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"remoteAgentTasks": []map[string]any{
			{"id": "task-1", "state": "running", "agentName": "reviewer", "poolName": "backend", "attempt": 1, "lastTransitionAt": "2026-04-14T12:00:00Z"},
			{"id": "task-2", "state": "completed", "agentName": "implementer", "poolName": "backend", "attempt": 1, "lastTransitionAt": "2026-04-14T11:00:00Z"},
		}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "list", "--active"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "task-1") {
		t.Fatalf("expected active task, got: %s", stdout.String())
	}
	if strings.Contains(stdout.String(), "task-2") {
		t.Fatalf("completed task should be filtered out, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskGetJSON(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks/task-1" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "task-1", "state": "completed", "agentName": "reviewer", "result": map[string]any{"summary": "Looks good"}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "get", "task-1", "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"summary": "Looks good"`) {
		t.Fatalf("expected task summary in json, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskCancel(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks/task-1/cancel" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "task-1", "state": "cancelled", "agentName": "reviewer", "cancelledAt": "2026-04-14T12:10:00Z"})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "cancel", "task-1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "cancelled task task-1") {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "State:") || !strings.Contains(stdout.String(), "cancelled") {
		t.Fatalf("expected cancelled summary, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskTranscriptTable(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks/task-1/transcript" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "task-1", "transcript": []map[string]any{{
			"sequence": 1, "at": "2026-04-14T12:00:00Z", "role": "user", "kind": "prompt", "text": "Review the patch",
		}}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "transcript", "task-1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	for _, want := range []string{"SEQ", "prompt", "Review the patch"} {
		if !strings.Contains(stdout.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, stdout.String())
		}
	}
}

func TestRemoteAgentTaskTranscriptEmptyTable(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks/task-1/transcript" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"taskId": "task-1", "transcript": []map[string]any{}})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "transcript", "task-1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "task task-1 has no transcript entries") {
		t.Fatalf("expected empty transcript message, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskArtifactsJSON(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks/task-1/artifacts" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":          "task-1",
			"inputArtifacts":  []map[string]any{{"name": "spec.md", "kind": "file"}},
			"outputArtifacts": []map[string]any{{"name": "review.md", "kind": "report", "uri": "s3://bucket/review.md"}},
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "artifacts", "task-1", "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"name": "review.md"`) {
		t.Fatalf("expected output artifact JSON, got: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), `"name": "spec.md"`) {
		t.Fatalf("expected input artifact JSON, got: %s", stdout.String())
	}
}

func TestRemoteAgentTaskArtifactsEmptyTable(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/remote-agent-tasks/task-1/artifacts" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"taskId":          "task-1",
			"inputArtifacts":  []map[string]any{},
			"outputArtifacts": []map[string]any{},
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "remote-agents", "tasks", "artifacts", "task-1"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "task task-1 has no artifacts") {
		t.Fatalf("expected empty artifacts message, got: %s", stdout.String())
	}
}
