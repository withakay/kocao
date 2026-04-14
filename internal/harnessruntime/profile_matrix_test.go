package harnessruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

type profileMatrix struct {
	Version                 int                       `json:"version"`
	Dockerfile              string                    `json:"dockerfile"`
	PreferredMinimalProfile string                    `json:"preferredMinimalProfile"`
	CompatibilityProfile    string                    `json:"compatibilityProfile"`
	BuildOrder              []string                  `json:"buildOrder"`
	SharedLayers            map[string]sharedLayer    `json:"sharedLayers"`
	Compatibility           profileCompatibility      `json:"compatibility"`
	Profiles                map[string]plannedProfile `json:"profiles"`
}

type sharedLayer struct {
	Purpose       string   `json:"purpose"`
	RequiredFiles []string `json:"requiredFiles"`
	RequiredTools []string `json:"requiredTools"`
	DevRuntimes   []string `json:"devRuntimes"`
}

type profileCompatibility struct {
	WorkspaceDir                 string   `json:"workspaceDir"`
	RunAsUser                    int      `json:"runAsUser"`
	RequiredAgents               []string `json:"requiredAgents"`
	RequiredTools                []string `json:"requiredTools"`
	RequiredEntrypoint           string   `json:"requiredEntrypoint"`
	RequiredHealthEndpoint       string   `json:"requiredHealthEndpoint"`
	RequiredAgentCatalogEndpoint string   `json:"requiredAgentCatalogEndpoint"`
}

type plannedProfile struct {
	ImageSuffix string   `json:"imageSuffix"`
	BuildTarget string   `json:"buildTarget"`
	Layers      []string `json:"layers"`
	DevRuntimes []string `json:"devRuntimes"`
	SmokeChecks []string `json:"smokeChecks"`
	IntendedFor []string `json:"intendedFor"`
}

