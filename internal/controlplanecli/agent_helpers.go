package controlplanecli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

func parseRequiredAgentRunID(command string, args []string) (string, []string, error) {
	usage := fmt.Sprintf("usage: kocao agent %s <run-id>", command)
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return "", nil, fmt.Errorf("%s", usage)
	}

	runID := strings.TrimSpace(args[0])
	if runID == "" {
		return "", nil, fmt.Errorf("%s", usage)
	}
	return runID, args[1:], nil
}

func parseAgentOutputFormat(raw string, allowed ...string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	for _, candidate := range allowed {
		if format == candidate {
			return format, nil
		}
	}
	if len(allowed) == 0 {
		return "", fmt.Errorf("no output formats configured")
	}
	return "", fmt.Errorf("unsupported output format %q (use %s)", format, strings.Join(allowed, ", "))
}

func formatAgentSessionCreatedAt(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func writeAgentSessionSummary(w io.Writer, heading string, session *AgentSession) error {
	if heading != "" {
		if _, err := fmt.Fprintln(w, heading); err != nil {
			return err
		}
	}

	lines := []struct {
		label string
		value string
	}{
		{"Run ID", valueOrDash(session.RunID)},
		{"Session ID", valueOrDash(session.SessionID)},
		{"Name", valueOrDash(session.DisplayName)},
		{"Runtime", valueOrDash(session.Runtime)},
		{"Agent", valueOrDash(session.Agent)},
		{"Phase", valueOrDash(session.Phase)},
		{"Workspace", valueOrDash(session.WorkspaceID)},
		{"Created", formatAgentSessionCreatedAt(session.CreatedAt)},
	}

	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "  %-10s %s\n", line.label+":", line.value); err != nil {
			return err
		}
	}
	return nil
}

func writeAgentEvent(w io.Writer, format string, event AgentSessionEvent) error {
	switch format {
	case "json":
		return json.NewEncoder(w).Encode(event)
	case "table":
		_, err := fmt.Fprintf(w, "%s\t%d\t%s\t%s\n",
			formatEventTimestamp(event.Timestamp),
			event.Seq,
			extractEventType(event.Data),
			truncateForLog(string(event.Data), 80),
		)
		return err
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func newFlagSet(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}
