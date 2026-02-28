<!-- ITO:START -->
# Tasks for: 002-05_install-agent-harness-clis

## Execution Notes

- **Tracking**: Use `ito tasks` CLI for status updates
- **Status legend**: `[ ] pending` · `[>] in-progress` · `[x] complete` · `[-] shelved`

```bash
ito tasks status 002-05_install-agent-harness-clis
ito tasks next 002-05_install-agent-harness-clis
ito tasks start 002-05_install-agent-harness-clis 1.1
ito tasks complete 002-05_install-agent-harness-clis 1.1
```

______________________________________________________________________

## Wave 1: Install agent CLIs into harness image

- **Depends On**: None

### Task 1.1: Install Claude Code, OpenCode, and Codex CLI in Dockerfile

- **Files**: `build/Dockerfile.harness`, `build/harness/runtime-matrix.json`
- **Dependencies**: None
- **Action**: Add Dockerfile layers to install: (1) Claude Code via `curl -fsSL https://claude.ai/install.sh | bash`, (2) OpenCode via `npm install -g opencode-ai@<version>`, (3) Codex CLI via `npm install -g @openai/codex@<version>`. Create necessary directories for OAuth file mounts (`~/.local/share/opencode/`, `~/.codex/`) with correct ownership. Update `runtime-matrix.json` with entries for all three CLIs. Ensure smoke test validates them.
- **Verify**: `docker build -f build/Dockerfile.harness -t kocao/harness-runtime:dev . && docker run --rm kocao/harness-runtime:dev bash -c "claude --version && opencode version && codex --version"`
- **Done When**: All three CLIs are installed, on PATH, and pass smoke test.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

______________________________________________________________________

## Wave 2: CRD extension and operator credential injection

- **Depends On**: None

### Task 2.1: Add agentAuth fields to HarnessRun CRD

- **Files**: `internal/operator/api/v1alpha1/types.go`, CRD manifests
- **Dependencies**: None
- **Action**: Add `AgentAuth *AgentAuth` field to `HarnessRunSpec`. Define `AgentAuth` struct with `ApiKeySecretName string` and `OauthSecretName string` fields (both optional). Regenerate CRD manifests.
- **Verify**: `go build ./... && go vet ./...`
- **Done When**: CRD types compile, manifests regenerated.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 2.2: Update buildHarnessPod to inject agent credentials

- **Files**: `internal/operator/controllers/pod.go`
- **Dependencies**: Task 2.1
- **Action**: In `buildHarnessPod()`, add logic after the git-auth block: (1) If `agentAuth.apiKeySecretName` is set, add `envFrom: [{secretRef: {name: <secretName>}}]` to the container. (2) If `agentAuth.oauthSecretName` is set, add a Secret volume with items mapped to the expected file paths (`opencode-auth.json` → `/home/kocao/.local/share/opencode/auth.json`, `codex-auth.json` → `/home/kocao/.codex/auth.json`) with `defaultMode: 0600`.
- **Verify**: `go test ./internal/operator/controllers/... -v -run TestBuildHarnessPod`
- **Done When**: Pod builder correctly adds envFrom and volume mounts when agentAuth fields are set; pods build correctly without agentAuth (backward compatible).
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 2.3: Add unit tests for agent credential injection

- **Files**: `internal/operator/controllers/pod_test.go` or `internal/operator/controllers/harnessrun_controller_test.go`
- **Dependencies**: Task 2.2
- **Action**: Add tests: (1) Pod without agentAuth has no extra envFrom or volumes. (2) Pod with apiKeySecretName has envFrom referencing the Secret. (3) Pod with oauthSecretName has volume + volumeMounts at correct paths with mode 0600. (4) Pod with both has envFrom AND volume mounts. (5) Pod with empty string secretNames is same as nil.
- **Verify**: `go test ./internal/operator/controllers/... -v -count=1`
- **Done When**: All tests pass, covering both tiers of credential injection.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

______________________________________________________________________

## Wave 3: Deploy manifests and integration test

- **Depends On**: Wave 2

### Task 3.1: Add sample Secret manifests for dev-kind overlay

- **Files**: `deploy/overlays/dev-kind/agent-api-keys.env`, `deploy/overlays/dev-kind/kustomization.yaml`
- **Dependencies**: None
- **Action**: Add a sample `kocao-agent-api-keys` Secret to the dev-kind overlay (with placeholder values). Add a sample `kocao-agent-oauth` Secret (empty or with placeholder). Update kustomization.yaml to include these resources. Document the Secret key names in comments.
- **Verify**: `kubectl apply -k deploy/overlays/dev-kind --dry-run=client`
- **Done When**: Secrets deploy cleanly to the dev-kind cluster.
- **Updated At**: 2026-02-27
- **Status**: [x] complete

### Task 3.2: Integration build, deploy, and verify

- **Files**: `Makefile` (if targets need updating)
- **Dependencies**: Task 3.1
- **Action**: Build full image with `make images`. Load into Kind with `make kind-load-images`. Apply updated CRD and deploy manifests. Create a test HarnessRun with `spec.agentAuth.apiKeySecretName` set. Verify the pod starts with the env vars injected and all three CLIs available.
- **Verify**: `make images && make kind-load-images && kubectl apply -k deploy/overlays/dev-kind && kubectl rollout restart deployment/control-plane-operator -n kocao-system`
- **Done When**: Harness run pod starts with agent CLIs available and credentials injected from Secrets.
- **Updated At**: 2026-02-27
- **Status**: [x] complete
<!-- ITO:END -->
