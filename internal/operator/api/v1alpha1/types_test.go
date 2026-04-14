package v1alpha1

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSymphonyProjectApplyDefaults(t *testing.T) {
	project := &SymphonyProject{
		Spec: SymphonyProjectSpec{
			Source: SymphonyProjectSourceSpec{
				Project:        GitHubProjectRef{Owner: "withakay", Number: 12},
				TokenSecretRef: SecretKeyRef{Name: "github-token"},
				ActiveStates:   []string{" Todo "},
				TerminalStates: []string{" Done "},
			},
			Repositories: []SymphonyProjectRepositorySpec{{Owner: "withakay", Name: "kocao"}},
			Runtime:      SymphonyProjectRuntimeSpec{Image: "ghcr.io/withakay/kocao-harness:latest"},
		},
	}

	project.ApplyDefaults()

	if got := project.Spec.Source.PollIntervalSec; got != DefaultSymphonyPollIntervalSeconds {
		t.Fatalf("expected poll interval default %d, got %d", DefaultSymphonyPollIntervalSeconds, got)
	}
	if got := project.Spec.Source.FieldName; got != "Status" {
		t.Fatalf("expected field name default Status, got %q", got)
	}
	if got := project.Spec.Runtime.MaxConcurrentItems; got != DefaultSymphonyMaxConcurrentItems {
		t.Fatalf("expected max concurrency default %d, got %d", DefaultSymphonyMaxConcurrentItems, got)
	}
	if got := project.Spec.Runtime.DefaultEgressMode; got != "restricted" {
		t.Fatalf("expected default egress mode restricted, got %q", got)
	}
	if project.Spec.Source.ActiveStates[0] != "Todo" {
		t.Fatalf("expected trimmed active state, got %q", project.Spec.Source.ActiveStates[0])
	}
	if project.Spec.Source.TerminalStates[0] != "Done" {
		t.Fatalf("expected trimmed terminal state, got %q", project.Spec.Source.TerminalStates[0])
	}
	if err := project.Validate(); err != nil {
		t.Fatalf("expected defaults to produce a valid project, got %v", err)
	}
}

func TestSymphonyProjectValidateRejectsInvalidConfig(t *testing.T) {
	project := &SymphonyProject{}
	project.ApplyDefaults()

	err := project.Validate()
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "spec.source.project.owner") {
		t.Fatalf("expected owner validation error, got %v", err)
	}

	project = &SymphonyProject{
		Spec: SymphonyProjectSpec{
			Source: SymphonyProjectSourceSpec{
				Project:         GitHubProjectRef{Owner: "withakay", Number: 7},
				TokenSecretRef:  SecretKeyRef{Name: "github-token"},
				ActiveStates:    []string{"Todo"},
				TerminalStates:  []string{"Done"},
				PollIntervalSec: 60,
			},
			Repositories: []SymphonyProjectRepositorySpec{
				{Owner: "withakay", Name: "kocao"},
				{Owner: "Withakay", Name: "KOCAO"},
			},
			Runtime: SymphonyProjectRuntimeSpec{
				Image:                 "ghcr.io/withakay/kocao-harness:latest",
				MaxConcurrentItems:    1,
				RetryBaseDelaySeconds: 60,
				RetryMaxDelaySeconds:  300,
			},
		},
	}

	err = project.Validate()
	if err == nil {
		t.Fatal("expected duplicate repository validation error, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate repository") {
		t.Fatalf("expected duplicate repository error, got %v", err)
	}
}

func TestAddToSchemeRegistersSymphonyProject(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}

	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&SymphonyProject{}).Build()
	project := &SymphonyProject{
		TypeMeta: metav1.TypeMeta{APIVersion: GroupVersion.String(), Kind: "SymphonyProject"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-a",
			Namespace: "default",
		},
		Spec: SymphonyProjectSpec{
			Source: SymphonyProjectSourceSpec{
				Project:         GitHubProjectRef{Owner: "withakay", Number: 42},
				TokenSecretRef:  SecretKeyRef{Name: "github-token"},
				ActiveStates:    []string{"Todo"},
				TerminalStates:  []string{"Done"},
				PollIntervalSec: 60,
			},
			Repositories: []SymphonyProjectRepositorySpec{{Owner: "withakay", Name: "kocao"}},
			Runtime: SymphonyProjectRuntimeSpec{
				Image:                 "ghcr.io/withakay/kocao-harness:latest",
				MaxConcurrentItems:    1,
				RetryBaseDelaySeconds: 60,
				RetryMaxDelaySeconds:  300,
			},
		},
	}
	if err := cl.Create(t.Context(), project); err != nil {
		t.Fatalf("create: %v", err)
	}

	var got SymphonyProject
	if err := cl.Get(t.Context(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get: %v", err)
	}
	got.Status.Phase = SymphonyProjectPhaseReady
	if err := cl.Status().Update(t.Context(), &got); err != nil {
		t.Fatalf("status update: %v", err)
	}

	var updated SymphonyProject
	if err := cl.Get(t.Context(), client.ObjectKeyFromObject(project), &updated); err != nil {
		t.Fatalf("get updated: %v", err)
	}
	if updated.Status.Phase != SymphonyProjectPhaseReady {
		t.Fatalf("expected status phase %q, got %q", SymphonyProjectPhaseReady, updated.Status.Phase)
	}
}

