package controlplanecli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientRemoteAgentTaskLifecycle(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/remote-agent-pools", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgentPool{ID: "pool-1", Name: "reviewers"})
	})
	mux.HandleFunc("/api/v1/remote-agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgent{ID: "agent-1", Name: "reviewer", PoolID: "pool-1", PoolName: "reviewers", Availability: "idle"})
	})
	mux.HandleFunc("/api/v1/remote-agent-tasks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		var req RemoteAgentTaskCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode task request: %v", err)
		}
		if req.Target.AgentName != "reviewer" {
			t.Fatalf("agent target = %q, want reviewer", req.Target.AgentName)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgentTask{ID: "task-1", AgentID: "agent-1", AgentName: "reviewer", State: "assigned"})
	})
	mux.HandleFunc("/api/v1/remote-agent-tasks/task-1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgentTask{ID: "task-1", AgentID: "agent-1", AgentName: "reviewer", State: "completed", Result: &RemoteAgentTaskResult{Summary: "done", Outcome: "completed"}})
	})
	mux.HandleFunc("/api/v1/remote-agent-tasks/task-1/transcript", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgentTaskTranscript{TaskID: "task-1", Transcript: []RemoteAgentTranscriptEntry{{Sequence: 1, Role: "agent", Text: "done"}}})
	})
	mux.HandleFunc("/api/v1/remote-agent-tasks/task-1/artifacts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgentTaskArtifacts{TaskID: "task-1", OutputArtifacts: []RemoteAgentArtifactRef{{ID: "artifact-1", Name: "review.md", Kind: "report"}}})
	})
	mux.HandleFunc("/api/v1/remote-agent-tasks/task-1/retry", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RemoteAgentTask{ID: "task-1", AgentID: "agent-1", AgentName: "reviewer", State: "assigned", Attempt: 2, RetryCount: 1})
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, err := NewClient(Config{BaseURL: srv.URL, Token: "test-token"})
	if err != nil {
		t.Fatalf("new client: %v", err)
	}
	ctx := context.Background()

	pool, err := client.CreateRemoteAgentPool(ctx, RemoteAgentPoolCreateRequest{Name: "reviewers"})
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	if pool.ID != "pool-1" {
		t.Fatalf("pool id = %q, want pool-1", pool.ID)
	}

	agent, err := client.CreateRemoteAgent(ctx, RemoteAgentCreateRequest{Name: "reviewer", PoolID: pool.ID})
	if err != nil {
		t.Fatalf("create agent: %v", err)
	}
	task, err := client.CreateRemoteAgentTask(ctx, RemoteAgentTaskCreateRequest{Target: RemoteAgentTaskTarget{AgentName: agent.Name}, Prompt: "Review the patch"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if task.ID != "task-1" {
		t.Fatalf("task id = %q, want task-1", task.ID)
	}

	task, err = client.GetRemoteAgentTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.Result == nil || task.Result.Summary != "done" {
		t.Fatalf("unexpected task result: %+v", task.Result)
	}

	transcript, err := client.GetRemoteAgentTaskTranscript(ctx, task.ID)
	if err != nil {
		t.Fatalf("get transcript: %v", err)
	}
	if len(transcript.Transcript) != 1 || transcript.Transcript[0].Text != "done" {
		t.Fatalf("unexpected transcript payload: %+v", transcript)
	}

	artifacts, err := client.GetRemoteAgentTaskArtifacts(ctx, task.ID)
	if err != nil {
		t.Fatalf("get artifacts: %v", err)
	}
	if len(artifacts.OutputArtifacts) != 1 || artifacts.OutputArtifacts[0].Name != "review.md" {
		t.Fatalf("unexpected artifact payload: %+v", artifacts)
	}

	task, err = client.RetryRemoteAgentTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("retry task: %v", err)
	}
	if task.Attempt != 2 || task.RetryCount != 1 || task.State != "assigned" {
		t.Fatalf("unexpected retried task: %+v", task)
	}
}
