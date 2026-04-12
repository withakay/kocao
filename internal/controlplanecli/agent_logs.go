package controlplanecli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

func runAgentLogsCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	runID, flagArgs, err := parseRequiredAgentRunID("logs", args)
	if err != nil {
		return fmt.Errorf("usage: kocao agent logs <run-id> [--follow] [--tail N] [--output table|json]")
	}

	fs := newFlagSet("kocao agent logs", stderr)
	follow := fs.Bool("follow", false, "stream events via SSE")
	fs.BoolVar(follow, "f", false, "stream events via SSE (shorthand)")
	tail := fs.Int("tail", 0, "show last N events (0 = all)")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	if *tail < 0 {
		return fmt.Errorf("--tail must be >= 0")
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *follow {
		return streamAgentLogs(ctx, client, runID, format, stdout)
	}
	return fetchAgentLogs(ctx, client, runID, *tail, format, stdout)
}

func fetchAgentLogs(ctx context.Context, client *Client, runID string, tail int, format string, stdout io.Writer) error {
	events, err := client.GetEvents(ctx, runID)
	if err != nil {
		return err
	}
	if tail > 0 && tail < len(events) {
		events = events[len(events)-tail:]
	}
	if format == "json" {
		return writeEventsJSONL(stdout, events)
	}
	return writeEventsTable(stdout, events)
}

func streamAgentLogs(ctx context.Context, client *Client, runID string, format string, stdout io.Writer) error {
	rc, err := client.StreamEvents(ctx, runID)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}

		payload, ok := parseSSEDataLine(scanner.Text())
		if !ok {
			continue
		}

		var event AgentSessionEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if err := writeAgentEvent(stdout, format, event); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("read event stream: %w", err)
	}
	return nil
}

func parseSSEDataLine(line string) (string, bool) {
	if !strings.HasPrefix(line, "data:") {
		return "", false
	}
	payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if payload == "" {
		return "", false
	}
	return payload, true
}

func writeEventsTable(w io.Writer, events []AgentSessionEvent) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TIMESTAMP\tSEQ\tTYPE\tDATA"); err != nil {
		return err
	}
	for _, event := range events {
		if err := writeAgentEvent(tw, "table", event); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeEventsJSONL(w io.Writer, events []AgentSessionEvent) error {
	for _, event := range events {
		if err := writeAgentEvent(w, "json", event); err != nil {
			return err
		}
	}
	return nil
}

func formatEventTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

func extractEventType(data json.RawMessage) string {
	if len(data) == 0 {
		return "-"
	}
	var parsed struct {
		Type string `json:"type"`
	}
	if json.Unmarshal(data, &parsed) == nil && strings.TrimSpace(parsed.Type) != "" {
		return parsed.Type
	}
	return "-"
}
