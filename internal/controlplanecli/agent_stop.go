package controlplanecli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

const agentStopFollowupTimeout = 5 * time.Second

func runAgentStopCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	runID, flagArgs, err := parseRequiredAgentRunID("stop", args)
	if err != nil {
		return fmt.Errorf("usage: kocao agent stop <run-id> [--json]")
	}

	fs := newFlagSet("kocao agent stop", stderr)
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	// Use a bounded context so the stop call does not hang indefinitely
	// when the sandbox-agent pod proxy is unresponsive.
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := client.StopAgentSession(ctx, runID); err != nil {
		var apiErr *APIError
		if !errors.As(err, &apiErr) || apiErr.StatusCode != 409 || !stopConflictIsTerminal(client, runID, timeout) {
			return err
		}
	}

	session, err := getAgentStopFollowupSession(client, runID, timeout)
	if err != nil {
		if *jsonOut {
			return writeJSON(stdout, map[string]any{"status": "stopped", "runId": runID})
		}
		_, _ = fmt.Fprintf(stdout, "Agent session stopped (run %s)\n", runID)
		return nil
	}
	if *jsonOut {
		return writeJSON(stdout, map[string]any{"status": "stopped", "session": session})
	}
	return writeAgentSessionSummary(stdout, "Agent session stopped", session)
}

func stopConflictIsTerminal(client *Client, runID string, timeout time.Duration) bool {
	session, err := getAgentStopFollowupSession(client, runID, timeout)
	if err != nil {
		return false
	}
	return operatorv1alpha1.NormalizeAgentSessionPhase(session.Phase).IsTerminal()
}

func getAgentStopFollowupSession(client *Client, runID string, timeout time.Duration) (*AgentSession, error) {
	ctx, cancel := context.WithTimeout(context.Background(), normalizedAgentStopFollowupTimeout(timeout))
	defer cancel()
	return client.GetAgentSession(ctx, runID)
}

func normalizedAgentStopFollowupTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 || timeout > agentStopFollowupTimeout {
		return agentStopFollowupTimeout
	}
	return timeout
}
