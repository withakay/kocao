package controlplanecli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"

	"sigs.k8s.io/yaml"
)

func runAgentListCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := newFlagSet("kocao agent list", stderr)
	workspace := fs.String("workspace", "", "filter by workspace session ID")
	output := fs.String("output", "table", "output format: table, json, yaml")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format, err := parseAgentOutputFormat(*output, "table", "json", "yaml")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	sessions, err := collectAgentSessions(ctx, client, strings.TrimSpace(*workspace))
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		switch format {
		case "json":
			return writeJSON(stdout, []AgentSession{})
		case "yaml":
			return writeYAML(stdout, []AgentSession{})
		default:
			_, _ = fmt.Fprintln(stdout, "no agent sessions found")
			return nil
		}
	}

	switch format {
	case "json":
		return writeJSON(stdout, sessions)
	case "yaml":
		return writeYAML(stdout, sessions)
	default:
		return writeAgentSessionsTable(stdout, sessions)
	}
}

func collectAgentSessions(ctx context.Context, client *Client, workspaceID string) ([]AgentSession, error) {
	if workspaceID != "" {
		return client.ListAgentSessions(ctx, workspaceID)
	}

	workspaces, err := client.ListWorkspaceSessions(ctx)
	if err != nil {
		return nil, err
	}

	var all []AgentSession
	for _, ws := range workspaces {
		sessions, err := client.ListAgentSessions(ctx, ws.ID)
		if err != nil {
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.StatusCode == 404 {
				continue
			}
			return nil, fmt.Errorf("list agent sessions for workspace %s: %w", ws.ID, err)
		}
		all = append(all, sessions...)
	}
	return all, nil
}

func writeAgentSessionsTable(w io.Writer, sessions []AgentSession) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SESSION ID\tRUN\tAGENT\tPHASE\tWORKSPACE\tCREATED"); err != nil {
		return err
	}
	for _, session := range sessions {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			session.SessionID,
			valueOrDash(session.RunID),
			valueOrDash(session.Agent),
			valueOrDash(session.Phase),
			valueOrDash(session.WorkspaceID),
			formatAgentSessionCreatedAt(session.CreatedAt),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeYAML(w io.Writer, v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	_, err = w.Write(b)
	return err
}
