package controlplaneapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
)

type TokenRecord struct {
	ID     string
	Hash   string
	Scopes string
}

type TokenStore struct {
	mu     sync.RWMutex
	byHash map[string]TokenRecord
}

func newTokenStore() *TokenStore {
	return &TokenStore{byHash: map[string]TokenRecord{}}
}

func tokenHash(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (t *TokenStore) EnsureBootstrapToken(_ context.Context, raw string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return t.Create(context.Background(), "bootstrap", raw, []string{"*"})
}

func (t *TokenStore) Create(_ context.Context, id, raw string, scopes []string) error {
	if strings.TrimSpace(id) == "" {
		return errors.New("token id required")
	}
	if strings.TrimSpace(raw) == "" {
		return errors.New("raw token required")
	}
	if len(scopes) == 0 {
		return errors.New("scopes required")
	}
	h := tokenHash(raw)
	joined := strings.Join(scopes, ",")

	t.mu.Lock()
	defer t.mu.Unlock()
	if _, exists := t.byHash[h]; exists {
		return nil
	}
	t.byHash[h] = TokenRecord{ID: id, Hash: h, Scopes: joined}
	return nil
}

func (t *TokenStore) Lookup(_ context.Context, raw string) (*TokenRecord, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	h := tokenHash(raw)
	t.mu.RLock()
	rec, ok := t.byHash[h]
	t.mu.RUnlock()
	if !ok {
		return nil, nil
	}
	return &rec, nil
}
