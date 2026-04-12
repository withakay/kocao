package controlplanecli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// agentStartResult holds the output data for a successful agent start.
type agentStartResult struct {
	SessionID string `json:"sessionId"`
	RunID     string `json:"runId"`
	Agent     string `json:"agent"`
	Phase     string `json:"phase"`
}

func runAgentStartCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := newFlagSet("kocao agent start", stderr)

	repo := fs.String("repo", "", "repository URL (required)")
	agent := fs.String("agent", "", "agent name, e.g. codex, claude, opencode, pi (required)")
	workspace := fs.String("workspace", "", "reuse existing workspace session ID")
	revision := fs.String("revision", "main", "repository revision")
	image := fs.String("image", "kocao/harness-runtime:dev", "harness runtime image")
	timeout := fs.Duration("timeout", 5*time.Minute, "timeout waiting for agent to become ready")
	output := fs.String("output", "table", "output format: table or json")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*repo) == "" {
		return fmt.Errorf("usage: kocao agent start --repo <url> --agent <name>: missing required flag --repo")
	}
	if strings.TrimSpace(*agent) == "" {
		return fmt.Errorf("usage: kocao agent start --repo <url> --agent <name>: missing required flag --agent")
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	_, _ = fmt.Fprintf(stderr, "Creating workspace session... ")
	runID, err := client.StartAgent(ctx, *workspace, *repo, *revision, *agent, *image)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "failed")
		return fmt.Errorf("start agent: %w", err)
	}
	_, _ = fmt.Fprintln(stderr, "done")

	_, _ = fmt.Fprintln(stderr, "Starting harness run... done")
	_, _ = fmt.Fprintf(stderr, "Waiting for agent session... ")

	session, err := pollAgentSession(ctx, client, runID, 2*time.Second)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "timeout")
		printLastKnownAgentState(stderr, client, runID)
		return err
	}
	_, _ = fmt.Fprintln(stderr, "ready")

	result := agentStartResult{
		SessionID: session.SessionID,
		RunID:     session.RunID,
		Agent:     session.Agent,
		Phase:     session.Phase,
	}
	if format == "json" {
		return writeJSON(stdout, result)
	}

	_, _ = fmt.Fprintf(stdout, "Session ID:  %s\n", result.SessionID)
	_, _ = fmt.Fprintf(stdout, "Run ID:      %s\n", result.RunID)
	_, _ = fmt.Fprintf(stdout, "Agent:       %s\n", result.Agent)
	_, _ = fmt.Fprintf(stdout, "Phase:       %s\n", result.Phase)
	return nil
}

func printLastKnownAgentState(stderr io.Writer, client *Client, runID string) {
	lastSession, err := client.GetAgentSession(context.Background(), runID)
	if err != nil || lastSession == nil {
		return
	}
	_, _ = fmt.Fprintf(stderr, "Last known state: phase=%s sessionId=%s\n", lastSession.Phase, lastSession.SessionID)
}

// pollAgentSession polls GetAgentSession until the session reaches "Ready"
// phase or the context is cancelled. It uses the provided interval between
// poll attempts.
func pollAgentSession(ctx context.Context, client *Client, runID string, interval time.Duration) (*AgentSession, error) {
	for {
		session, err := client.GetAgentSession(ctx, runID)
		if err == nil && strings.EqualFold(strings.TrimSpace(session.Phase), "ready") {
			return session, nil
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("timed out waiting for agent session to become ready")
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("timed out waiting for agent session to become ready")
		case <-timer.C:
		}
	}
}
