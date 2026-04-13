package controlplaneapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestAgentSessionPersistenceAndResumeHistory_API_StoreBacked(t *testing.T) {
	api, cleanup := newTestAPI(t)
	defer cleanup()
	store := newAgentSessionStore("")
	transport := newFakeAgentSessionTransport()
	api.AgentSessions = newAgentSessionService(transport, store)

	if err := api.Tokens.Create(context.Background(), "t-full", "full", []string{"harness-run:write", "harness-run:read"}); err != nil {
		t.Fatalf("create token: %v", err)
	}

	original := &operatorv1alpha1.HarnessRun{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-original", Namespace: api.Namespace},
		Spec: operatorv1alpha1.HarnessRunSpec{
			RepoURL:    "https://example.com/repo",
			Image:      "kocao/harness-runtime:dev",
			WorkingDir: "/workspace/repo",
			AgentSession: &operatorv1alpha1.AgentSessionSpec{
				Runtime: operatorv1alpha1.AgentRuntimeSandboxAgent,
				Agent:   operatorv1alpha1.AgentKindClaude,
			},
		},
	}
	if err := api.K8s.Create(context.Background(), original); err != nil {
		t.Fatalf("create original run: %v", err)
	}
	var stored operatorv1alpha1.HarnessRun
	if err := api.K8s.Get(context.Background(), client.ObjectKeyFromObject(original), &stored); err != nil {
		t.Fatalf("get original run: %v", err)
	}
	stored.Status.PodName = "pod-original"
	stored.Status.Phase = operatorv1alpha1.HarnessRunPhaseRunning
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update original run status: %v", err)
	}

	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	resp, b := doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+original.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+original.Name+"/agent-session/prompt", "full", map[string]any{"prompt": "hello sandbox"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prompt status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	for i := 0; i < 20; i++ {
		resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+original.Name+"/agent-session/events?offset=0&limit=10", "full", nil)
		var payload struct {
			Events []agentSessionEvent `json:"events"`
		}
		_ = json.Unmarshal(b, &payload)
		if len(payload.Events) != 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	_, _ = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+original.Name+"/agent-session/stop", "full", nil)

	api.AgentSessions = newAgentSessionService(newFakeAgentSessionTransport(), store)

	resumed := &operatorv1alpha1.HarnessRun{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-resumed",
			Namespace: api.Namespace,
			Labels:    map[string]string{"kocao.withakay.github.com/resumed-from": original.Name},
		},
		Spec: original.Spec,
	}
	if err := api.K8s.Create(context.Background(), resumed); err != nil {
		t.Fatalf("create resumed run: %v", err)
	}
	if err := api.K8s.Get(context.Background(), client.ObjectKeyFromObject(resumed), &stored); err != nil {
		t.Fatalf("get resumed run: %v", err)
	}
	stored.Status.PodName = "pod-resumed"
	stored.Status.Phase = operatorv1alpha1.HarnessRunPhaseRunning
	if err := api.K8s.Status().Update(context.Background(), &stored); err != nil {
		t.Fatalf("update resumed run status: %v", err)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+resumed.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get resumed agent session status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	var resumedDTO agentSessionDTO
	if err := json.Unmarshal(b, &resumedDTO); err != nil {
		t.Fatalf("unmarshal resumed state: %v", err)
	}
	if resumedDTO.RunID != resumed.Name || resumedDTO.SessionID != "sas-123" {
		t.Fatalf("unexpected resumed state: %+v", resumedDTO)
	}

	resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+resumed.Name+"/agent-session/events?offset=0&limit=10", "full", nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get resumed events status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}
	var eventsPayload struct {
		Events     []agentSessionEvent `json:"events"`
		NextOffset int64               `json:"nextOffset"`
	}
	if err := json.Unmarshal(b, &eventsPayload); err != nil {
		t.Fatalf("unmarshal resumed events: %v", err)
	}
	if len(eventsPayload.Events) == 0 {
		t.Fatal("expected resumed run to see persisted events")
	}
	priorNext := eventsPayload.NextOffset

	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+resumed.Name+"/agent-session", "full", nil)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create resumed session status = %d, want 201 (body=%s)", resp.StatusCode, string(b))
	}
	resp, b = doJSON(t, srv.Client(), http.MethodPost, srv.URL+"/api/v1/harness-runs/"+resumed.Name+"/agent-session/prompt", "full", map[string]any{"prompt": "continue after resume"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("prompt resumed session status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
	}

	for i := 0; i < 20; i++ {
		resp, b = doJSON(t, srv.Client(), http.MethodGet, srv.URL+"/api/v1/harness-runs/"+resumed.Name+"/agent-session/events?offset="+strconv.FormatInt(priorNext, 10)+"&limit=10", "full", nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("get resumed incremental events status = %d, want 200 (body=%s)", resp.StatusCode, string(b))
		}
		if err := json.Unmarshal(b, &eventsPayload); err != nil {
			t.Fatalf("unmarshal resumed incremental events: %v", err)
		}
		if len(eventsPayload.Events) != 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(eventsPayload.Events) == 0 {
		t.Fatal("expected resumed run to emit events after previous offset")
	}
	for _, event := range eventsPayload.Events {
		if event.Sequence <= priorNext {
			t.Fatalf("expected resumed event sequence > %d, got %d", priorNext, event.Sequence)
		}
	}
}

func TestAgentSessionStoreSanitizesSecretShapedKeys(t *testing.T) {
	store := newAgentSessionStore("")
	store.AppendEvent("run-1", agentSessionEvent{
		Sequence: 1,
		At:       time.Now().UTC(),
		Envelope: json.RawMessage(`{"params":{"credential":{"token":"secret-value"},"content":{"text":"hello"}}}`),
	})
	events, _, ok := store.ListEvents(0, 10, "run-1")
	if !ok || len(events) != 1 {
		t.Fatalf("expected one stored event, got ok=%v len=%d", ok, len(events))
	}
	body := string(events[0].Envelope)
	if body == "" || body == `{"params":{"credential":{"token":"secret-value"},"content":{"text":"hello"}}}` {
		t.Fatalf("expected sanitized envelope, got %s", body)
	}
	if strings.Contains(body, "secret-value") {
		t.Fatalf("expected sensitive values to be redacted, got %s", body)
	}
}

func TestAgentSessionStoreListEventsReportsLatestOffsetBeyondPageLimit(t *testing.T) {
	store := newAgentSessionStore("")
	for i := int64(1); i <= 3; i++ {
		store.AppendEvent("run-1", agentSessionEvent{
			Sequence: i,
			At:       time.Now().UTC(),
			Envelope: json.RawMessage(`{"sequence":` + strconv.FormatInt(i, 10) + `}`),
		})
	}
	events, next, ok := store.ListEvents(0, 1, "run-1")
	if !ok {
		t.Fatal("expected list events to succeed")
	}
	if len(events) != 1 {
		t.Fatalf("expected one paged event, got %d", len(events))
	}
	if next != 3 {
		t.Fatalf("expected next offset to report latest sequence 3, got %d", next)
	}
}

func TestAgentSessionBridgeSanitizesEventsBeforeBroadcast(t *testing.T) {
	bridge := &agentSessionBridge{subscribers: map[chan agentSessionEvent]struct{}{}}
	ch, unsubscribe := bridge.subscribe()
	defer unsubscribe()

	bridge.appendEvent(json.RawMessage(`{"params":{"authorization":"Bearer secret-token"}}`))

	select {
	case event := <-ch:
		body := string(event.Envelope)
		if strings.Contains(body, "secret-token") {
			t.Fatalf("expected broadcast event to be sanitized, got %s", body)
		}
		if !strings.Contains(body, "[redacted]") {
			t.Fatalf("expected broadcast event to include redaction marker, got %s", body)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for broadcast event")
	}
}
