package controlplanecli

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

func parseImageProfileSelection(profile string, policy string) (*operatorv1alpha1.HarnessImageProfileSpec, error) {
	normalizedProfile, err := normalizeCLIHarnessImageProfile(profile)
	if err != nil {
		return nil, err
	}
	normalizedPolicy, err := normalizeCLIHarnessImageProfileSelectionPolicy(policy)
	if err != nil {
		return nil, err
	}
	if normalizedProfile != "" && normalizedPolicy != "" && normalizedPolicy != operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto {
		return nil, fmt.Errorf("--image-profile and --image-profile-policy are mutually exclusive")
	}
	if normalizedProfile == "" && normalizedPolicy == "" {
		return nil, nil
	}
	return &operatorv1alpha1.HarnessImageProfileSpec{
		Profile:         normalizedProfile,
		SelectionPolicy: normalizedPolicy,
	}, nil
}

func normalizeCLIHarnessImageProfile(raw string) (operatorv1alpha1.HarnessImageProfile, error) {
	switch operatorv1alpha1.HarnessImageProfile(strings.ToLower(strings.TrimSpace(raw))) {
	case "":
		return "", nil
	case operatorv1alpha1.HarnessImageProfileBase,
		operatorv1alpha1.HarnessImageProfileGo,
		operatorv1alpha1.HarnessImageProfileWeb,
		operatorv1alpha1.HarnessImageProfileFull:
		return operatorv1alpha1.HarnessImageProfile(strings.ToLower(strings.TrimSpace(raw))), nil
	default:
		return "", fmt.Errorf("unsupported --image-profile %q (use base, go, web, full)", strings.TrimSpace(raw))
	}
}

func normalizeCLIHarnessImageProfileSelectionPolicy(raw string) (operatorv1alpha1.HarnessImageProfileSelectionPolicy, error) {
	switch operatorv1alpha1.HarnessImageProfileSelectionPolicy(strings.ToLower(strings.TrimSpace(raw))) {
	case "":
		return "", nil
	case operatorv1alpha1.HarnessImageProfileSelectionPolicyAuto,
		operatorv1alpha1.HarnessImageProfileSelectionPolicyPreferredMinimal,
		operatorv1alpha1.HarnessImageProfileSelectionPolicyCompatibility:
		return operatorv1alpha1.HarnessImageProfileSelectionPolicy(strings.ToLower(strings.TrimSpace(raw))), nil
	default:
		return "", fmt.Errorf("unsupported --image-profile-policy %q (use auto, preferred-minimal, compatibility)", strings.TrimSpace(raw))
	}
}

func formatHarnessImageProfile(status *operatorv1alpha1.HarnessImageProfileStatus) string {
	if status == nil || strings.TrimSpace(string(status.SelectedProfile)) == "" {
		return "-"
	}
	selected := string(status.SelectedProfile)
	source := strings.TrimSpace(string(status.SelectionSource))
	if source == "" {
		return selected
	}
	return selected + " (" + source + ")"
}
