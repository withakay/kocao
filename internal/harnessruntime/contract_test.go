package harnessruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type runtimeMatrix struct {
	Go   string `json:"go"`
	Node string `json:"node"`
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
	if mat.Go == "" || mat.Node == "" {
		t.Fatalf("runtime matrix must include go and node versions: %#v", mat)
	}

	dockerBytes, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	dockerfile := string(dockerBytes)

	if !strings.Contains(dockerfile, "FROM golang:"+mat.Go+"-bookworm") {
		t.Fatalf("dockerfile must pin golang base image to %q", mat.Go)
	}
	if !strings.Contains(dockerfile, "FROM node:"+mat.Node+"-bookworm-slim") {
		t.Fatalf("dockerfile must pin node base image to %q", mat.Node)
	}

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
