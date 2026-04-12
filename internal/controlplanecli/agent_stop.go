package controlplanecli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

func runAgentStopCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return fmt.Errorf("usage: kocao agent stop <run-id> [--json]")
	}

	runID := strings.TrimSpace(args[0])
	if runID == "" {
		return fmt.Errorf("usage: kocao agent stop <run-id> [--json]")
	}

	fs := flag.NewFlagSet("kocao agent stop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()

	if err := client.StopAgentSession(ctx, runID); err != nil {
		var apiErr *APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 409 {
			return fmt.Errorf("agent session already stopped (run %s)", runID)
		}
		return err
	}

	session, err := client.GetAgentSession(ctx, runID)
	if err != nil {
		// Stop succeeded but fetching details failed — still report success.
		if *jsonOut {
			return writeJSON(stdout, map[string]any{
				"status": "stopped",
				"runId":  runID,
			})
		}
		_, _ = fmt.Fprintf(stdout, "Agent session stopped (run %s)\n", runID)
		return nil
	}

	if *jsonOut {
		return writeJSON(stdout, map[string]any{
			"status":  "stopped",
			"session": session,
		})
	}

	_, _ = fmt.Fprintln(stdout, "Agent session stopped")
	_, _ = fmt.Fprintf(stdout, "  Run ID:      %s\n", session.RunID)
	_, _ = fmt.Fprintf(stdout, "  Session ID:  %s\n", session.SessionID)
	_, _ = fmt.Fprintf(stdout, "  Name:        %s\n", valueOrDash(session.DisplayName))
	_, _ = fmt.Fprintf(stdout, "  Runtime:     %s\n", valueOrDash(session.Runtime))
	_, _ = fmt.Fprintf(stdout, "  Agent:       %s\n", valueOrDash(session.Agent))
	_, _ = fmt.Fprintf(stdout, "  Phase:       %s\n", valueOrDash(session.Phase))
	_, _ = fmt.Fprintf(stdout, "  Workspace:   %s\n", valueOrDash(session.WorkspaceID))
	return nil
}
