package controlplanecli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func runSessionLogsCommand(cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao sessions logs <workspace-session-id> [--tail N] [--container NAME] [--follow] [--json]")
	}
	sessionID := strings.TrimSpace(args[0])
	if sessionID == "" || strings.HasPrefix(sessionID, "-") {
		return fmt.Errorf("usage: kocao sessions logs <workspace-session-id> [--tail N] [--container NAME] [--follow] [--json]")
	}

	fs := flag.NewFlagSet("kocao sessions logs", flag.ContinueOnError)
	fs.SetOutput(stderr)
	tail := fs.Int64("tail", 200, "number of log lines to request")
	container := fs.String("container", "", "container name")
	follow := fs.Bool("follow", false, "poll logs continuously")
	interval := fs.Duration("interval", 2*time.Second, "poll interval for --follow")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	if *jsonOut && *follow {
		return fmt.Errorf("--json cannot be combined with --follow")
	}
	if *tail <= 0 {
		return fmt.Errorf("--tail must be greater than 0")
	}
	if *interval <= 0 {
		return fmt.Errorf("--interval must be greater than 0")
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	status, err := BuildSessionStatus(ctx, client, sessionID)
	if err != nil {
		return err
	}
	if status.Run == nil || strings.TrimSpace(status.Run.PodName) == "" {
		return fmt.Errorf("no active run pod for workspace session %q", sessionID)
	}

	if !*follow {
		resp, err := client.GetPodLogs(ctx, status.Run.PodName, *container, *tail)
		if err != nil {
			return err
		}
		if *jsonOut {
			return writeJSON(stdout, resp)
		}
		_, _ = io.WriteString(stdout, resp.Logs)
		if !strings.HasSuffix(resp.Logs, "\n") {
			_, _ = io.WriteString(stdout, "\n")
		}
		return nil
	}

	return followPodLogs(ctx, client, status.Run.PodName, *container, *tail, *interval, stdout)
}

func followPodLogs(ctx context.Context, client *Client, podName string, container string, tail int64, interval time.Duration, stdout io.Writer) error {
	last := ""
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		resp, err := client.GetPodLogs(ctx, podName, container, tail)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		chunk := diffLogs(last, resp.Logs)
		if chunk != "" {
			_, _ = io.WriteString(stdout, chunk)
			if !strings.HasSuffix(chunk, "\n") {
				_, _ = io.WriteString(stdout, "\n")
			}
		}
		last = resp.Logs

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func diffLogs(prev string, curr string) string {
	if prev == "" {
		return curr
	}
	if strings.HasPrefix(curr, prev) {
		return curr[len(prev):]
	}
	if curr == prev {
		return ""
	}
	return "\n--- log stream reset ---\n" + curr
}
