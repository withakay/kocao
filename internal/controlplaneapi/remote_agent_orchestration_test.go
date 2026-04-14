package controlplaneapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createHarnessRunForRemoteAgent(t *testing.T, api *API, run operatorv1alpha1.HarnessRun) {
	t.Helper()
	if err := api.K8s.Create(context.Background(), &run); err != nil {
		t.Fatalf("create harness run: %v", err)
	}
}

func TestRemoteAgentOrchestrationAPIContract_TaskLifecycleTranscriptAndArtifacts(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-orchestration", "orchestration", []string{
		ScopeRemoteAgentRead,
		ScopeRemoteAgentWrite,
		ScopeRemoteAgentTaskRead,
		ScopeRemoteAgentTaskWrite,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, body := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-pools", "orchestration", map[string]any{
		"name":        "reviewers",
		"displayName": "Reviewers",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create pool status = %d, want 201 (body=%s)", resp.StatusCode, string(body))
	}
	var pool remoteAgentPool
	if err := json.Unmarshal(body, &pool); err != nil {
		t.Fatalf("unmarshal pool: %v", err)
	}

	createHarnessRunForRemoteAgent(t, api, operatorv1alpha1.HarnessRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-123", Namespace: "test-ns"},
		Spec: operatorv1alpha1.HarnessRunSpec{
			WorkspaceSessionName: "ws-123",
			AgentSession: &operatorv1alpha1.AgentSessionSpec{
				Runtime: operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:   operatorv1alpha1.AgentKindClaude,
			},
		},
		Status: operatorv1alpha1.HarnessRunStatus{
			PodName: "pod-123",
			AgentSession: &operatorv1alpha1.AgentSessionStatus{
				Runtime:   operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:     operatorv1alpha1.AgentKindClaude,
				SessionID: "sas-123",
			},
		},
	})

	resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agents", "orchestration", map[string]any{
		"name":               "reviewer",
		"displayName":        "Primary Reviewer",
		"poolId":             pool.ID,
		"workspaceSessionId": "ws-123",
		"runtime":            operatorv1alpha1.AgentRuntimeSandboxAgent,
		"agent":              operatorv1alpha1.AgentKindClaude,
		"currentSession": map[string]any{
			"harnessRunId": "run-123",
			"sessionId":    "spoofed-session",
			"podName":      "spoofed-pod",
			"runtime":      operatorv1alpha1.AgentRuntimeSandboxAgent,
			"agent":        operatorv1alpha1.AgentKindCodex,
		},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent status = %d, want 201 (body=%s)", resp.StatusCode, string(body))
	}
	var agent remoteAgent
	if err := json.Unmarshal(body, &agent); err != nil {
		t.Fatalf("unmarshal agent: %v", err)
	}
	if agent.CurrentSession == nil || agent.CurrentSession.HarnessRunID != "run-123" || agent.CurrentSession.SessionID != "sas-123" || agent.CurrentSession.PodName != "pod-123" || agent.CurrentSession.Agent != operatorv1alpha1.AgentKindClaude {
		t.Fatalf("expected current session to be derived from harness run state, got %+v", agent.CurrentSession)
	}

	resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-tasks", "orchestration", map[string]any{
		"target": map[string]any{
			"agentName": "reviewer",
			"poolName":  "reviewers",
		},
		"prompt":         "Review the latest patch",
		"timeoutSeconds": 900,
		"inputArtifacts": []map[string]any{{
			"name": "patch.diff",
			"kind": "patch",
			"path": "/workspace/patch.diff",
		}},
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("dispatch task status = %d, want 201 (body=%s)", resp.StatusCode, string(body))
	}
	var task remoteAgentTask
	if err := json.Unmarshal(body, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}
	if task.AgentID != agent.ID || task.State != remoteAgentTaskStateAssigned {
		t.Fatalf("unexpected dispatched task: %+v", task)
	}

	if _, err := api.RemoteAgentOrchestration.StartTask(task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}
	if _, err := api.RemoteAgentOrchestration.AppendTranscript(task.ID, remoteAgentTranscriptEntry{Role: remoteAgentTranscriptRoleUser, Kind: "prompt", Text: "Review the latest patch"}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}
	if _, err := api.RemoteAgentOrchestration.AddOutputArtifact(task.ID, remoteAgentArtifactCreateRequest{Name: "review.md", Kind: remoteAgentArtifactKindReport, Path: "/workspace/review.md", MediaType: "text/markdown"}); err != nil {
		t.Fatalf("add output artifact: %v", err)
	}
	if _, err := api.RemoteAgentOrchestration.CompleteTask(task.ID, remoteAgentTaskCompleteRequest{Summary: "Review completed", Outcome: "approved-with-comments"}); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	resp, body = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/remote-agent-tasks/"+task.ID, "orchestration", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get task status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &task); err != nil {
		t.Fatalf("unmarshal completed task: %v", err)
	}
	if task.State != remoteAgentTaskStateCompleted || task.Result == nil || task.Result.Summary != "Review completed" {
		t.Fatalf("unexpected completed task payload: %+v", task)
	}

	resp, body = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/remote-agent-tasks/"+task.ID+"/transcript", "orchestration", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get transcript status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
	}
	var transcriptPayload struct {
		Transcript []remoteAgentTranscriptEntry `json:"transcript"`
	}
	if err := json.Unmarshal(body, &transcriptPayload); err != nil {
		t.Fatalf("unmarshal transcript: %v", err)
	}
	if len(transcriptPayload.Transcript) != 1 || transcriptPayload.Transcript[0].Text != "Review the latest patch" {
		t.Fatalf("unexpected transcript payload: %+v", transcriptPayload)
	}

	resp, body = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/remote-agent-tasks/"+task.ID+"/artifacts", "orchestration", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get artifacts status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
	}
	var artifactPayload struct {
		InputArtifacts  []remoteAgentArtifactRef `json:"inputArtifacts"`
		OutputArtifacts []remoteAgentArtifactRef `json:"outputArtifacts"`
	}
	if err := json.Unmarshal(body, &artifactPayload); err != nil {
		t.Fatalf("unmarshal artifacts: %v", err)
	}
	if len(artifactPayload.InputArtifacts) != 1 || len(artifactPayload.OutputArtifacts) != 1 {
		t.Fatalf("unexpected artifact payload: %+v", artifactPayload)
	}

	resp, body = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/remote-agents/"+agent.ID, "orchestration", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get agent status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &agent); err != nil {
		t.Fatalf("unmarshal agent after completion: %v", err)
	}
	if agent.CurrentTaskID != "" || agent.Availability != remoteAgentAvailabilityIdle {
		t.Fatalf("expected agent to be released after completion: %+v", agent)
	}
}

