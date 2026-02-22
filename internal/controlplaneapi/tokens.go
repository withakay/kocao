package controlplaneapi

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
)

type TokenRecord struct {
	ID        string
	Hash      string
	Scopes    string
	ExpiresAt time.Time
	Claims    map[string]string
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

func (t *TokenStore) Create(ctx context.Context, id, raw string, scopes []string) error {
	return t.CreateWithClaims(ctx, id, raw, scopes, time.Time{}, nil)
}

func (t *TokenStore) CreateWithClaims(_ context.Context, id, raw string, scopes []string, expiresAt time.Time, claims map[string]string) error {
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
	var copied map[string]string
	if len(claims) != 0 {
		copied = map[string]string{}
		for k, v := range claims {
			copied[k] = v
		}
	}
	t.byHash[h] = TokenRecord{ID: id, Hash: h, Scopes: joined, ExpiresAt: expiresAt, Claims: copied}
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
	if !rec.ExpiresAt.IsZero() && time.Now().After(rec.ExpiresAt) {
		t.mu.Lock()
		delete(t.byHash, h)
		t.mu.Unlock()
		return nil, nil
	}
	return &rec, nil
}
