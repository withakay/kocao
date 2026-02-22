package controlplaneapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AuditEvent struct {
	ID           string          `json:"id"`
	At           time.Time       `json:"at"`
	Actor        string          `json:"actor"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resourceType"`
	ResourceID   string          `json:"resourceID"`
	Outcome      string          `json:"outcome"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
}

// AuditStore persists append-only audit events.
// If Path is empty, it operates as an in-memory store (used for tests).
type AuditStore struct {
	mu     sync.Mutex
	Path   string
	mem    []AuditEvent
	maxMem int
}

func newAuditStore(path string) *AuditStore {
	return &AuditStore{Path: path, maxMem: 10_000}
}

func (a *AuditStore) Append(ctx context.Context, actor, action, resourceType, resourceID, outcome string, metadata any) {
	_ = ctx
	meta := json.RawMessage(nil)
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			meta = json.RawMessage(b)
		}
	}

	e := AuditEvent{
		ID:           newID(),
		At:           time.Now().UTC(),
		Actor:        actor,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Outcome:      outcome,
		Metadata:     meta,
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.Path == "" {
		a.mem = append(a.mem, e)
		if a.maxMem > 0 && len(a.mem) > a.maxMem {
			a.mem = a.mem[len(a.mem)-a.maxMem:]
		}
		return
	}

	_ = os.MkdirAll(filepath.Dir(a.Path), 0o755)
	f, err := os.OpenFile(a.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	enc := json.NewEncoder(f)
	_ = enc.Encode(e)
	_ = f.Sync()
	_ = f.Close()
}

func (a *AuditStore) List(ctx context.Context, limit int) ([]AuditEvent, error) {
	_ = ctx
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.Path == "" {
		if len(a.mem) <= limit {
			out := make([]AuditEvent, len(a.mem))
			copy(out, a.mem)
			return out, nil
		}
		out := make([]AuditEvent, limit)
		copy(out, a.mem[len(a.mem)-limit:])
		return out, nil
	}

	f, err := os.Open(a.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	// For early bring-up, a simple scan+decode is sufficient.
	var all []AuditEvent
	s := bufio.NewScanner(f)
	for s.Scan() {
		var e AuditEvent
		if err := json.Unmarshal(s.Bytes(), &e); err != nil {
			continue
		}
		all = append(all, e)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	if len(all) <= limit {
		return all, nil
	}
	return all[len(all)-limit:], nil
}
