package tokensync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// FileMapping maps a filesystem path to a Secret data key.
type FileMapping struct {
	Path      string
	SecretKey string
}

// Watcher polls files and patches a Secret when content changes.
type Watcher struct {
	pollInterval time.Duration
	mappings     []FileMapping
	patcher      Patcher
	lastHashes   map[string]string
}

// New creates a Watcher that checks mappings every interval.
func New(interval time.Duration, mappings []FileMapping, patcher Patcher) *Watcher {
	return &Watcher{
		pollInterval: interval,
		mappings:     mappings,
		patcher:      patcher,
		lastHashes:   make(map[string]string),
	}
}

// Run polls until ctx is cancelled. It returns nil on clean shutdown.
func (w *Watcher) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Do an initial check immediately.
	w.poll(ctx)

	for {
		select {
		case <-ctx.Done():
			slog.Info("watcher stopping")
			return nil
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Watcher) poll(ctx context.Context) {
	for _, m := range w.mappings {
		if err := w.checkFile(ctx, m); err != nil {
			slog.Error("check file failed", "path", m.Path, "error", err)
		}
	}
}

func (w *Watcher) checkFile(ctx context.Context, m FileMapping) error {
	_, err := os.Stat(m.Path)
	if os.IsNotExist(err) {
		slog.Debug("file not found, skipping", "path", m.Path)
		return nil
	}
	if err != nil {
		return fmt.Errorf("stat %s: %w", m.Path, err)
	}

	content, err := os.ReadFile(m.Path)
	if err != nil {
		return fmt.Errorf("read %s: %w", m.Path, err)
	}

	hash := sha256sum(content)
	if w.lastHashes[m.Path] == hash {
		slog.Debug("file unchanged", "path", m.Path)
		return nil
	}

	if err := w.patcher.Patch(ctx, m.SecretKey, content); err != nil {
		return fmt.Errorf("patch %s: %w", m.SecretKey, err)
	}

	w.lastHashes[m.Path] = hash
	return nil
}

func sha256sum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}
