package controlplaneapi

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type agentSessionStoreRecord struct {
	Type      string             `json:"type"`
	HarnessID string             `json:"harnessRunID"`
	At        time.Time          `json:"at"`
	State     *agentSessionState `json:"state,omitempty"`
	Event     *agentSessionEvent `json:"event,omitempty"`
}

type AgentSessionStore struct {
	mu     sync.Mutex
	path   string
	mem    []agentSessionStoreRecord
	maxMem int
}

func newAgentSessionStore(path string) *AgentSessionStore {
	return &AgentSessionStore{path: path, maxMem: 50_000}
}

func agentSessionStorePath(auditPath string) string {
	if auditPath == "" {
		return ""
	}
	dir := filepath.Dir(auditPath)
	return filepath.Join(dir, "kocao.agent_sessions.jsonl")
}

func (s *AgentSessionStore) append(record agentSessionStoreRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		s.mem = append(s.mem, record)
		if s.maxMem > 0 && len(s.mem) > s.maxMem {
			s.mem = s.mem[len(s.mem)-s.maxMem:]
		}
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_ = json.NewEncoder(f).Encode(record)
	_ = f.Sync()
}

func (s *AgentSessionStore) SaveState(state agentSessionState) {
	s.append(agentSessionStoreRecord{Type: "state", HarnessID: state.HarnessRunID, At: time.Now().UTC(), State: &state})
}

func (s *AgentSessionStore) AppendEvent(runID string, event agentSessionEvent) {
	event.Envelope = sanitizeAgentSessionEnvelope(event.Envelope)
	s.append(agentSessionStoreRecord{Type: "event", HarnessID: runID, At: time.Now().UTC(), Event: &event})
}

func (s *AgentSessionStore) records() ([]agentSessionStoreRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		out := make([]agentSessionStoreRecord, len(s.mem))
		copy(out, s.mem)
		return out, nil
	}
	f, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []agentSessionStoreRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec agentSessionStoreRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AgentSessionStore) LoadState(runIDs ...string) (agentSessionState, bool) {
	records, err := s.records()
	if err != nil {
		return agentSessionState{}, false
	}
	wanted := map[string]struct{}{}
	for _, id := range runIDs {
		if id != "" {
			wanted[id] = struct{}{}
		}
	}
	var latest agentSessionState
	var found bool
	for _, rec := range records {
		if rec.State == nil {
			continue
		}
		if _, ok := wanted[rec.HarnessID]; !ok {
			continue
		}
		latest = *rec.State
		found = true
	}
	return latest, found
}

func sanitizeAgentSessionEnvelope(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return raw
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return raw
	}
	sanitized, err := json.Marshal(sanitizeAgentSessionValue(decoded))
	if err != nil {
		return raw
	}
	return json.RawMessage(sanitized)
}

func sanitizeAgentSessionValue(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if agentSessionKeyLooksSensitive(key) {
				out[key] = "[redacted]"
				continue
			}
			out[key] = sanitizeAgentSessionValue(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = sanitizeAgentSessionValue(value[i])
		}
		return out
	default:
		return v
	}
}

func agentSessionKeyLooksSensitive(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, fragment := range []string{"secret", "token", "password", "authorization", "credential", "apiKey", "api-key"} {
		if strings.Contains(key, strings.ToLower(fragment)) {
			return true
		}
	}
	return false
}

func (s *AgentSessionStore) ListEvents(offset int64, limit int, runIDs ...string) ([]agentSessionEvent, int64, bool) {
	records, err := s.records()
	if err != nil {
		return nil, 0, false
	}
	wanted := map[string]struct{}{}
	for _, id := range runIDs {
		if id != "" {
			wanted[id] = struct{}{}
		}
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	var out []agentSessionEvent
	var next int64
	for _, rec := range records {
		if rec.Event == nil {
			continue
		}
		if _, ok := wanted[rec.HarnessID]; !ok {
			continue
		}
		if rec.Event.Sequence > next {
			next = rec.Event.Sequence
		}
		if rec.Event.Sequence <= offset {
			continue
		}
		if len(out) < limit {
			out = append(out, *rec.Event)
		}
	}
	return out, next, true
}
