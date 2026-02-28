package controlplanecli

import "testing"

func TestSelectPreferredRunPrefersRunningWithPod(t *testing.T) {
	runs := []HarnessRun{
		{ID: "run-a", Phase: "Succeeded", PodName: "pod-old"},
		{ID: "run-b", Phase: "Running", PodName: "pod-live"},
		{ID: "run-c", Phase: "Starting", PodName: "pod-start"},
	}
	got := selectPreferredRun(runs)
	if got == nil {
		t.Fatalf("expected run selection")
	}
	if got.ID != "run-b" {
		t.Fatalf("selected run = %q, want run-b", got.ID)
	}
}

func TestDiffLogs(t *testing.T) {
	if got := diffLogs("", "abc"); got != "abc" {
		t.Fatalf("diffLogs first chunk = %q", got)
	}
	if got := diffLogs("abc", "abcdef"); got != "def" {
		t.Fatalf("diffLogs append = %q", got)
	}
	if got := diffLogs("abc", "abc"); got != "" {
		t.Fatalf("diffLogs same = %q", got)
	}
	if got := diffLogs("abc", "xyz"); got == "" {
		t.Fatalf("diffLogs reset should be non-empty")
	}
}
