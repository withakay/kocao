package controlplaneapi

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

const (
	harnessPreferredMinimalProfile = operatorv1alpha1.HarnessImageProfileBase
	harnessCompatibilityProfile    = operatorv1alpha1.HarnessImageProfileFull
)

func normalizeHarnessImageProfileSelection(spec *operatorv1alpha1.HarnessImageProfileSpec) (*operatorv1alpha1.HarnessImageProfileSpec, *operatorv1alpha1.HarnessImageProfileStatus, error) {
	if spec == nil {
		normalized, status := defaultHarnessImageProfileSelection()
		return normalized, status, nil
	}

	profile, err := normalizeHarnessImageProfile(spec.Profile)
	if err != nil {
		return nil, nil, fmt.Errorf("imageProfile.profile: %w", err)
	}
	policy, err := normalizeHarnessImageProfileSelectionPolicy(spec.SelectionPolicy)
	if err != nil {
		return nil, nil, fmt.Errorf("imageProfile.selectionPolicy: %w", err)
	}
	if profile != "" && policy != "" && policy != operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto {
		return nil, nil, fmt.Errorf("imageProfile.profile and imageProfile.selectionPolicy are mutually exclusive")
	}

	if profile != "" {
		return &operatorv1alpha1.HarnessImageProfileSpec{
				Profile:         profile,
				SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
			}, &operatorv1alpha1.HarnessImageProfileStatus{
				RequestedProfile: profile,
				SelectionPolicy:  operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
				SelectedProfile:  profile,
				SelectionSource:  operatorv1alpha1.HarnessImageProfileSelectionSourceExplicit,
				FallbackProfile:  harnessCompatibilityProfile,
				Reason:           "explicit-request",
			}, nil
	}

	switch policy {
	case "", operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto:
		normalized, status := defaultHarnessImageProfileSelection()
		return normalized, status, nil
	case operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal:
		return &operatorv1alpha1.HarnessImageProfileSpec{
				SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal,
			}, &operatorv1alpha1.HarnessImageProfileStatus{
				SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal,
				SelectedProfile: harnessPreferredMinimalProfile,
				SelectionSource: operatorv1alpha1.HarnessImageProfileSelectionSourcePolicy,
				FallbackProfile: harnessCompatibilityProfile,
				Reason:          "policy-preferred-minimal",
			}, nil
	case operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility:
		return &operatorv1alpha1.HarnessImageProfileSpec{
				SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility,
			}, &operatorv1alpha1.HarnessImageProfileStatus{
				SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility,
				SelectedProfile: harnessCompatibilityProfile,
				SelectionSource: operatorv1alpha1.HarnessImageProfileSelectionSourcePolicy,
				FallbackProfile: harnessCompatibilityProfile,
				Reason:          "policy-compatibility",
			}, nil
	default:
		return nil, nil, fmt.Errorf("unsupported selection policy %q", policy)
	}
}

func defaultHarnessImageProfileSelection() (*operatorv1alpha1.HarnessImageProfileSpec, *operatorv1alpha1.HarnessImageProfileStatus) {
	return &operatorv1alpha1.HarnessImageProfileSpec{
			SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
		}, &operatorv1alpha1.HarnessImageProfileStatus{
			SelectionPolicy: operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
			SelectedProfile: harnessCompatibilityProfile,
			SelectionSource: operatorv1alpha1.HarnessImageProfileSelectionSourceFallback,
			FallbackProfile: harnessCompatibilityProfile,
			Reason:          "auto-inference-pending",
		}
}

func harnessImageProfileStatusForRun(run *operatorv1alpha1.HarnessRun) *operatorv1alpha1.HarnessImageProfileStatus {
	if run == nil {
		return nil
	}
	if run.Status.ImageProfile != nil {
		return run.Status.ImageProfile
	}
	if status := harnessImageProfileStatusFromAnnotations(run.Annotations); status != nil {
		return status
	}
	_, status, err := normalizeHarnessImageProfileSelection(run.Spec.ImageProfile)
	if err != nil {
		return nil
	}
	return status
}