func TestRemoteAgentOrchestrationAPIContract_RequiresUnambiguousNamedAgentAndSupportsCancel(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	if err := api.Tokens.Create(context.Background(), "t-orchestration", "orchestration", []string{
		ScopeRemoteAgentRead,
		ScopeRemoteAgentWrite,
		ScopeRemoteAgentTaskRead,
		ScopeRemoteAgentTaskWrite,
	}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	for _, poolName := range []string{"reviewers", "implementers"} {
		resp, body := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-pools", "orchestration", map[string]any{"name": poolName})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create pool %s status = %d, want 201 (body=%s)", poolName, resp.StatusCode, string(body))
		}
	}
	var poolsPayload struct {
		RemoteAgentPools []remoteAgentPool `json:"remoteAgentPools"`
	}
	resp, body := doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/remote-agent-pools", "orchestration", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list pools status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &poolsPayload); err != nil {
		t.Fatalf("unmarshal pools: %v", err)
	}
	poolsByName := map[string]string{}
	for _, pool := range poolsPayload.RemoteAgentPools {
		poolsByName[pool.Name] = pool.ID
	}

	for _, poolName := range []string{"reviewers", "implementers"} {
		resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agents", "orchestration", map[string]any{
			"name":   "worker",
			"poolId": poolsByName[poolName],
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create agent in %s status = %d, want 201 (body=%s)", poolName, resp.StatusCode, string(body))
		}
	}

	resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-tasks", "orchestration", map[string]any{
		"target": map[string]any{"agentName": "worker"},
		"prompt": "Do the thing",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("ambiguous dispatch status = %d, want 409 (body=%s)", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %s", string(body))
	}

	resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-tasks", "orchestration", map[string]any{
		"target": map[string]any{
			"agentName": "worker",
			"poolName":  "reviewers",
		},
		"prompt": "Do the thing",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("scoped dispatch status = %d, want 201 (body=%s)", resp.StatusCode, string(body))
	}
	var task remoteAgentTask
	if err := json.Unmarshal(body, &task); err != nil {
		t.Fatalf("unmarshal task: %v", err)
	}

	resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-tasks", "orchestration", map[string]any{
		"target": map[string]any{
			"agentName": "worker",
			"poolName":  "reviewers",
		},
		"prompt": "Do another thing",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("busy dispatch status = %d, want 409 (body=%s)", resp.StatusCode, string(body))
	}
	if !strings.Contains(string(body), "busy") {
		t.Fatalf("expected busy error, got %s", string(body))
	}

	resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agent-tasks/"+task.ID+"/cancel", "orchestration", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("cancel task status = %d, want 200 (body=%s)", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &task); err != nil {
		t.Fatalf("unmarshal cancelled task: %v", err)
	}
	if task.State != remoteAgentTaskStateCancelled || task.CancelledAt == "" {
		t.Fatalf("unexpected cancelled task payload: %+v", task)
	}
}

