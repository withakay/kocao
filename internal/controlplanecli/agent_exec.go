package controlplanecli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"
)

// runAgentExecCommand sends a prompt to a running agent session and displays
// the response events.
//
// Usage:
//
//	kocao agent exec <run-id> --prompt "your prompt"
//	kocao agent exec <run-id> "your prompt"
func runAgentExecCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("kocao agent exec", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var prompt string
	var output string
	fs.StringVar(&prompt, "prompt", "", "prompt text to send to the agent")
	fs.StringVar(&prompt, "p", "", "prompt text to send to the agent (shorthand)")
	fs.StringVar(&output, "output", "table", "output format: table or json")

	// We need to extract the run-id (first positional arg) before flag parsing,
	// because flag.Parse stops at the first non-flag argument.
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao agent exec <run-id> [--prompt <text> | <text>]")
	}

	// First arg that doesn't start with "-" is the run-id.
	runID := ""
	var flagArgs []string
	for i, a := range args {
		if !strings.HasPrefix(a, "-") && runID == "" {
			runID = strings.TrimSpace(a)
			flagArgs = append(flagArgs, args[i+1:]...)
			break
		}
		flagArgs = append(flagArgs, a)
	}

	if runID == "" {
		return fmt.Errorf("usage: kocao agent exec <run-id> [--prompt <text> | <text>]")
	}

	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	// If --prompt was not set, use remaining positional args as the prompt.
	if strings.TrimSpace(prompt) == "" {
		remaining := fs.Args()
		if len(remaining) > 0 {
			prompt = strings.Join(remaining, " ")
		}
	}

	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("usage: kocao agent exec <run-id> [--prompt <text> | <text>]")
	}

	output = strings.ToLower(strings.TrimSpace(output))
	if output != "table" && output != "json" {
		return fmt.Errorf("usage: --output must be \"table\" or \"json\"")
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx := context.Background()
	resp, err := client.SendPrompt(ctx, runID, prompt)
	if err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}

	if output == "json" {
		return writeExecJSON(stdout, resp)
	}
	return writeExecTable(stdout, resp)
}

// writeExecJSON writes the prompt response events as raw JSON.
func writeExecJSON(w io.Writer, resp *PromptResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

// writeExecTable writes the prompt response events in a human-readable table.
func writeExecTable(w io.Writer, resp *PromptResponse) error {
	if len(resp.Events) == 0 {
		_, err := fmt.Fprintln(w, "(no events)")
		return err
	}
	for _, ev := range resp.Events {
		ts := ev.Timestamp.Format("15:04:05")
		summary := summarizeEventData(ev.Data)
		if _, err := fmt.Fprintf(w, "[%s] seq=%d  %s\n", ts, ev.Seq, summary); err != nil {
			return err
		}
	}
	return nil
}

// summarizeEventData produces a short human-readable summary of the event's
// JSON data payload.
func summarizeEventData(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(empty)"
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		// Not a JSON object — return the raw string, truncated.
		s := strings.TrimSpace(string(raw))
		if len(s) > 120 {
			return s[:120] + "..."
		}
		return s
	}

	// Try to produce a useful one-liner from common fields.
	typ, _ := m["type"].(string)
	text, _ := m["text"].(string)

	switch {
	case typ != "" && text != "":
		return fmt.Sprintf("type=%s text=%q", typ, truncateForLog(text, 100))
	case typ != "":
		return fmt.Sprintf("type=%s", typ)
	case text != "":
		return fmt.Sprintf("text=%q", truncateForLog(text, 100))
	default:
		b, _ := json.Marshal(m)
		s := string(b)
		if len(s) > 120 {
			return s[:120] + "..."
		}
		return s
	}
}
