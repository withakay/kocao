package controlplanecli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// sampleEvents returns a deterministic set of events for testing.
func sampleEvents() []AgentSessionEvent {
	return []AgentSessionEvent{
		{Seq: 1, Timestamp: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC), Data: json.RawMessage(`{"type":"start"}`)},
		{Seq: 2, Timestamp: time.Date(2025, 6, 15, 10, 30, 1, 0, time.UTC), Data: json.RawMessage(`{"type":"output","text":"hello"}`)},
		{Seq: 3, Timestamp: time.Date(2025, 6, 15, 10, 30, 2, 0, time.UTC), Data: json.RawMessage(`{"type":"end"}`)},
	}
}

func eventsHandler(t *testing.T) http.Handler {
	t.Helper()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/harness-runs/run-1/agent-session/events" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"events": sampleEvents(),
		})
	})
}

func TestAgentLogs_FetchEvents(t *testing.T) {
	srv := httptest.NewServer(eventsHandler(t))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL,
		"--token", "test-token",
		"agent", "logs", "run-1",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	out := stdout.String()
	// Table header
	if !strings.Contains(out, "TIMESTAMP") {
		t.Errorf("missing table header, got:\n%s", out)
	}
	if !strings.Contains(out, "SEQ") {
		t.Errorf("missing SEQ header, got:\n%s", out)
	}
	// Check all three events appear
	for _, want := range []string{"start", "output", "end"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing event type %q in output:\n%s", want, out)
		}
	}
	// Check sequence numbers
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") || !strings.Contains(out, "3") {
		t.Errorf("missing sequence numbers in output:\n%s", out)
	}
}

func TestAgentLogs_JSONOutput(t *testing.T) {
	srv := httptest.NewServer(eventsHandler(t))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL,
		"--token", "test-token",
		"agent", "logs", "run-1", "--output", "json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 JSONL lines, got %d:\n%s", len(lines), stdout.String())
	}

	for i, line := range lines {
		var event AgentSessionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("line %d: invalid JSON: %v\nline: %s", i, err, line)
		}
		if event.Seq != i+1 {
			t.Errorf("line %d: seq = %d, want %d", i, event.Seq, i+1)
		}
	}
}

func TestAgentLogs_TailN(t *testing.T) {
	srv := httptest.NewServer(eventsHandler(t))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL,
		"--token", "test-token",
		"agent", "logs", "run-1", "--tail", "2",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	out := stdout.String()
	// With --tail 2, we should see events 2 and 3 but not event 1's "start" type
	// (event 1 has type "start", event 2 has "output", event 3 has "end")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// Header + 2 data lines = 3 lines
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 events), got %d:\n%s", len(lines), out)
	}

	// First data line should be seq 2 (output), not seq 1 (start)
	if strings.Contains(lines[1], "\tstart\t") {
		t.Errorf("tail=2 should skip first event, but found 'start' in:\n%s", out)
	}
	if !strings.Contains(out, "output") {
		t.Errorf("tail=2 should include 'output' event:\n%s", out)
	}
	if !strings.Contains(out, "end") {
		t.Errorf("tail=2 should include 'end' event:\n%s", out)
	}
}

func TestAgentLogs_MissingRunID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"--api-url", "http://127.0.0.1:9999",
		"--token", "test-token",
		"agent", "logs",
	}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Errorf("expected usage error, got:\n%s", stderr.String())
	}
}

func TestAgentLogs_StreamFollow(t *testing.T) {
	events := sampleEvents()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/harness-runs/run-1/agent-session/events/stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, e := range events {
			b, _ := json.Marshal(e)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	err := streamAgentLogs(ctx, client, "run-1", "table", &stdout)
	if err != nil {
		t.Fatalf("streamAgentLogs: %v", err)
	}

	out := stdout.String()
	// Verify all events streamed
	for _, want := range []string{"start", "output", "end"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing event type %q in streamed output:\n%s", want, out)
		}
	}
	// Verify sequence numbers
	if !strings.Contains(out, "1") || !strings.Contains(out, "2") || !strings.Contains(out, "3") {
		t.Errorf("missing sequence numbers in streamed output:\n%s", out)
	}
}

func TestAgentLogs_StreamFollowJSON(t *testing.T) {
	events := sampleEvents()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for _, e := range events {
			b, _ := json.Marshal(e)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		}
	}))
	defer srv.Close()

	client := newTestClient(t, srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var stdout bytes.Buffer
	err := streamAgentLogs(ctx, client, "run-1", "json", &stdout)
	if err != nil {
		t.Fatalf("streamAgentLogs json: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 JSONL lines, got %d:\n%s", len(lines), stdout.String())
	}
	for i, line := range lines {
		var event AgentSessionEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", i, err)
		}
		if event.Seq != i+1 {
			t.Errorf("line %d: seq = %d, want %d", i, event.Seq, i+1)
		}
	}
}

func TestAgentLogs_TailNJSON(t *testing.T) {
	srv := httptest.NewServer(eventsHandler(t))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := Main([]string{
		"--api-url", srv.URL,
		"--token", "test-token",
		"agent", "logs", "run-1", "--tail", "1", "--output", "json",
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSONL line, got %d:\n%s", len(lines), stdout.String())
	}
	var event AgentSessionEvent
	if err := json.Unmarshal([]byte(lines[0]), &event); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if event.Seq != 3 {
		t.Errorf("seq = %d, want 3 (last event)", event.Seq)
	}
}

func TestExtractEventType(t *testing.T) {
	tests := []struct {
		name string
		data json.RawMessage
		want string
	}{
		{"with type", json.RawMessage(`{"type":"start"}`), "start"},
		{"no type", json.RawMessage(`{"text":"hello"}`), "-"},
		{"empty", json.RawMessage(`{}`), "-"},
		{"nil", nil, "-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEventType(tt.data)
			if got != tt.want {
				t.Errorf("extractEventType(%s) = %q, want %q", tt.data, got, tt.want)
			}
		})
	}
}

func TestFormatEventTimestamp(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	got := formatEventTimestamp(ts)
	if got != "2025-06-15T10:30:00Z" {
		t.Errorf("formatEventTimestamp = %q", got)
	}
	if formatEventTimestamp(time.Time{}) != "-" {
		t.Error("zero time should return dash")
	}
}
