package tokensync

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// mockPatcher records Patch calls for test assertions.
type mockPatcher struct {
	mu    sync.Mutex
	calls []patchCall
}

type patchCall struct {
	Key  string
	Data []byte
}

func (m *mockPatcher) Patch(_ context.Context, key string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, patchCall{Key: key, Data: append([]byte(nil), data...)})
	return nil
}

func (m *mockPatcher) callCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.calls)
}

func (m *mockPatcher) lastCall() patchCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls[len(m.calls)-1]
}

func TestWatcher_DetectsFileChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")

	// Write initial content.
	if err := os.WriteFile(path, []byte(`{"token":"v1"}`), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockPatcher{}
	w := New(50*time.Millisecond, []FileMapping{
		{Path: path, SecretKey: "auth.json"},
	}, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go func() { _ = w.Run(ctx) }()

	// Wait for at least one poll cycle.
	time.Sleep(150 * time.Millisecond)

	if mock.callCount() < 1 {
		t.Fatal("expected at least 1 Patch call for initial file")
	}

	last := mock.lastCall()
	if last.Key != "auth.json" {
		t.Fatalf("expected key 'auth.json', got %q", last.Key)
	}
	if string(last.Data) != `{"token":"v1"}` {
		t.Fatalf("expected data %q, got %q", `{"token":"v1"}`, string(last.Data))
	}
}

func TestWatcher_DeduplicatesSameContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")

	content := []byte(`{"token":"same"}`)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockPatcher{}
	w := New(50*time.Millisecond, []FileMapping{
		{Path: path, SecretKey: "auth.json"},
	}, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	go func() { _ = w.Run(ctx) }()

	// Wait for several poll cycles.
	time.Sleep(250 * time.Millisecond)

	// Should only have been called once (initial detection), not on every tick.
	if mock.callCount() != 1 {
		t.Fatalf("expected exactly 1 Patch call (dedup), got %d", mock.callCount())
	}
}

func TestWatcher_ToleratesMissingFile(t *testing.T) {
	mock := &mockPatcher{}
	w := New(50*time.Millisecond, []FileMapping{
		{Path: "/nonexistent/path/auth.json", SecretKey: "auth.json"},
	}, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Should not panic or return error.
	err := w.Run(ctx)
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if mock.callCount() != 0 {
		t.Fatalf("expected 0 Patch calls for missing file, got %d", mock.callCount())
	}
}

func TestWatcher_StopsOnContextCancel(t *testing.T) {
	mock := &mockPatcher{}
	w := New(50*time.Millisecond, []FileMapping{
		{Path: "/nonexistent/path/auth.json", SecretKey: "auth.json"},
	}, mock)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- w.Run(ctx)
	}()

	// Cancel after a short delay.
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() returned error on cancel: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after context cancel")
	}
}

func TestWatcher_DetectsContentUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth.json")

	// Write initial content.
	if err := os.WriteFile(path, []byte(`{"token":"v1"}`), 0644); err != nil {
		t.Fatal(err)
	}

	mock := &mockPatcher{}
	w := New(50*time.Millisecond, []FileMapping{
		{Path: path, SecretKey: "auth.json"},
	}, mock)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	go func() { _ = w.Run(ctx) }()

	// Wait for initial detection.
	time.Sleep(100 * time.Millisecond)

	// Update the file content.
	if err := os.WriteFile(path, []byte(`{"token":"v2"}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for the change to be detected.
	time.Sleep(150 * time.Millisecond)

	if mock.callCount() < 2 {
		t.Fatalf("expected at least 2 Patch calls (initial + update), got %d", mock.callCount())
	}

	last := mock.lastCall()
	if string(last.Data) != `{"token":"v2"}` {
		t.Fatalf("expected updated data, got %q", string(last.Data))
	}
}
