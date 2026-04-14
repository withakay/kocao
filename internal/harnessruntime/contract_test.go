package harnessruntime

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
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

	// Verify agent CLIs and sandbox-agent are present in the tools section.
	for _, tool := range []string{"claude", "opencode", "codex", "pi", "sandbox-agent"} {
		if _, ok := mat.Tools[tool]; !ok {
			t.Fatalf("runtime matrix must include agent CLI tool %q", tool)
		}
	}

	dockerBytes, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatalf("read dockerfile: %v", err)
	}
	dockerfile := string(dockerBytes)

	for _, stage := range []string{"contract-shared", "contract-base", "contract-go", "contract-web", "contract-full", "full-extra-toolchains"} {
		if !strings.Contains(dockerfile, "AS "+stage) {
			t.Fatalf("dockerfile must define %q stage", stage)
		}
	}

	requiredCopies := []string{
		"COPY build/harness/kocao-harness-entrypoint.sh /usr/local/bin/kocao-harness-entrypoint",
		"COPY build/harness/kocao-git-askpass.sh /usr/local/bin/kocao-git-askpass",
		"COPY build/harness/smoke.sh /usr/local/bin/kocao-harness-smoke",
	}
	for _, s := range requiredCopies {
		if !strings.Contains(dockerfile, s) {
			t.Fatalf("dockerfile must include %q", s)
		}
	}

	if !strings.Contains(dockerfile, "FROM contract-base AS harness-profile-base") {
		t.Fatalf("dockerfile must derive harness-profile-base directly from contract-base")
	}

	for _, profile := range []string{"go", "web", "full"} {
		if !strings.Contains(dockerfile, "COPY --from=contract-"+profile+" /etc/kocao /etc/kocao") {
			t.Fatalf("dockerfile must consume contract-%s stage in final image", profile)
		}
	}

	if strings.Contains(dockerfile, "FROM go-toolchain AS full-toolchain") {
		t.Fatalf("dockerfile should use full-extra-toolchains stage instead of full-toolchain")
	}

	for _, token := range []string{"sandbox-agent --version", "pi --version"} {
		if !strings.Contains(dockerfile, token) {
			t.Fatalf("dockerfile must validate %q during build", token)
		}
	}

	if !strings.Contains(dockerfile, "ENTRYPOINT") || !strings.Contains(dockerfile, "kocao-harness-entrypoint") {
		t.Fatalf("dockerfile must set entrypoint to kocao-harness-entrypoint")
	}
}

func TestHarnessProfileMetadataDrivesSandboxSmokeChecks(t *testing.T) {
	root := filepath.Join("..", "..")
	profilesDir := filepath.Join(root, "build", "harness", "profiles")

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		t.Fatalf("read profiles dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".harness-profile.json") {
			continue
		}

		path := filepath.Join(profilesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read profile %s: %v", entry.Name(), err)
		}

		var profile struct {
			RequiredAgents []string `json:"requiredAgents"`
			RequiredTools  []string `json:"requiredTools"`
		}
		if err := json.Unmarshal(data, &profile); err != nil {
			t.Fatalf("unmarshal profile %s: %v", entry.Name(), err)
		}

		if len(profile.RequiredAgents) == 0 {
			t.Fatalf("profile %s must declare requiredAgents", entry.Name())
		}
		if len(profile.RequiredTools) == 0 {
			t.Fatalf("profile %s must declare requiredTools", entry.Name())
		}
		if bytes.Contains(data, []byte("smokeAgentCatalogAgents")) || bytes.Contains(data, []byte("smokeReportAgents")) {
			t.Fatalf("profile %s should derive sandbox smoke checks from required metadata only", entry.Name())
		}

		for _, agent := range profile.RequiredAgents {
			if !slices.Contains(profile.RequiredTools, agent) {
				t.Fatalf("profile %s must keep required agent %q in requiredTools for sandbox-agent verification", entry.Name(), agent)
			}
		}
	}
}