func TestRemoteAgentOrchestrationStorePersistsTranscriptsAndArtifacts(t *testing.T) {
	store := newRemoteAgentOrchestrationStore(filepath.Join(t.TempDir(), "orchestration.jsonl"))
	service := newRemoteAgentOrchestrationService(store, "", nil, nil)

	pool, err := service.CreatePool(remoteAgentPoolCreateRequest{Name: "researchers"})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	agent, err := service.CreateAgent(remoteAgentCreateRequest{Name: "researcher", PoolID: pool.ID})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := service.DispatchTask("tester", remoteAgentTaskCreateRequest{
		Target:         remoteAgentTaskTarget{AgentID: agent.ID},
		Prompt:         "Research the API",
		InputArtifacts: []remoteAgentArtifactCreateRequest{{Name: "brief.md", Kind: remoteAgentArtifactKindFile, Path: "/workspace/brief.md"}},
	})
	if err != nil {
		t.Fatalf("dispatch task: %v", err)
	}
	if _, err := service.StartTask(task.ID); err != nil {
		t.Fatalf("start task: %v", err)
	}
	if _, err := service.AppendTranscript(task.ID, remoteAgentTranscriptEntry{Role: remoteAgentTranscriptRoleAgent, Kind: "message", Text: "Here is the research summary."}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}
	if _, err := service.AddOutputArtifact(task.ID, remoteAgentArtifactCreateRequest{Name: "summary.md", Kind: remoteAgentArtifactKindReport, Path: "/workspace/summary.md"}); err != nil {
		t.Fatalf("add output artifact: %v", err)
	}
	if _, err := service.CompleteTask(task.ID, remoteAgentTaskCompleteRequest{Summary: "Done"}); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	reloaded := newRemoteAgentOrchestrationService(store, "", nil, nil)
	restoredTask, ok := reloaded.GetTask(task.ID)
	if !ok {
		t.Fatal("expected persisted task to be loaded")
	}
	if restoredTask.Result == nil || restoredTask.Result.Summary != "Done" {
		t.Fatalf("unexpected restored task result: %+v", restoredTask)
	}
	if len(restoredTask.Transcript) != 1 || len(restoredTask.OutputArtifacts) != 1 || len(restoredTask.InputArtifacts) != 1 {
		t.Fatalf("expected transcript and artifacts to persist, got %+v", restoredTask)
	}
	for _, path := range []string{
		"/api/v1/remote-agent-pools",
		"/api/v1/remote-agents",
		"/api/v1/remote-agent-tasks",
		"/api/v1/remote-agent-tasks/{taskID}/transcript",
		"/api/v1/remote-agent-tasks/{taskID}/artifacts",
	} {
		if !strings.Contains(string(openAPISpec), path) {
			t.Fatalf("expected OpenAPI spec to contain %q", path)
		}
	}
}

