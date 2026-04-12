package controlplanecli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentExec_Success(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
			http.Error(w, "bad method", http.StatusBadRequest)
			return
		}
		if r.URL.Path != "/api/v1/harness-runs/run-42/agent-session/prompt" {
			t.Errorf("path = %q", r.URL.Path)
			http.Error(w, "bad path", http.StatusBadRequest)
			return
		}
		var req PromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Prompt != "hello agent" {
			t.Errorf("prompt = %q, want hello agent", req.Prompt)
			http.Error(w, "bad prompt", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{"seq": 1, "timestamp": "2025-06-15T10:30:00Z", "data": map[string]any{"type": "response", "text": "hi there"}},
				{"seq": 2, "timestamp": "2025-06-15T10:30:01Z", "data": map[string]any{"type": "end"}},
			},
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "exec", "run-42", "--prompt", "hello agent"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "seq=1") {
		t.Errorf("expected seq=1 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "seq=2") {
		t.Errorf("expected seq=2 in output, got:\n%s", out)
	}
	if !strings.Contains(out, "hi there") {
		t.Errorf("expected response text in output, got:\n%s", out)
	}
	if !strings.Contains(out, "10:30:00") {
		t.Errorf("expected timestamp in output, got:\n%s", out)
	}
}

func TestAgentExec_PositionalPrompt(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req PromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if req.Prompt != "do something" {
			t.Errorf("prompt = %q, want do something", req.Prompt)
			http.Error(w, "bad prompt", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{"seq": 1, "timestamp": "2025-06-15T10:30:00Z", "data": map[string]any{"type": "ack"}},
			},
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "exec", "run-42", "do", "something"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "seq=1") {
		t.Errorf("expected seq=1 in output, got:\n%s", stdout.String())
	}
}

func TestAgentExec_MissingRunID(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "exec"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code for missing run-id")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage message in stderr, got:\n%s", stderr.String())
	}
}

func TestAgentExec_MissingPrompt(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "exec", "run-42"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("expected non-zero exit code for missing prompt")
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage message in stderr, got:\n%s", stderr.String())
	}
}

func TestAgentExec_JSONOutput(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": []map[string]any{
				{"seq": 1, "timestamp": "2025-06-15T10:30:00Z", "data": map[string]any{"type": "response", "text": "hello"}},
				{"seq": 2, "timestamp": "2025-06-15T10:30:01Z", "data": map[string]any{"type": "end"}},
			},
		})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "exec", "run-42", "--prompt", "test", "--output", "json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}

	// Verify it's valid JSON.
	var resp PromptResponse
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, stdout.String())
	}
	if len(resp.Events) != 2 {
		t.Fatalf("len(events) = %d, want 2", len(resp.Events))
	}
	if resp.Events[0].Seq != 1 {
		t.Fatalf("events[0].Seq = %d, want 1", resp.Events[0].Seq)
	}
	if resp.Events[1].Seq != 2 {
		t.Fatalf("events[1].Seq = %d, want 2", resp.Events[1].Seq)
	}
}

func TestAgentExec_APIError(t *testing.T) {
	t.Setenv(EnvToken, "")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": "agent session not found"})
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", srv.URL, "--token", "test-token", "agent", "exec", "run-missing", "--prompt", "hello"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "agent session not found") {
		t.Errorf("expected error message in stderr, got:\n%s", stderr.String())
	}
}
