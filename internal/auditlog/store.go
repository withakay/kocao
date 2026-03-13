package auditlog

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var sensitiveKeyFragments = []string{"secret", "token", "password", "authorization", "credential"}

type Event struct {
	ID           string          `json:"id"`
	At           time.Time       `json:"at"`
	Actor        string          `json:"actor"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resourceType"`
	ResourceID   string          `json:"resourceID"`
	Outcome      string          `json:"outcome"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

type idGenerator func() string

type Store struct {
	mu     sync.Mutex
	Path   string
	mem    []Event
	maxMem int
	newID  idGenerator
}

func New(path string, generator idGenerator) *Store {
	if generator == nil {
		generator = func() string { return "" }
	}
	return &Store{Path: path, maxMem: 10_000, newID: generator}
}

func (s *Store) Append(ctx context.Context, actor, action, resourceType, resourceID, outcome string, metadata any) {
	_ = ctx
	meta := json.RawMessage(nil)
	if metadata != nil {
		if b, err := json.Marshal(sanitizeMetadata(metadata)); err == nil {
			meta = json.RawMessage(b)
		}
	}

	e := Event{
		ID:           s.newID(),
		At:           time.Now().UTC(),
		Actor:        actor,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      outcome,
		Metadata:     meta,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Path == "" {
		s.mem = append(s.mem, e)
		if s.maxMem > 0 && len(s.mem) > s.maxMem {
			s.mem = s.mem[len(s.mem)-s.maxMem:]
		}
		return
	}

	_ = os.MkdirAll(filepath.Dir(s.Path), 0o755)
	f, err := os.OpenFile(s.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	enc := json.NewEncoder(f)
	_ = enc.Encode(e)
	_ = f.Sync()
	_ = f.Close()
}

func (s *Store) List(ctx context.Context, limit int) ([]Event, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Path == "" {
		if len(s.mem) <= limit {
			out := make([]Event, len(s.mem))
			copy(out, s.mem)
			return out, nil
		}
		out := make([]Event, limit)
		copy(out, s.mem[len(s.mem)-limit:])
		return out, nil
	}

	f, err := os.Open(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var all []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		all = append(all, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(all) <= limit {
		return all, nil
	}
	return all[len(all)-limit:], nil
}

func AppendSymphony(ctx context.Context, audit *Store, actor, action, resourceID, outcome string, metadata map[string]any) {
	if audit == nil {
		return
	}
	audit.Append(ctx, actor, action, "symphony-project", resourceID, outcome, metadata)
}

func sanitizeMetadata(metadata any) any {
	switch value := metadata.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			if keyLooksSensitive(key) {
				out[key] = "[redacted]"
				continue
			}
			out[key] = sanitizeMetadata(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = sanitizeMetadata(value[i])
		}
		return out
	case []map[string]any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = sanitizeMetadata(value[i])
		}
		return out
	default:
		return metadata
	}
}

func keyLooksSensitive(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, fragment := range sensitiveKeyFragments {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}