func harnessImageProfileAnnotations(status *operatorv1alpha1.HarnessImageProfileStatus) map[string]string {
	if status == nil {
		return nil
	}
	annotations := map[string]string{}
	if status.RequestedProfile != "" {
		annotations[annotationHarnessImageProfileRequested] = string(status.RequestedProfile)
	}
	if status.SelectionPolicy != "" {
		annotations[annotationHarnessImageProfilePolicy] = string(status.SelectionPolicy)
	}
	if status.SelectedProfile != "" {
		annotations[annotationHarnessImageProfileSelected] = string(status.SelectedProfile)
	}
	if status.SelectionSource != "" {
		annotations[annotationHarnessImageProfileSource] = string(status.SelectionSource)
	}
	if status.FallbackProfile != "" {
		annotations[annotationHarnessImageProfileFallback] = string(status.FallbackProfile)
	}
	if strings.TrimSpace(status.Reason) != "" {
		annotations[annotationHarnessImageProfileReason] = status.Reason
	}
	if len(annotations) == 0 {
		return nil
	}
	return annotations
}

func harnessImageProfileStatusFromAnnotations(annotations map[string]string) *operatorv1alpha1.HarnessImageProfileStatus {
	if len(annotations) == 0 {
		return nil
	}
	status := &operatorv1alpha1.HarnessImageProfileStatus{
		RequestedProfile: operatorv1alpha1.HarnessImageProfile(strings.TrimSpace(annotations[annotationHarnessImageProfileRequested])),
		SelectionPolicy:  operatorv1alpha1.HarnessImageProfileSelectionPolicy(strings.TrimSpace(annotations[annotationHarnessImageProfilePolicy])),
		SelectedProfile:  operatorv1alpha1.HarnessImageProfile(strings.TrimSpace(annotations[annotationHarnessImageProfileSelected])),
		SelectionSource:  operatorv1alpha1.HarnessImageProfileSelectionSource(strings.TrimSpace(annotations[annotationHarnessImageProfileSource])),
		FallbackProfile:  operatorv1alpha1.HarnessImageProfile(strings.TrimSpace(annotations[annotationHarnessImageProfileFallback])),
		Reason:           strings.TrimSpace(annotations[annotationHarnessImageProfileReason]),
	}
	if status.RequestedProfile == "" && status.SelectionPolicy == "" && status.SelectedProfile == "" && status.SelectionSource == "" && status.FallbackProfile == "" && status.Reason == "" {
		return nil
	}
	return status
}

func normalizeHarnessImageProfile(profile operatorv1alpha1.HarnessImageProfile) (operatorv1alpha1.HarnessImageProfile, error) {
	switch operatorv1alpha1.HarnessImageProfile(strings.ToLower(strings.TrimSpace(string(profile)))) {
	case "":
		return "", nil
	case operatorv1alpha1.HarnessImageProfileBase,
		operatorv1alpha1.HarnessImageProfileGo,
		operatorv1alpha1.HarnessImageProfileWeb,
		operatorv1alpha1.HarnessImageProfileFull:
		return operatorv1alpha1.HarnessImageProfile(strings.ToLower(strings.TrimSpace(string(profile)))), nil
	default:
		return "", fmt.Errorf("unsupported profile %q", profile)
	}
}

func normalizeHarnessImageProfileSelectionPolicy(policy operatorv1alpha1.HarnessImageProfileSelectionPolicy) (operatorv1alpha1.HarnessImageProfileSelectionPolicy, error) {
	switch operatorv1alpha1.HarnessImageProfileSelectionPolicy(strings.ToLower(strings.TrimSpace(string(policy)))) {
	case "":
		return "", nil
	case operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
		operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal,
		operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility:
		return operatorv1alpha1.HarnessImageProfileSelectionPolicy(strings.ToLower(strings.TrimSpace(string(policy)))), nil
	default:
		return "", fmt.Errorf("unsupported selection policy %q", policy)
	}
}