func TestHarnessProfileMatrixDefinesConcreteBuildPlan(t *testing.T) {
	root := filepath.Join("..", "..")
	matrixPath := filepath.Join(root, "build", "harness", "profile-matrix.json")

	data, err := os.ReadFile(matrixPath)
	if err != nil {
		t.Fatalf("read profile matrix: %v", err)
	}

	var matrix profileMatrix
	if err := json.Unmarshal(data, &matrix); err != nil {
		t.Fatalf("unmarshal profile matrix: %v", err)
	}

	if matrix.Version != 1 {
		t.Fatalf("expected version 1, got %d", matrix.Version)
	}
	if matrix.Dockerfile != "build/Dockerfile.harness" {
		t.Fatalf("unexpected dockerfile path %q", matrix.Dockerfile)
	}
	if matrix.PreferredMinimalProfile != "base" {
		t.Fatalf("preferred minimal profile must be base, got %q", matrix.PreferredMinimalProfile)
	}
	if matrix.CompatibilityProfile != "full" {
		t.Fatalf("compatibility profile must be full, got %q", matrix.CompatibilityProfile)
	}

	expectedOrder := []string{"base", "go", "web", "full"}
	if !slices.Equal(matrix.BuildOrder, expectedOrder) {
		t.Fatalf("build order mismatch: got %v want %v", matrix.BuildOrder, expectedOrder)
	}

	for _, layer := range []string{"os-common", "agent-runtime", "contract", "native-build", "go-toolchain", "web-toolchain", "full-extra-toolchains"} {
		if _, ok := matrix.SharedLayers[layer]; !ok {
			t.Fatalf("shared layer %q missing from profile matrix", layer)
		}
	}

	profiles := map[string]struct {
		target string
	}{
		"base": {target: "harness-profile-base"},
		"go":   {target: "harness-profile-go"},
		"web":  {target: "harness-profile-web"},
		"full": {target: "harness-profile-full"},
	}

	seenTargets := map[string]string{}
	seenSuffixes := map[string]string{}
	for name, want := range profiles {
		profile, ok := matrix.Profiles[name]
		if !ok {
			t.Fatalf("profile %q missing", name)
		}
		if profile.BuildTarget != want.target {
			t.Fatalf("profile %q build target mismatch: got %q want %q", name, profile.BuildTarget, want.target)
		}
		if len(profile.Layers) == 0 {
			t.Fatalf("profile %q must declare at least one layer", name)
		}
		if len(profile.SmokeChecks) == 0 {
			t.Fatalf("profile %q must declare smoke checks", name)
		}
		if len(profile.IntendedFor) == 0 {
			t.Fatalf("profile %q must declare intended use cases", name)
		}
		if other, ok := seenTargets[profile.BuildTarget]; ok {
			t.Fatalf("build target %q reused by %q and %q", profile.BuildTarget, other, name)
		}
		seenTargets[profile.BuildTarget] = name
		if other, ok := seenSuffixes[profile.ImageSuffix]; ok {
			t.Fatalf("image suffix %q reused by %q and %q", profile.ImageSuffix, other, name)
		}
		seenSuffixes[profile.ImageSuffix] = name
	}

	if !slices.Equal(matrix.Profiles["base"].SmokeChecks, []string{"contract"}) {
		t.Fatalf("base profile must only run contract smoke checks")
	}
	if !slices.Equal(matrix.Profiles["base"].Layers, []string{"os-common", "agent-runtime", "contract"}) {
		t.Fatalf("base profile layers mismatch: %v", matrix.Profiles["base"].Layers)
	}
	if !slices.Equal(matrix.Profiles["go"].Layers, []string{"os-common", "agent-runtime", "contract", "native-build", "go-toolchain"}) {
		t.Fatalf("go profile layers mismatch: %v", matrix.Profiles["go"].Layers)
	}
	if !slices.Equal(matrix.Profiles["web"].Layers, []string{"os-common", "agent-runtime", "contract", "web-toolchain"}) {
		t.Fatalf("web profile layers mismatch: %v", matrix.Profiles["web"].Layers)
	}
	if !slices.Equal(matrix.Profiles["full"].Layers, []string{"os-common", "agent-runtime", "contract", "native-build", "go-toolchain", "web-toolchain", "full-extra-toolchains"}) {
		t.Fatalf("full profile layers mismatch: %v", matrix.Profiles["full"].Layers)
	}
	if !slices.Contains(matrix.Profiles["full"].SmokeChecks, "full") {
		t.Fatalf("full profile must include full smoke coverage")
	}

	if matrix.Compatibility.WorkspaceDir != "/workspace" {
		t.Fatalf("workspace dir mismatch: %q", matrix.Compatibility.WorkspaceDir)
	}
	if matrix.Compatibility.RunAsUser != 10001 {
		t.Fatalf("run-as user mismatch: got %d", matrix.Compatibility.RunAsUser)
	}
	if matrix.Compatibility.RequiredEntrypoint != "/usr/local/bin/kocao-harness-entrypoint" {
		t.Fatalf("required entrypoint mismatch: %q", matrix.Compatibility.RequiredEntrypoint)
	}

	for _, tool := range []string{"sandbox-agent", "claude", "codex", "opencode", "pi"} {
		if !slices.Contains(matrix.Compatibility.RequiredTools, tool) {
			t.Fatalf("compatibility tools must include %q", tool)
		}
	}
	for _, agent := range []string{"claude", "codex", "opencode", "pi"} {
		if !slices.Contains(matrix.Compatibility.RequiredAgents, agent) {
			t.Fatalf("compatibility agents must include %q", agent)
		}
	}

	agentRuntime := matrix.SharedLayers["agent-runtime"]
	for _, tool := range matrix.Compatibility.RequiredTools {
		if !slices.Contains(agentRuntime.RequiredTools, tool) {
			t.Fatalf("agent-runtime layer must provide required tool %q", tool)
		}
	}

	contract := matrix.SharedLayers["contract"]
	for _, requiredFile := range []string{
		"/etc/kocao/runtime-matrix.json",
		"/etc/kocao/harness-profile.json",
		"/usr/local/bin/kocao-harness-entrypoint",
		"/usr/local/bin/kocao-git-askpass",
		"/usr/local/bin/kocao-harness-smoke",
	} {
		if !slices.Contains(contract.RequiredFiles, requiredFile) {
			t.Fatalf("contract layer must include %q", requiredFile)
		}
	}

	if !slices.Equal(matrix.Profiles["go"].DevRuntimes, []string{"go"}) {
		t.Fatalf("go profile runtimes mismatch: %v", matrix.Profiles["go"].DevRuntimes)
	}
	if !slices.Equal(matrix.Profiles["web"].DevRuntimes, []string{"node", "bun"}) {
		t.Fatalf("web profile runtimes mismatch: %v", matrix.Profiles["web"].DevRuntimes)
	}
	for _, runtimeName := range []string{"go", "node", "bun", "python", "rust", "dotnet", "zig", "uv"} {
		if !slices.Contains(matrix.Profiles["full"].DevRuntimes, runtimeName) {
			t.Fatalf("full profile must include runtime %q", runtimeName)
		}
	}
}
