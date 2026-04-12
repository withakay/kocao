package controlplanecli

import (
	"bytes"
	"strings"
	"testing"
)

func TestAgentCommand_NoArgs_ShowsHelp(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"list", "start", "stop", "logs", "exec", "status"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing subcommand %q, got:\n%s", want, out)
		}
	}
}

func TestAgentCommand_Help_ShowsHelp(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "kocao agent") {
		t.Errorf("help output missing header, got:\n%s", out)
	}
}

func TestAgentCommand_UnknownSubcommand(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent", "bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), `unknown agent subcommand "bogus"`) {
		t.Errorf("expected unknown subcommand error, got:\n%s", stderr.String())
	}
}

func TestAgentCommand_SharedFlags(t *testing.T) {
	t.Setenv(EnvToken, "")

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "api-url and token via flags",
			args: []string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token", "agent"},
		},
		{
			name: "api-url and token via env",
			args: []string{"agent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "api-url and token via env" {
				t.Setenv(EnvAPIURL, "http://127.0.0.1:9999")
				t.Setenv(EnvToken, "env-token")
			}
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := Main(tt.args, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
			}
			// Reaching the agent help means global flags were parsed successfully
			if !strings.Contains(stdout.String(), "kocao agent") {
				t.Errorf("expected agent help output, got:\n%s", stdout.String())
			}
		})
	}
}

func TestAgentCommand_SubcommandStubs(t *testing.T) {
	// All agent subcommands are now implemented — no stubs remain.
	// This test is kept as a placeholder; add new stubs here if needed.
	t.Skip("all agent subcommands are implemented")
}

func TestAgentCommand_RootUsageIncludesAgent(t *testing.T) {
	t.Setenv(EnvToken, "")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	// No subcommand → root help
	code := Main([]string{"--api-url", "http://127.0.0.1:9999", "--token", "test-token"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "agent") {
		t.Errorf("root usage missing 'agent' command, got:\n%s", stdout.String())
	}
}
