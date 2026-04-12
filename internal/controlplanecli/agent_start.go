package controlplanecli

import (
	"context"
	"encoding/json"
	"flag"
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
	fs := flag.NewFlagSet("kocao agent start", flag.ContinueOnError)
	fs.SetOutput(stderr)

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

	_, _ = fmt.Fprintf(stderr, "Starting harness run... done\n")
	_, _ = fmt.Fprintf(stderr, "Waiting for agent session... ")

	session, err := pollAgentSession(ctx, client, runID, 2*time.Second)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "timeout")
		// On timeout, try to fetch last known state for diagnostics.
		lastSession, lastErr := client.GetAgentSession(context.Background(), runID)
		if lastErr == nil && lastSession != nil {
			_, _ = fmt.Fprintf(stderr, "Last known state: phase=%s sessionId=%s\n", lastSession.Phase, lastSession.SessionID)
		}
		return err
	}
	_, _ = fmt.Fprintln(stderr, "ready")

	result := agentStartResult{
		SessionID: session.SessionID,
		RunID:     session.RunID,
		Agent:     session.Agent,
		Phase:     session.Phase,
	}

	if strings.ToLower(strings.TrimSpace(*output)) == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	_, _ = fmt.Fprintf(stdout, "Session ID:  %s\n", result.SessionID)
	_, _ = fmt.Fprintf(stdout, "Run ID:      %s\n", result.RunID)
	_, _ = fmt.Fprintf(stdout, "Agent:       %s\n", result.Agent)
	_, _ = fmt.Fprintf(stdout, "Phase:       %s\n", result.Phase)
	return nil
}

// pollAgentSession polls GetAgentSession until the session reaches "Ready"
// phase or the context is cancelled. It uses the provided interval between
// poll attempts.
func pollAgentSession(ctx context.Context, client *Client, runID string, interval time.Duration) (*AgentSession, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for agent session to become ready")
		case <-ticker.C:
			session, err := client.GetAgentSession(ctx, runID)
			if err != nil {
				// Transient errors during polling are expected; keep trying
				// unless the context is done.
				if ctx.Err() != nil {
					return nil, fmt.Errorf("timed out waiting for agent session to become ready")
				}
				continue
			}
			if strings.EqualFold(strings.TrimSpace(session.Phase), "ready") {
				return session, nil
			}
		}
	}
}
