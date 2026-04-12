package controlplanecli

import (
	"context"
	"fmt"
	"io"
)

func runAgentStatusCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	runID, flagArgs, err := parseRequiredAgentRunID("status", args)
	if err != nil {
		return fmt.Errorf("usage: kocao agent status <run-id> [--output table|json]")
	}

	fs := newFlagSet("kocao agent status", stderr)
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	session, err := client.GetAgentSession(context.Background(), runID)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, session)
	}
	return writeAgentSessionSummary(stdout, "", session)
}
