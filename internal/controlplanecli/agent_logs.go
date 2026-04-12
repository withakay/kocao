package controlplanecli

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
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
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return fmt.Errorf("usage: kocao agent logs <run-id> [--follow] [--tail N] [--output table|json]")
	}
	runID := strings.TrimSpace(args[0])
	if runID == "" {
		return fmt.Errorf("usage: kocao agent logs <run-id> [--follow] [--tail N] [--output table|json]")
	}

	fs := flag.NewFlagSet("kocao agent logs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	follow := fs.Bool("follow", false, "stream events via SSE")
	fs.BoolVar(follow, "f", false, "stream events via SSE (shorthand)")
	tail := fs.Int("tail", 0, "show last N events (0 = all)")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format := strings.ToLower(strings.TrimSpace(*output))
	if format != "table" && format != "json" {
		return fmt.Errorf("--output must be table or json")
	}

	if *tail < 0 {
		return fmt.Errorf("--tail must be >= 0")
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

// fetchAgentLogs retrieves all events for a run and displays them.
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

// streamAgentLogs opens an SSE stream and displays events as they arrive.
func streamAgentLogs(ctx context.Context, client *Client, runID string, format string, stdout io.Writer) error {
	rc, err := client.StreamEvents(ctx, runID)
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()

	scanner := bufio.NewScanner(rc)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}

		var event AgentSessionEvent
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			// Skip malformed events.
			continue
		}

		if format == "json" {
			b, _ := json.Marshal(event)
			_, _ = fmt.Fprintln(stdout, string(b))
		} else {
			_, _ = fmt.Fprintf(stdout, "%s\t%d\t%s\t%s\n",
				formatEventTimestamp(event.Timestamp),
				event.Seq,
				extractEventType(event.Data),
				truncateForLog(string(event.Data), 80),
			)
		}
	}

	if err := scanner.Err(); err != nil {
		// Context cancellation is expected during follow mode.
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("read event stream: %w", err)
	}
	return nil
}

// writeEventsTable renders events in a tabwriter table.
func writeEventsTable(w io.Writer, events []AgentSessionEvent) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TIMESTAMP\tSEQ\tTYPE\tDATA"); err != nil {
		return err
	}
	for _, e := range events {
		if _, err := fmt.Fprintf(tw, "%s\t%d\t%s\t%s\n",
			formatEventTimestamp(e.Timestamp),
			e.Seq,
			extractEventType(e.Data),
			truncateForLog(string(e.Data), 80),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// writeEventsJSONL writes events as newline-delimited JSON.
func writeEventsJSONL(w io.Writer, events []AgentSessionEvent) error {
	enc := json.NewEncoder(w)
	for _, e := range events {
		if err := enc.Encode(e); err != nil {
			return err
		}
	}
	return nil
}

// formatEventTimestamp formats a timestamp for table display.
func formatEventTimestamp(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format(time.RFC3339)
}

// extractEventType attempts to pull a "type" field from the raw JSON data.
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
