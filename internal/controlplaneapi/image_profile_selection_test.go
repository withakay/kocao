package controlplaneapi

import (
	"strings"
	"testing"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

func TestNormalizeHarnessImageProfileSelection(t *testing.T) {
	tests := []struct {
		name          string
		spec          *operatorv1alpha1.HarnessImageProfileSpec
		wantProfile   operatorv1alpha1.HarnessImageProfile
		wantPolicy    operatorv1alpha1.HarnessImageProfileSelectionPolicy
		wantSelected  operatorv1alpha1.HarnessImageProfile
		wantSource    operatorv1alpha1.HarnessImageProfileSelectionSource
		wantReason    string
		wantErrSubstr string
	}{
		{
			name:         "default auto falls back to compatibility profile",
			wantPolicy:   operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
			wantSelected: operatorv1alpha1.HarnessImageProfileFull,
			wantSource:   operatorv1alpha1.HarnessImageProfileSelectionSourceFallback,
			wantReason:   "auto-inference-pending",
		},
		{
			name:         "explicit profile wins",
			spec:         &operatorv1alpha1.HarnessImageProfileSpec{Profile: operatorv1alpha1.HarnessImageProfileGo},
			wantProfile:  operatorv1alpha1.HarnessImageProfileGo,
			wantPolicy:   operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
			wantSelected: operatorv1alpha1.HarnessImageProfileGo,
			wantSource:   operatorv1alpha1.HarnessImageProfileSelectionSourceExplicit,
			wantReason:   "explicit-request",
		},
		{
			name:         "preferred minimal policy selects base",
			spec:         &operatorv1alpha1.HarnessImageProfileSpec{SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal},
			wantPolicy:   operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal,
			wantSelected: operatorv1alpha1.HarnessImageProfileBase,
			wantSource:   operatorv1alpha1.HarnessImageProfileSelectionSourcePolicy,
			wantReason:   "policy-preferred-minimal",
		},
		{
			name:         "compatibility policy selects full",
			spec:         &operatorv1alpha1.HarnessImageProfileSpec{SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility},
			wantPolicy:   operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility,
			wantSelected: operatorv1alpha1.HarnessImageProfileFull,
			wantSource:   operatorv1alpha1.HarnessImageProfileSelectionSourcePolicy,
			wantReason:   "policy-compatibility",
		},
		{
			name:          "invalid profile rejected",
			spec:          &operatorv1alpha1.HarnessImageProfileSpec{Profile: "python"},
			wantErrSubstr: "unsupported profile",
		},
		{
			name:          "conflicting profile and policy rejected",
			spec:          &operatorv1alpha1.HarnessImageProfileSpec{Profile: operatorv1alpha1.HarnessImageProfileWeb, SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility},
			wantErrSubstr: "mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, status, err := normalizeHarnessImageProfileSelection(tt.spec)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeHarnessImageProfileSelection: %v", err)
			}
			if normalized == nil || status == nil {
				t.Fatal("expected normalized spec and status")
			}
			if normalized.Profile != tt.wantProfile {
				t.Fatalf("normalized profile = %q, want %q", normalized.Profile, tt.wantProfile)
			}
			if normalized.SelectionPolicy != tt.wantPolicy {
				t.Fatalf("normalized selectionPolicy = %q, want %q", normalized.SelectionPolicy, tt.wantPolicy)
			}
			if status.SelectedProfile != tt.wantSelected {
				t.Fatalf("selectedProfile = %q, want %q", status.SelectedProfile, tt.wantSelected)
			}
			if status.SelectionSource != tt.wantSource {
				t.Fatalf("selectionSource = %q, want %q", status.SelectionSource, tt.wantSource)
			}
			if status.Reason != tt.wantReason {
				t.Fatalf("reason = %q, want %q", status.Reason, tt.wantReason)
			}
			if status.FallbackProfile != operatorv1alpha1.HarnessImageProfileFull {
				t.Fatalf("fallbackProfile = %q, want full", status.FallbackProfile)
			}
		})
	}
}