func TestRemoteAgentOrchestrationAPIContract_ValidatesCurrentSessionBinding(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()

	createHarnessRunForRemoteAgent(t, api, operatorv1alpha1.HarnessRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-123", Namespace: "test-ns"},
		Spec: operatorv1alpha1.HarnessRunSpec{
			WorkspaceSessionName: "ws-123",
			AgentSession: &operatorv1alpha1.AgentSessionSpec{
				Runtime: operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:   operatorv1alpha1.AgentKindClaude,
			},
		},
		Status: operatorv1alpha1.HarnessRunStatus{
			PodName: "pod-123",
			AgentSession: &operatorv1alpha1.AgentSessionStatus{
				Runtime:   operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:     operatorv1alpha1.AgentKindClaude,
				SessionID: "sas-123",
			},
		},
	})

	_, err := api.RemoteAgentOrchestration.CreateAgent(remoteAgentCreateRequest{
		Name:               "reviewer",
		WorkspaceSessionID: "ws-123",
		CurrentSession: &remoteAgentSessionBinding{
			HarnessRunID: "run-123",
			SessionID:    "spoofed-session",
			PodName:      "spoofed-pod",
			Runtime:      operatorv1alpha1.AgentRuntimeSandboxAgent,
			Agent:        operatorv1alpha1.AgentKindCodex,
		},
	})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}

	_, err = api.RemoteAgentOrchestration.CreateAgent(remoteAgentCreateRequest{
		Name:               "reviewer-2",
		WorkspaceSessionID: "ws-123",
		CurrentSession:     &remoteAgentSessionBinding{SessionID: "sas-123"},
	})
	if err == nil || !strings.Contains(err.Error(), "currentSession.harnessRunId required") {
		t.Fatalf("expected harnessRunId validation error, got %v", err)
	}
}

func TestRemoteAgentOrchestrationService_ArtifactAndTranscriptMutationRequiresActiveTask(t *testing.T) {
	service := newRemoteAgentOrchestrationService(newRemoteAgentOrchestrationStore(""), "", nil, nil)

	agent, err := service.CreateAgent(remoteAgentCreateRequest{Name: "reviewer"})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := service.DispatchTask("tester", remoteAgentTaskCreateRequest{
		Target: remoteAgentTaskTarget{AgentID: agent.ID},
		Prompt: "Review this patch",
	})
	if err != nil {
		t.Fatalf("dispatch task: %v", err)
	}
	if _, err := service.CompleteTask(task.ID, remoteAgentTaskCompleteRequest{Summary: "done"}); err != nil {
		t.Fatalf("complete task: %v", err)
	}
	if _, err := service.AppendTranscript(task.ID, remoteAgentTranscriptEntry{Role: remoteAgentTranscriptRoleAgent, Text: "late transcript"}); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("expected transcript mutation to be rejected after completion, got %v", err)
	}
	if _, err := service.AddOutputArtifact(task.ID, remoteAgentArtifactCreateRequest{Name: "late.md", Kind: remoteAgentArtifactKindReport}); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("expected artifact mutation to be rejected after completion, got %v", err)
	}
}

func TestRemoteAgentOrchestrationStoreRedactsSensitiveTaskPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "orchestration.jsonl")
	store := newRemoteAgentOrchestrationStore(path)
	service := newRemoteAgentOrchestrationService(store, "", nil, nil)

	agent, err := service.CreateAgent(remoteAgentCreateRequest{Name: "researcher", CurrentSession: &remoteAgentSessionBinding{HarnessRunID: "run-456", SessionID: "spoofed"}})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := service.DispatchTask("tester", remoteAgentTaskCreateRequest{
		Target: remoteAgentTaskTarget{AgentID: agent.ID},
		Prompt: "authorization: Bearer super-secret-token",
		InputArtifacts: []remoteAgentArtifactCreateRequest{{
			Name: "brief token=topsecret.md",
			Kind: remoteAgentArtifactKindFile,
			URI:  "https://user:pass@example.invalid/out?token=topsecret",
		}},
	})
	if err != nil {
		t.Fatalf("dispatch task: %v", err)
	}
	if _, err := service.AppendTranscript(task.ID, remoteAgentTranscriptEntry{Role: remoteAgentTranscriptRoleUser, Text: "password=hunter2"}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}
	if _, err := service.AddOutputArtifact(task.ID, remoteAgentArtifactCreateRequest{Name: "report", Kind: remoteAgentArtifactKindReport, Path: "/tmp/token=abc"}); err != nil {
		t.Fatalf("add output artifact: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read store: %v", err)
	}
	content := string(raw)
	for _, secret := range []string{"super-secret-token", "topsecret", "hunter2", "user:pass"} {
		if strings.Contains(content, secret) {
			t.Fatalf("expected persisted store to redact %q, got %s", secret, content)
		}
	}
	if !strings.Contains(content, "[redacted]") {
		t.Fatalf("expected persisted store to contain redaction markers, got %s", content)
	}
}
