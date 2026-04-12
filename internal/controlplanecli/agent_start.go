package controlplanecli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// agentStartResult holds the output data for a successful agent start.
type agentStartResult struct {
	SessionID string `json:"sessionId"`
	RunID     string `json:"runId"`
	Agent     string `json:"agent"`
	Phase     string `json:"phase"`
}

func runAgentStartCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := newFlagSet("kocao agent start", stderr)

	repo := fs.String("repo", "", "repository URL (required)")
	agent := fs.String("agent", "", "agent name, e.g. codex, claude, opencode, pi (required)")
	workspace := fs.String("workspace", "", "reuse existing workspace session ID")
	revision := fs.String("revision", "main", "repository revision")
	image := fs.String("image", "kocao/harness-runtime:dev", "harness runtime image")
	imagePullSecret := fs.String("image-pull-secret", "", "Kubernetes secret name for pulling the harness image")
	egressMode := fs.String("egress-mode", "", "egress mode for the harness pod: restricted (default), full")
	timeout := fs.Duration("timeout", 5*time.Minute, "timeout waiting for agent to become ready")
	output := fs.String("output", "table", "output format: table or json")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional argument: %s", fs.Arg(0))
	}
	if strings.TrimSpace(*repo) == "" {
		return fmt.Errorf("usage: kocao agent start --repo <url> --agent <name>: missing required flag --repo")
	}
	if strings.TrimSpace(*agent) == "" {
		return fmt.Errorf("usage: kocao agent start --repo <url> --agent <name>: missing required flag --agent")
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}

	sigCtx, sigCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer sigCancel()
	ctx, cancel := context.WithTimeout(sigCtx, *timeout)
	defer cancel()

	var pullSecrets []string
	if s := strings.TrimSpace(*imagePullSecret); s != "" {
		pullSecrets = []string{s}
	}

	_, _ = fmt.Fprintf(stderr, "Creating workspace session... ")
	runID, err := client.StartAgent(ctx, *workspace, *repo, *revision, *agent, *image, pullSecrets, strings.TrimSpace(*egressMode))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "failed")
		return fmt.Errorf("start agent: %w", err)
	}
	_, _ = fmt.Fprintln(stderr, "done")

	_, _ = fmt.Fprintln(stderr, "Starting harness run... done")
	_, _ = fmt.Fprintf(stderr, "Waiting for agent session... ")

	session, err := pollAgentSession(ctx, client, runID, 2*time.Second)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "timeout")
		printLastKnownAgentState(stderr, client, runID)
		return err
	}
	_, _ = fmt.Fprintln(stderr, "ready")

	result := agentStartResult{
		SessionID: session.SessionID,
		RunID:     session.RunID,
		Agent:     session.Agent,
		Phase:     session.Phase,
	}
	if format == "json" {
		return writeJSON(stdout, result)
	}

	_, _ = fmt.Fprintf(stdout, "Session ID:  %s\n", result.SessionID)
	_, _ = fmt.Fprintf(stdout, "Run ID:      %s\n", result.RunID)
	_, _ = fmt.Fprintf(stdout, "Agent:       %s\n", result.Agent)
	_, _ = fmt.Fprintf(stdout, "Phase:       %s\n", result.Phase)
	return nil
}

func printLastKnownAgentState(stderr io.Writer, client *Client, runID string) {
	lastSession, err := client.GetAgentSession(context.Background(), runID)
	if err != nil || lastSession == nil {
		return
	}
	_, _ = fmt.Fprintf(stderr, "Last known state: phase=%s sessionId=%s\n", lastSession.Phase, lastSession.SessionID)
}

// pollAgentSession polls until the agent session reaches "Ready" phase or the
// context is cancelled. On each iteration it first tries to initialize the
// session (CreateAgentSession) which is idempotent — the API will return the
// existing session if one is already active. If initialization fails because
// the harness pod is not ready (502) or the session spec hasn't propagated
// yet (404), it retries after the given interval.
func pollAgentSession(ctx context.Context, client *Client, runID string, interval time.Duration) (*AgentSession, error) {
	for {
		// Try to ensure the session is initialized. This is safe to call
		// repeatedly — the API returns the existing session if already created.
		session, err := client.CreateAgentSession(ctx, runID)
		if err == nil && strings.EqualFold(strings.TrimSpace(session.Phase), "ready") {
			return session, nil
		}
		// If create succeeded but phase isn't ready yet, try GET for fresh status.
		if err == nil {
			session, err = client.GetAgentSession(ctx, runID)
			if err == nil && strings.EqualFold(strings.TrimSpace(session.Phase), "ready") {
				return session, nil
			}
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("timed out waiting for agent session to become ready")
		}
		// 502 (pod not ready) and 404 (session not configured yet) are retryable.
		if err != nil && isNonRetryableError(err) && !isRetryableAgentSessionError(err) {
			return nil, fmt.Errorf("agent session lookup failed: %w", err)
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("timed out waiting for agent session to become ready")
		case <-timer.C:
		}
	}
}

// isRetryableAgentSessionError returns true for errors that indicate the
// agent session is not ready yet but may become ready (pod starting, etc).
func isRetryableAgentSessionError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusBadGateway || apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// isNonRetryableError returns true for HTTP errors that should not be retried
// (authentication, authorization, not-found, and server errors).
func isNonRetryableError(err error) bool {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.StatusCode == http.StatusUnauthorized,
			apiErr.StatusCode == http.StatusForbidden,
			apiErr.StatusCode == http.StatusNotFound,
			apiErr.StatusCode >= http.StatusInternalServerError:
			return true
		}
	}
	return false
}
