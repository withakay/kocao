package controlplanecli

import (
	"fmt"
	"io"
	"strings"
)

// runAgentCommand dispatches agent management subcommands.
// It accepts the remaining args after "agent" has been consumed by the root dispatcher.
func runAgentCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeAgentUsage(stdout)
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "ls", "list":
		return runAgentListCommand(args[1:], cfg, stdout, stderr)
	case "start":
		return runAgentStartCommand(args[1:], cfg, stdout, stderr)
	case "stop":
		return runAgentStopCommand(args[1:], cfg, stdout, stderr)
	case "logs":
		return runAgentLogsCommand(args[1:], cfg, stdout, stderr)
	case "exec":
		return runAgentExecCommand(args[1:], cfg, stdout, stderr)
	case "status":
		return runAgentStatusCommand(args[1:], cfg, stdout, stderr)
	case "help", "-h", "--help":
		writeAgentUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown agent subcommand %q", sub)
	}
}

func writeAgentUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao agent — manage sandbox agents")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao agent <subcommand> [flags]")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Subcommands:")
	_, _ = fmt.Fprintln(w, "  list, ls    List agents")
	_, _ = fmt.Fprintln(w, "  start       Start an agent")
	_, _ = fmt.Fprintln(w, "  stop        Stop an agent")
	_, _ = fmt.Fprintln(w, "  logs        View agent logs")
	_, _ = fmt.Fprintln(w, "  exec        Execute a command in an agent")
	_, _ = fmt.Fprintln(w, "  status      Show agent status")
}