func TestAgentSessionSpecApplyDefaultsAndValidate(t *testing.T) {
	spec := &AgentSessionSpec{Agent: AgentKind(" CODEX ")}
	spec.ApplyDefaults()
	if spec.Runtime != AgentRuntimeSandboxAgent {
		t.Fatalf("runtime = %q, want %q", spec.Runtime, AgentRuntimeSandboxAgent)
	}
	if spec.Agent != AgentKindCodex {
		t.Fatalf("agent = %q, want %q", spec.Agent, AgentKindCodex)
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	mockSpec := &AgentSessionSpec{Agent: AgentKind(" mock ")}
	mockSpec.ApplyDefaults()
	if mockSpec.Agent != AgentKindMock {
		t.Fatalf("mock agent = %q, want %q", mockSpec.Agent, AgentKindMock)
	}
	if err := mockSpec.Validate(); err != nil {
		t.Fatalf("validate mock: %v", err)
	}
}

func TestAgentSessionSpecValidateRejectsInvalidConfig(t *testing.T) {
	cases := []AgentSessionSpec{
		{Runtime: AgentRuntime("custom"), Agent: AgentKindCodex},
		{Runtime: AgentRuntimeSandboxAgent},
		{Runtime: AgentRuntimeSandboxAgent, Agent: AgentKind("cursor")},
	}
	for _, tc := range cases {
		tc.ApplyDefaults()
		if err := tc.Validate(); err == nil {
			t.Fatalf("expected validation error for %#v", tc)
		}
	}
}

func TestNormalizeAgentSessionPhase(t *testing.T) {
	cases := map[string]AgentSessionPhase{
		"":               "",
		" Provisioning ": AgentSessionPhaseProvisioning,
		"ready":          AgentSessionPhaseReady,
		"running":        AgentSessionPhaseRunning,
		"Active":         AgentSessionPhaseRunning,
		"stopping":       AgentSessionPhaseStopping,
		"completed":      AgentSessionPhaseCompleted,
		"failed":         AgentSessionPhaseFailed,
	}
	for input, want := range cases {
		if got := NormalizeAgentSessionPhase(input); got != want {
			t.Fatalf("NormalizeAgentSessionPhase(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAgentSessionPhaseCanTransitionTo(t *testing.T) {
	tests := []struct {
		name string
		from AgentSessionPhase
		to   AgentSessionPhase
		want bool
	}{
		{name: "create starts provisioning", from: "", to: AgentSessionPhaseProvisioning, want: true},
		{name: "provisioning to ready", from: AgentSessionPhaseProvisioning, to: AgentSessionPhaseReady, want: true},
		{name: "ready to running", from: AgentSessionPhaseReady, to: AgentSessionPhaseRunning, want: true},
		{name: "running to stopping", from: AgentSessionPhaseRunning, to: AgentSessionPhaseStopping, want: true},
		{name: "stopping to completed", from: AgentSessionPhaseStopping, to: AgentSessionPhaseCompleted, want: true},
		{name: "ready cannot go back to provisioning", from: AgentSessionPhaseReady, to: AgentSessionPhaseProvisioning, want: false},
		{name: "completed is durable", from: AgentSessionPhaseCompleted, to: AgentSessionPhaseRunning, want: false},
		{name: "failed is durable", from: AgentSessionPhaseFailed, to: AgentSessionPhaseReady, want: false},
		{name: "active alias normalizes to running", from: AgentSessionPhaseActive, to: AgentSessionPhaseStopping, want: true},
	}
	for _, tt := range tests {
		if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
			t.Fatalf("%s: %q -> %q = %v, want %v", tt.name, tt.from, tt.to, got, tt.want)
		}
	}
}

func TestAgentSessionPhaseIsTerminal(t *testing.T) {
	if !AgentSessionPhaseCompleted.IsTerminal() {
		t.Fatal("completed phase should be terminal")
	}
	if !AgentSessionPhaseFailed.IsTerminal() {
		t.Fatal("failed phase should be terminal")
	}
	if AgentSessionPhaseRunning.IsTerminal() {
		t.Fatal("running phase should not be terminal")
	}
}
