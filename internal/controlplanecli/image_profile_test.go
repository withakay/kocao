package controlplanecli

import (
	"strings"
	"testing"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

func TestParseImageProfileSelection(t *testing.T) {
	tests := []struct {
		name          string
		profile       string
		policy        string
		wantProfile   operatorv1alpha1.HarnessImageProfile
		wantPolicy    operatorv1alpha1.HarnessImageProfileSelectionPolicy
		wantNil       bool
		wantErrSubstr string
	}{
		{
			name:        "explicit profile",
			profile:     "go",
			wantProfile: operatorv1alpha1.HarnessImageProfileGo,
		},
		{
			name:       "policy selection",
			policy:     "preferred-minimal",
			wantPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal,
		},
		{
			name:        "explicit profile with auto policy",
			profile:     "web",
			policy:      "auto",
			wantProfile: operatorv1alpha1.HarnessImageProfileWeb,
			wantPolicy:  operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
		},
		{
			name:    "omitted selection",
			wantNil: true,
		},
		{
			name:          "reject conflicting explicit policy",
			profile:       "base",
			policy:        "compatibility",
			wantErrSubstr: "mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseImageProfileSelection(tt.profile, tt.policy)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseImageProfileSelection: %v", err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("selection = %#v, want nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("selection is nil")
			}
			if got.Profile != tt.wantProfile {
				t.Fatalf("profile = %q, want %q", got.Profile, tt.wantProfile)
			}
			if got.SelectionPolicy != tt.wantPolicy {
				t.Fatalf("selectionPolicy = %q, want %q", got.SelectionPolicy, tt.wantPolicy)
			}
		})
	}
}

func TestFormatHarnessImageProfile(t *testing.T) {
	if got := formatHarnessImageProfile(&operatorv1alpha1.HarnessImageProfileStatus{
		SelectedProfile: operatorv1alpha1.HarnessImageProfileBase,
		SelectionSource: operatorv1alpha1.HarnessImageProfileSelectionSourcePolicy,
	}); got != "base (policy)" {
		t.Fatalf("formatHarnessImageProfile = %q, want %q", got, "base (policy)")
	}

	if got := formatHarnessImageProfile(&operatorv1alpha1.HarnessImageProfileStatus{}); got != "-" {
		t.Fatalf("empty status = %q, want -", got)
	}
}
