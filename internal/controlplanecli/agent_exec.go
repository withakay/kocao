package controlplanecli

import (
	"context"
	"encoding/json"
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
	fs := newFlagSet("kocao agent exec", stderr)

	var prompt string
	var output string
	fs.StringVar(&prompt, "prompt", "", "prompt text to send to the agent")
	fs.StringVar(&prompt, "p", "", "prompt text to send to the agent (shorthand)")
	fs.StringVar(&output, "output", "table", "output format: table or json")

	if len(args) == 0 {
		return fmt.Errorf("usage: kocao agent exec <run-id> [--prompt <text> | <text>]")
	}

	runID := ""
	var flagArgs []string
	for i, arg := range args {
		if !strings.HasPrefix(arg, "-") && runID == "" {
			runID = strings.TrimSpace(arg)
			flagArgs = append(flagArgs, args[i+1:]...)
			break
		}
		flagArgs = append(flagArgs, arg)
	}
	if runID == "" {
		return fmt.Errorf("usage: kocao agent exec <run-id> [--prompt <text> | <text>]")
	}
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}

	if strings.TrimSpace(prompt) == "" && len(fs.Args()) > 0 {
		prompt = strings.Join(fs.Args(), " ")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("usage: kocao agent exec <run-id> [--prompt <text> | <text>]")
	}

	format, err := parseAgentOutputFormat(output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	resp, err := client.SendPrompt(context.Background(), runID, prompt)
	if err != nil {
		return fmt.Errorf("send prompt: %w", err)
	}
	if format == "json" {
		return writeExecJSON(stdout, resp)
	}
	return writeExecTable(stdout, resp)
}

func writeExecJSON(w io.Writer, resp *PromptResponse) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(resp)
}

func writeExecTable(w io.Writer, resp *PromptResponse) error {
	if len(resp.Events) == 0 {
		_, err := fmt.Fprintln(w, "(no events)")
		return err
	}
	for _, ev := range resp.Events {
		if _, err := fmt.Fprintf(w, "[%s] seq=%d  %s\n", ev.Timestamp.Format("15:04:05"), ev.Seq, summarizeEventData(ev.Data)); err != nil {
			return err
		}
	}
	return nil
}

func summarizeEventData(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "(empty)"
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		s := strings.TrimSpace(string(raw))
		if len(s) > 120 {
			return s[:120] + "..."
		}
		return s
	}

	typ, _ := payload["type"].(string)
	text, _ := payload["text"].(string)
	switch {
	case typ != "" && text != "":
		return fmt.Sprintf("type=%s text=%q", typ, truncateForLog(text, 100))
	case typ != "":
		return fmt.Sprintf("type=%s", typ)
	case text != "":
		return fmt.Sprintf("text=%q", truncateForLog(text, 100))
	default:
		b, _ := json.Marshal(payload)
		s := string(b)
		if len(s) > 120 {
			return s[:120] + "..."
		}
		return s
	}
}
