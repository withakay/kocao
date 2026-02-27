package harnessruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runtimeEntry struct {
	Versions []string `json:"versions"`
	Check    string   `json:"check"`
}

type runtimeMatrix struct {
	Runtimes map[string]runtimeEntry `json:"runtimes"`
	Tools    map[string]string       `json:"tools"`
}

func TestHarnessDockerfileIsPinnedAndCopiesContractArtifacts(t *testing.T) {
	root := filepath.Join("..", "..")
	matPath := filepath.Join(root, "build", "harness", "runtime-matrix.json")
	dockerfilePath := filepath.Join(root, "build", "Dockerfile.harness")

	matBytes, err := os.ReadFile(matPath)
	if err != nil {
		t.Fatalf("read runtime matrix: %v", err)
	}
	var mat runtimeMatrix
	if err := json.Unmarshal(matBytes, &mat); err != nil {
		t.Fatalf("unmarshal runtime matrix: %v", err)
	}

	// Verify required runtimes are present with at least one version.
	for _, name := range []string{"go", "node", "python", "rust"} {
		entry, ok := mat.Runtimes[name]
		if !ok || len(entry.Versions) == 0 {
			t.Fatalf("runtime matrix must include %s with at least one version", name)
		}
	}

	// Verify agent CLIs are present in the tools section.
	for _, tool := range []string{"claude", "opencode", "codex"} {
		if _, ok := mat.Tools[tool]; !ok {
			t.Fatalf("runtime matrix must include agent CLI tool %q", tool)
		}
	}

	dockerBytes, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	dockerfile := string(dockerBytes)

	requiredCopies := []string{
		"COPY build/harness/runtime-matrix.json /etc/kocao/runtime-matrix.json",
		"COPY build/harness/kocao-harness-entrypoint.sh /usr/local/bin/kocao-harness-entrypoint",
		"COPY build/harness/kocao-git-askpass.sh /usr/local/bin/kocao-git-askpass",
		"COPY build/harness/smoke.sh /usr/local/bin/kocao-harness-smoke",
	}
	for _, s := range requiredCopies {
		if !strings.Contains(dockerfile, s) {
			t.Fatalf("dockerfile must include %q", s)
		}
	}

	if !strings.Contains(dockerfile, "ENTRYPOINT") || !strings.Contains(dockerfile, "kocao-harness-entrypoint") {
		t.Fatalf("dockerfile must set entrypoint to kocao-harness-entrypoint")
	}
}
