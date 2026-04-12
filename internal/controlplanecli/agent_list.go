package controlplanecli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"sigs.k8s.io/yaml"
)

func runAgentListCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("kocao agent list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	workspace := fs.String("workspace", "", "filter by workspace session ID")
	output := fs.String("output", "table", "output format: table, json, yaml")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format := strings.ToLower(strings.TrimSpace(*output))
	switch format {
	case "table", "json", "yaml":
	default:
		return fmt.Errorf("unsupported output format %q (use table, json, or yaml)", format)
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()

	sessions, err := collectAgentSessions(ctx, client, strings.TrimSpace(*workspace))
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		_, _ = fmt.Fprintln(stdout, "no agent sessions found")
		return nil
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

// collectAgentSessions fetches agent sessions. If workspaceID is non-empty,
// it fetches sessions for that workspace only. Otherwise it lists all workspaces
// and aggregates sessions across them.
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
	for _, s := range sessions {
		created := formatCreatedAt(s.CreatedAt)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n",
			s.SessionID,
			valueOrDash(s.RunID),
			valueOrDash(s.Agent),
			valueOrDash(s.Phase),
			valueOrDash(s.WorkspaceID),
			created,
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func formatCreatedAt(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func writeYAML(w io.Writer, v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}
	_, err = w.Write(b)
	return err
}
