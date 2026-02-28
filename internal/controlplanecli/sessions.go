package controlplanecli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

func runSessionsCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeSessionsUsage(stdout)
		return nil
	}

	ctx := context.Background()
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "ls", "list":
		return runSessionListCommand(ctx, cfg, args[1:], stdout, stderr)
	case "get", "inspect":
		return runSessionGetCommand(ctx, cfg, args[1:], stdout, stderr)
	case "status":
		return runSessionStatusCommand(ctx, cfg, args[1:], stdout, stderr)
	case "logs":
		return runSessionLogsCommand(cfg, args[1:], stdout, stderr)
	case "attach":
		return runSessionAttachCommand(cfg, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		writeSessionsUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown sessions subcommand %q", sub)
	}
}

func runSessionListCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("kocao sessions ls", flag.ContinueOnError)
	fs.SetOutput(stderr)
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	sessions, err := client.ListWorkspaceSessions(ctx)
	if err != nil {
		return err
	}
	sortSessionsNewestFirst(sessions)

	if *jsonOut {
		return writeJSON(stdout, map[string]any{"workspaceSessions": sessions})
	}
	return writeSessionsTable(stdout, sessions)
}

func runSessionGetCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao sessions get <workspace-session-id> [--json]")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" || strings.HasPrefix(sessionID, "-") {
		return fmt.Errorf("usage: kocao sessions get <workspace-session-id> [--json]")
	}

	fs := flag.NewFlagSet("kocao sessions get", flag.ContinueOnError)
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
	session, err := client.GetWorkspaceSession(ctx, sessionID)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(stdout, session)
	}
	name := session.DisplayName
	if strings.TrimSpace(name) == "" {
		name = session.ID
	}
	_, _ = fmt.Fprintf(stdout, "ID:         %s\n", session.ID)
	_, _ = fmt.Fprintf(stdout, "Name:       %s\n", name)
	_, _ = fmt.Fprintf(stdout, "Phase:      %s\n", valueOrDash(session.Phase))
	_, _ = fmt.Fprintf(stdout, "Repo URL:   %s\n", valueOrDash(session.RepoURL))
	_, _ = fmt.Fprintf(stdout, "Created At: %s\n", valueOrDash(session.CreatedAt))
	return nil
}

func runSessionStatusCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao sessions status <workspace-session-id> [--json]")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" || strings.HasPrefix(sessionID, "-") {
		return fmt.Errorf("usage: kocao sessions status <workspace-session-id> [--json]")
	}

	fs := flag.NewFlagSet("kocao sessions status", flag.ContinueOnError)
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
	status, err := BuildSessionStatus(ctx, client, sessionID)
	if err != nil {
		return err
	}

	if *jsonOut {
		return writeJSON(stdout, status)
	}
	name := status.Session.DisplayName
	if strings.TrimSpace(name) == "" {
		name = status.Session.ID
	}
	_, _ = fmt.Fprintf(stdout, "Session:    %s\n", name)
	_, _ = fmt.Fprintf(stdout, "Session ID: %s\n", status.Session.ID)
	_, _ = fmt.Fprintf(stdout, "Phase:      %s\n", valueOrDash(status.Session.Phase))
	_, _ = fmt.Fprintf(stdout, "Repo URL:   %s\n", valueOrDash(status.Session.RepoURL))
	if status.Run == nil {
		_, _ = fmt.Fprintln(stdout, "Run:        none")
		return nil
	}
	_, _ = fmt.Fprintf(stdout, "Run:        %s\n", status.Run.ID)
	_, _ = fmt.Fprintf(stdout, "Run Phase:  %s\n", valueOrDash(status.Run.Phase))
	_, _ = fmt.Fprintf(stdout, "Pod Name:   %s\n", valueOrDash(status.Run.PodName))
	return nil
}

func writeRootUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao CLI")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao [--config PATH] [--api-url URL] [--token TOKEN] [--timeout DURATION] [--verbose|--debug] <command>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Global Flags:")
	_, _ = fmt.Fprintln(w, "  --config    Config file path (.json)")
	_, _ = fmt.Fprintln(w, "  --verbose   Print HTTP request/response diagnostics")
	_, _ = fmt.Fprintln(w, "  --debug     Alias for --verbose")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Commands:")
	_, _ = fmt.Fprintln(w, "  sessions   Manage workspace sessions")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintf(w, "Environment:\n  %s (default: http://127.0.0.1:8080)\n  %s\n  %s (example: 15s)\n  %s (true|false)\n", EnvAPIURL, EnvToken, EnvTimeout, EnvVerbose)
}

func writeSessionsUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao sessions")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao sessions ls [--json]")
	_, _ = fmt.Fprintln(w, "  kocao sessions get <workspace-session-id> [--json]")
	_, _ = fmt.Fprintln(w, "  kocao sessions status <workspace-session-id> [--json]")
	_, _ = fmt.Fprintln(w, "  kocao sessions logs <workspace-session-id> [--tail N] [--container NAME] [--follow] [--json]")
	_, _ = fmt.Fprintln(w, "  kocao sessions attach <workspace-session-id> [--driver]")
}

func writeSessionsTable(w io.Writer, sessions []WorkspaceSession) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "ID\tNAME\tPHASE\tCREATED"); err != nil {
		return err
	}
	for _, s := range sessions {
		name := s.DisplayName
		if strings.TrimSpace(name) == "" {
			name = s.ID
		}
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", s.ID, name, valueOrDash(s.Phase), valueOrDash(s.CreatedAt)); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func valueOrDash(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "-"
	}
	return v
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
