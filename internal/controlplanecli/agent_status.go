package controlplanecli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
)

func runAgentStatusCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return fmt.Errorf("usage: kocao agent status <run-id> [--output table|json]")
	}
	runID := strings.TrimSpace(args[0])
	if runID == "" {
		return fmt.Errorf("usage: kocao agent status <run-id> [--output table|json]")
	}

	fs := flag.NewFlagSet("kocao agent status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format := strings.ToLower(strings.TrimSpace(*output))
	if format != "table" && format != "json" {
		return fmt.Errorf("usage: --output must be \"table\" or \"json\"")
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	session, err := client.GetAgentSession(ctx, runID)
	if err != nil {
		return err
	}

	if format == "json" {
		return writeJSON(stdout, session)
	}

	createdAt := "-"
	if !session.CreatedAt.IsZero() {
		createdAt = session.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	}

	_, _ = fmt.Fprintf(stdout, "Session ID:   %s\n", valueOrDash(session.SessionID))
	_, _ = fmt.Fprintf(stdout, "Run ID:       %s\n", valueOrDash(session.RunID))
	_, _ = fmt.Fprintf(stdout, "Agent:        %s\n", valueOrDash(session.Agent))
	_, _ = fmt.Fprintf(stdout, "Runtime:      %s\n", valueOrDash(session.Runtime))
	_, _ = fmt.Fprintf(stdout, "Phase:        %s\n", valueOrDash(session.Phase))
	_, _ = fmt.Fprintf(stdout, "Workspace:    %s\n", valueOrDash(session.WorkspaceID))
	_, _ = fmt.Fprintf(stdout, "Created:      %s\n", createdAt)
	return nil
}
