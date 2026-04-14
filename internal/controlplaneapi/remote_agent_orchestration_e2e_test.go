package controlplaneapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/withakay/kocao/internal/controlplanecli"
)

func TestRemoteAgentOrchestrationE2E_MultiAgentWorkflowViaCLI(t *testing.T) {
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
		"name":        "workflow",
		"displayName": "Workflow Agents",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create pool status = %d, want 201 (body=%s)", resp.StatusCode, string(body))
	}
	var pool remoteAgentPool
	if err := json.Unmarshal(body, &pool); err != nil {
		t.Fatalf("unmarshal pool: %v", err)
	}

	for _, name := range []string{"researcher", "implementer", "reviewer"} {
		resp, body = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/remote-agents", "orchestration", map[string]any{
			"name":        name,
			"displayName": name,
			"poolId":      pool.ID,
		})
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create agent %s status = %d, want 201 (body=%s)", name, resp.StatusCode, string(body))
		}
	}

	type workflowStep struct {
		agent         string
		prompt        string
		summary       string
		outcome       string
		artifactName  string
		artifactKind  remoteAgentArtifactKind
		artifactPath  string
		assistantText string
	}

	steps := []workflowStep{
		{
			agent:         "researcher",
			prompt:        "Research the regression and summarize the likely root cause.",
			summary:       "Research completed",
			outcome:       "brief-ready",
			artifactName:  "research-notes.md",
			artifactKind:  remoteAgentArtifactKindReport,
			artifactPath:  "/workspace/research-notes.md",
			assistantText: "Likely root cause narrowed to orchestration state handling during retries.",
		},
		{
			agent:         "implementer",
			prompt:        "Implement the fix described in research-notes.md.",
			summary:       "Implementation completed",
			outcome:       "patch-ready",
			artifactName:  "fix.patch",
			artifactKind:  remoteAgentArtifactKindPatch,
			artifactPath:  "/workspace/fix.patch",
			assistantText: "Patched the retry state transition and added validation coverage.",
		},
		{
			agent:         "reviewer",
			prompt:        "Review fix.patch and confirm whether it is ready to merge.",
			summary:       "Review completed",
			outcome:       "approved-with-comments",
			artifactName:  "review.md",
			artifactKind:  remoteAgentArtifactKindReport,
			artifactPath:  "/workspace/review.md",
			assistantText: "Patch looks correct; add one assertion for transcript retention before merge.",
		},
	}

	tasksByAgent := make(map[string]remoteAgentTask, len(steps))
	for _, step := range steps {
		stdout, stderr, code := runRemoteAgentWorkflowCLI(t, srv.URL,
			"remote-agents", "tasks", "dispatch",
			"--agent", step.agent,
			"--pool", "workflow",
			"--prompt", step.prompt,
			"--output", "json",
		)
		if code != 0 {
			t.Fatalf("dispatch %s exit code = %d stderr=%s", step.agent, code, stderr)
		}
		var task remoteAgentTask
		if err := json.Unmarshal(stdout, &task); err != nil {
			t.Fatalf("unmarshal dispatched task for %s: %v (stdout=%s)", step.agent, err, string(stdout))
		}
		if task.AgentName != step.agent || task.State != remoteAgentTaskStateAssigned {
			t.Fatalf("unexpected dispatched task for %s: %+v", step.agent, task)
		}

		if _, err := api.RemoteAgentOrchestration.StartTask(task.ID); err != nil {
			t.Fatalf("start task %s: %v", task.ID, err)
		}
		if _, err := api.RemoteAgentOrchestration.AppendTranscript(task.ID, remoteAgentTranscriptEntry{Role: remoteAgentTranscriptRoleUser, Kind: "prompt", Text: step.prompt}); err != nil {
			t.Fatalf("append user transcript for %s: %v", step.agent, err)
		}
		if _, err := api.RemoteAgentOrchestration.AppendTranscript(task.ID, remoteAgentTranscriptEntry{Role: remoteAgentTranscriptRoleAgent, Kind: "message", Text: step.assistantText}); err != nil {
			t.Fatalf("append assistant transcript for %s: %v", step.agent, err)
		}
		if _, err := api.RemoteAgentOrchestration.AddOutputArtifact(task.ID, remoteAgentArtifactCreateRequest{
			Name:      step.artifactName,
			Kind:      step.artifactKind,
			Path:      step.artifactPath,
			MediaType: "text/markdown",
		}); err != nil {
			t.Fatalf("add output artifact for %s: %v", step.agent, err)
		}
		if _, err := api.RemoteAgentOrchestration.CompleteTask(task.ID, remoteAgentTaskCompleteRequest{
			Summary: step.summary,
			Outcome: step.outcome,
		}); err != nil {
			t.Fatalf("complete task for %s: %v", step.agent, err)
		}

		tasksByAgent[step.agent] = task
	}

	stdout, stderr, code := runRemoteAgentWorkflowCLI(t, srv.URL,
		"remote-agents", "tasks", "list", "--output", "json",
	)
	if code != 0 {
		t.Fatalf("list tasks exit code = %d stderr=%s", code, stderr)
	}
	var tasks []remoteAgentTask
	if err := json.Unmarshal(stdout, &tasks); err != nil {
		t.Fatalf("unmarshal task list: %v (stdout=%s)", err, string(stdout))
	}
	if len(tasks) != len(steps) {
		t.Fatalf("task list len = %d, want %d", len(tasks), len(steps))
	}
	for _, step := range steps {
		task := mustFindTaskByID(t, tasks, tasksByAgent[step.agent].ID)
		if task.AgentName != step.agent || task.State != remoteAgentTaskStateCompleted {
			t.Fatalf("unexpected listed task for %s: %+v", step.agent, task)
		}
	}

	for _, step := range steps {
		taskID := tasksByAgent[step.agent].ID

		stdout, stderr, code = runRemoteAgentWorkflowCLI(t, srv.URL,
			"remote-agents", "tasks", "get", taskID, "--output", "json",
		)
		if code != 0 {
			t.Fatalf("get task %s exit code = %d stderr=%s", taskID, code, stderr)
		}
		var task remoteAgentTask
		if err := json.Unmarshal(stdout, &task); err != nil {
			t.Fatalf("unmarshal task %s: %v (stdout=%s)", taskID, err, string(stdout))
		}
		if task.Result == nil || task.Result.Summary != step.summary || task.Result.OutputArtifactCount != 1 || task.Result.TranscriptEntries != 2 {
			t.Fatalf("unexpected task result for %s: %+v", step.agent, task.Result)
		}

		stdout, stderr, code = runRemoteAgentWorkflowCLI(t, srv.URL,
			"remote-agents", "tasks", "transcript", taskID, "--output", "json",
		)
		if code != 0 {
			t.Fatalf("transcript task %s exit code = %d stderr=%s", taskID, code, stderr)
		}
		var transcript struct {
			TaskID     string                       `json:"taskId"`
			Transcript []remoteAgentTranscriptEntry `json:"transcript"`
		}
		if err := json.Unmarshal(stdout, &transcript); err != nil {
			t.Fatalf("unmarshal transcript %s: %v (stdout=%s)", taskID, err, string(stdout))
		}
		if len(transcript.Transcript) != 2 || transcript.Transcript[0].Text != step.prompt || transcript.Transcript[1].Text != step.assistantText {
			t.Fatalf("unexpected transcript for %s: %+v", step.agent, transcript)
		}

		stdout, stderr, code = runRemoteAgentWorkflowCLI(t, srv.URL,
			"remote-agents", "tasks", "artifacts", taskID, "--output", "json",
		)
		if code != 0 {
			t.Fatalf("artifacts task %s exit code = %d stderr=%s", taskID, code, stderr)
		}
		var artifacts struct {
			TaskID          string                   `json:"taskId"`
			InputArtifacts  []remoteAgentArtifactRef `json:"inputArtifacts"`
			OutputArtifacts []remoteAgentArtifactRef `json:"outputArtifacts"`
		}
		if err := json.Unmarshal(stdout, &artifacts); err != nil {
			t.Fatalf("unmarshal artifacts %s: %v (stdout=%s)", taskID, err, string(stdout))
		}
		if len(artifacts.InputArtifacts) != 0 || len(artifacts.OutputArtifacts) != 1 || artifacts.OutputArtifacts[0].Name != step.artifactName {
			t.Fatalf("unexpected artifacts for %s: %+v", step.agent, artifacts)
		}
	}

	stdout, stderr, code = runRemoteAgentWorkflowCLI(t, srv.URL,
		"remote-agents", "agents", "get", "reviewer", "--pool", "workflow", "--output", "json",
	)
	if code != 0 {
		t.Fatalf("get reviewer exit code = %d stderr=%s", code, stderr)
	}
	var reviewer remoteAgent
	if err := json.Unmarshal(stdout, &reviewer); err != nil {
		t.Fatalf("unmarshal reviewer: %v (stdout=%s)", err, string(stdout))
	}
	if reviewer.Availability != remoteAgentAvailabilityIdle || reviewer.CurrentTaskID != "" {
		t.Fatalf("expected reviewer to be idle after workflow completion: %+v", reviewer)
	}
}

func runRemoteAgentWorkflowCLI(t *testing.T, apiURL string, args ...string) ([]byte, string, int) {
	t.Helper()

	argv := append([]string{"--api-url", apiURL, "--token", "orchestration"}, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := controlplanecli.Main(argv, &stdout, &stderr)
	return stdout.Bytes(), stderr.String(), code
}

func mustFindTaskByID(t *testing.T, tasks []remoteAgentTask, taskID string) remoteAgentTask {
	t.Helper()

	for _, task := range tasks {
		if task.ID == taskID {
			return task
		}
	}
	t.Fatalf("task %s not found in %+v", taskID, tasks)
	return remoteAgentTask{}
}
