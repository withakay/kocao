package controllers

import "testing"

func TestParseGitHubEgressCIDRs(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		valid, invalid := parseGitHubEgressCIDRs("")
		if len(valid) != 0 {
			t.Fatalf("expected no valid CIDRs, got %v", valid)
		}
		if len(invalid) != 0 {
			t.Fatalf("expected no invalid CIDRs, got %v", invalid)
		}
	})

	t.Run("valid", func(t *testing.T) {
		valid, invalid := parseGitHubEgressCIDRs("192.30.252.0/22,185.199.108.0/22")
		wantValid := []string{"192.30.252.0/22", "185.199.108.0/22"}
		if len(invalid) != 0 {
			t.Fatalf("expected no invalid CIDRs, got %v", invalid)
		}
		if len(valid) != len(wantValid) {
			t.Fatalf("expected %d valid CIDRs, got %v", len(wantValid), valid)
		}
		for i := range wantValid {
			if valid[i] != wantValid[i] {
				t.Fatalf("expected valid[%d]=%q, got %q", i, wantValid[i], valid[i])
			}
		}
	})

	t.Run("mixed", func(t *testing.T) {
		valid, invalid := parseGitHubEgressCIDRs(" 192.30.252.0/22 , , not-a-cidr , 2001:db8::/32, 1.2.3.4 ")
		wantValid := []string{"192.30.252.0/22", "2001:db8::/32"}
		wantInvalid := []string{"not-a-cidr", "1.2.3.4"}

		if len(valid) != len(wantValid) {
			t.Fatalf("expected %d valid CIDRs, got %v", len(wantValid), valid)
		}
		for i := range wantValid {
			if valid[i] != wantValid[i] {
				t.Fatalf("expected valid[%d]=%q, got %q", i, wantValid[i], valid[i])
			}
		}

		if len(invalid) != len(wantInvalid) {
			t.Fatalf("expected %d invalid CIDRs, got %v", len(wantInvalid), invalid)
		}
		for i := range wantInvalid {
			if invalid[i] != wantInvalid[i] {
				t.Fatalf("expected invalid[%d]=%q, got %q", i, wantInvalid[i], invalid[i])
			}
		}
	})
}

func TestGitHubEgressCIDRs_UsesEnv(t *testing.T) {
	t.Parallel()

	t.Setenv(envGitHubEgressCIDRs, "192.30.252.0/22, not-a-cidr")
	valid, invalid := githubEgressCIDRs()

	if len(valid) != 1 || valid[0] != "192.30.252.0/22" {
		t.Fatalf("unexpected valid CIDRs: %v", valid)
	}
	if len(invalid) != 1 || invalid[0] != "not-a-cidr" {
		t.Fatalf("unexpected invalid CIDRs: %v", invalid)
	}
}
