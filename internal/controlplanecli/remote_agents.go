package controlplanecli

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"text/tabwriter"
)

func runRemoteAgentsCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeRemoteAgentsUsage(stdout)
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "agents", "agent":
		return runRemoteAgentAgentsCommand(args[1:], cfg, stdout, stderr)
	case "tasks", "task":
		return runRemoteAgentTasksCommand(args[1:], cfg, stdout, stderr)
	case "help", "-h", "--help":
		writeRemoteAgentsUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown remote-agents subcommand %q", sub)
	}
}

func runRemoteAgentAgentsCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeRemoteAgentAgentsUsage(stdout)
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "ls", "list":
		return runRemoteAgentListCommand(args[1:], cfg, stdout, stderr)
	case "get", "inspect":
		return runRemoteAgentGetCommand(args[1:], cfg, stdout, stderr)
	case "help", "-h", "--help":
		writeRemoteAgentAgentsUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown remote-agents agents subcommand %q", sub)
	}
}

func runRemoteAgentTasksCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeRemoteAgentTasksUsage(stdout)
		return nil
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "ls", "list":
		return runRemoteAgentTaskListCommand(args[1:], cfg, stdout, stderr)
	case "get", "inspect":
		return runRemoteAgentTaskGetCommand(args[1:], cfg, stdout, stderr)
	case "dispatch":
		return runRemoteAgentTaskDispatchCommand(args[1:], cfg, stdout, stderr)
	case "cancel":
		return runRemoteAgentTaskCancelCommand(args[1:], cfg, stdout, stderr)
	case "transcript":
		return runRemoteAgentTaskTranscriptCommand(args[1:], cfg, stdout, stderr)
	case "artifacts":
		return runRemoteAgentTaskArtifactsCommand(args[1:], cfg, stdout, stderr)
	case "help", "-h", "--help":
		writeRemoteAgentTasksUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown remote-agents tasks subcommand %q", sub)
	}
}

func runRemoteAgentListCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := newFlagSet("kocao remote-agents agents list", stderr)
	pool := fs.String("pool", "", "filter by pool name")
	workspace := fs.String("workspace", "", "filter by workspace session ID")
	availability := fs.String("availability", "", "filter by availability: idle, busy, offline")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
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

	agents, err := client.ListRemoteAgents(ctx)
	if err != nil {
		return err
	}
	agents = filterRemoteAgents(agents, remoteAgentFilter{
		Pool:         *pool,
		Workspace:    *workspace,
		Availability: *availability,
	})
	sortRemoteAgents(agents)

	if format == "json" {
		return writeJSON(stdout, agents)
	}
	if len(agents) == 0 {
		_, _ = fmt.Fprintln(stdout, "no remote agents found")
		return nil
	}
	return writeRemoteAgentsTable(stdout, agents)
}

func runRemoteAgentGetCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao remote-agents agents get <agent-id-or-name> [--pool NAME] [--workspace ID] [--output table|json]")
	}
	ref := strings.TrimSpace(args[0])
	if ref == "" || strings.HasPrefix(ref, "-") {
		return fmt.Errorf("usage: kocao remote-agents agents get <agent-id-or-name> [--pool NAME] [--workspace ID] [--output table|json]")
	}

	fs := newFlagSet("kocao remote-agents agents get", stderr)
	pool := fs.String("pool", "", "disambiguate by pool name")
	workspace := fs.String("workspace", "", "disambiguate by workspace session ID")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	agent, err := resolveRemoteAgent(context.Background(), client, ref, *pool, *workspace)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, agent)
	}
	return writeRemoteAgentSummary(stdout, agent)
}

func runRemoteAgentTaskListCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := newFlagSet("kocao remote-agents tasks list", stderr)
	agent := fs.String("agent", "", "filter by agent name")
	pool := fs.String("pool", "", "filter by pool name")
	state := fs.String("state", "", "filter by state (comma-separated)")
	active := fs.Bool("active", false, "show only assigned or running tasks")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	tasks, err := client.ListRemoteAgentTasks(context.Background())
	if err != nil {
		return err
	}
	tasks = filterRemoteAgentTasks(tasks, remoteAgentTaskFilter{
		Agent:  *agent,
		Pool:   *pool,
		States: parseCSVSet(*state),
		Active: *active,
	})
	sortRemoteAgentTasks(tasks)

	if format == "json" {
		return writeJSON(stdout, tasks)
	}
	if len(tasks) == 0 {
		_, _ = fmt.Fprintln(stdout, "no remote agent tasks found")
		return nil
	}
	return writeRemoteAgentTasksTable(stdout, tasks)
}

func runRemoteAgentTaskGetCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	taskID, flagArgs, err := parseRequiredRemoteAgentTaskID("get", args)
	if err != nil {
		return err
	}
	fs := newFlagSet("kocao remote-agents tasks get", stderr)
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	task, err := client.GetRemoteAgentTask(context.Background(), taskID)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, task)
	}
	return writeRemoteAgentTaskSummary(stdout, task)
}

func runRemoteAgentTaskDispatchCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	fs := newFlagSet("kocao remote-agents tasks dispatch", stderr)
	agentID := fs.String("agent-id", "", "dispatch directly to a durable agent ID")
	agentName := fs.String("agent", "", "dispatch to a named agent (preferred)")
	pool := fs.String("pool", "", "disambiguate named agent by pool name")
	workspace := fs.String("workspace", "", "disambiguate named agent by workspace session ID")
	prompt := fs.String("prompt", "", "task prompt text")
	promptFile := fs.String("prompt-file", "", "read task prompt from file")
	timeoutSeconds := fs.Int("timeout-seconds", 0, "task timeout in seconds")
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	if strings.TrimSpace(*agentID) == "" && strings.TrimSpace(*agentName) == "" {
		return fmt.Errorf("must specify --agent <name> or --agent-id <id>")
	}
	if strings.TrimSpace(*agentID) != "" && strings.TrimSpace(*agentName) != "" {
		return fmt.Errorf("--agent and --agent-id are mutually exclusive")
	}
	if strings.TrimSpace(*prompt) == "" && strings.TrimSpace(*promptFile) == "" {
		return fmt.Errorf("must specify --prompt <text> or --prompt-file <path>")
	}
	if strings.TrimSpace(*prompt) != "" && strings.TrimSpace(*promptFile) != "" {
		return fmt.Errorf("--prompt and --prompt-file are mutually exclusive")
	}
	if *timeoutSeconds < 0 {
		return fmt.Errorf("--timeout-seconds must be >= 0")
	}

	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}

	promptText := strings.TrimSpace(*prompt)
	if strings.TrimSpace(*promptFile) != "" {
		b, err := os.ReadFile(strings.TrimSpace(*promptFile))
		if err != nil {
			return fmt.Errorf("read prompt file: %w", err)
		}
		promptText = strings.TrimSpace(string(b))
	}
	if promptText == "" {
		return fmt.Errorf("task prompt cannot be empty")
	}

	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	target, err := resolveRemoteAgentTaskTarget(context.Background(), client, strings.TrimSpace(*agentName), strings.TrimSpace(*agentID), strings.TrimSpace(*pool), strings.TrimSpace(*workspace))
	if err != nil {
		return err
	}
	task, err := client.CreateRemoteAgentTask(context.Background(), RemoteAgentTaskCreateRequest{
		Target:         target,
		Prompt:         promptText,
		TimeoutSeconds: int32(*timeoutSeconds),
	})
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, task)
	}
	_, _ = fmt.Fprintf(stdout, "dispatched task %s to %s\n", task.ID, remoteAgentTaskDisplayAgent(task))
	return writeRemoteAgentTaskSummary(stdout, task)
}

func runRemoteAgentTaskCancelCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	taskID, flagArgs, err := parseRequiredRemoteAgentTaskID("cancel", args)
	if err != nil {
		return err
	}
	fs := newFlagSet("kocao remote-agents tasks cancel", stderr)
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	task, err := client.CancelRemoteAgentTask(context.Background(), taskID)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, task)
	}
	_, _ = fmt.Fprintf(stdout, "cancelled task %s\n", task.ID)
	return writeRemoteAgentTaskSummary(stdout, task)
}

func runRemoteAgentTaskTranscriptCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	taskID, flagArgs, err := parseRequiredRemoteAgentTaskID("transcript", args)
	if err != nil {
		return err
	}
	fs := newFlagSet("kocao remote-agents tasks transcript", stderr)
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	transcript, err := client.GetRemoteAgentTaskTranscript(context.Background(), taskID)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, transcript)
	}
	if len(transcript.Transcript) == 0 {
		_, _ = fmt.Fprintf(stdout, "task %s has no transcript entries\n", taskID)
		return nil
	}
	return writeRemoteAgentTranscriptTable(stdout, transcript.Transcript)
}

func runRemoteAgentTaskArtifactsCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	taskID, flagArgs, err := parseRequiredRemoteAgentTaskID("artifacts", args)
	if err != nil {
		return err
	}
	fs := newFlagSet("kocao remote-agents tasks artifacts", stderr)
	output := fs.String("output", "table", "output format: table or json")
	if err := fs.Parse(flagArgs); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	format, err := parseAgentOutputFormat(*output, "table", "json")
	if err != nil {
		return err
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	artifacts, err := client.GetRemoteAgentTaskArtifacts(context.Background(), taskID)
	if err != nil {
		return err
	}
	if format == "json" {
		return writeJSON(stdout, artifacts)
	}
	if len(artifacts.InputArtifacts) == 0 && len(artifacts.OutputArtifacts) == 0 {
		_, _ = fmt.Fprintf(stdout, "task %s has no artifacts\n", taskID)
		return nil
	}
	return writeRemoteAgentArtifactsTable(stdout, artifacts.InputArtifacts, artifacts.OutputArtifacts)
}

func writeRemoteAgentsUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao remote-agents")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents agents <subcommand>")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks <subcommand>")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Subcommands:")
	_, _ = fmt.Fprintln(w, "  agents   Inspect named remote agents")
	_, _ = fmt.Fprintln(w, "  tasks    Dispatch and inspect remote-agent tasks")
}

func writeRemoteAgentAgentsUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao remote-agents agents")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents agents list [--pool NAME] [--workspace ID] [--availability idle|busy|offline] [--output table|json]")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents agents get <agent-id-or-name> [--pool NAME] [--workspace ID] [--output table|json]")
}

func writeRemoteAgentTasksUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao remote-agents tasks")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks list [--agent NAME] [--pool NAME] [--state assigned,running] [--active] [--output table|json]")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks get <task-id> [--output table|json]")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks dispatch --agent <name>|--agent-id <id> --prompt <text>|--prompt-file <path> [--pool NAME] [--workspace ID] [--timeout-seconds N] [--output table|json]")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks cancel <task-id> [--output table|json]")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks transcript <task-id> [--output table|json]")
	_, _ = fmt.Fprintln(w, "  kocao remote-agents tasks artifacts <task-id> [--output table|json]")
}

type remoteAgentFilter struct {
	Pool         string
	Workspace    string
	Availability string
}

func filterRemoteAgents(agents []RemoteAgent, filter remoteAgentFilter) []RemoteAgent {
	if len(agents) == 0 {
		return nil
	}
	pool := strings.ToLower(strings.TrimSpace(filter.Pool))
	workspace := strings.TrimSpace(filter.Workspace)
	availability := strings.ToLower(strings.TrimSpace(filter.Availability))
	out := make([]RemoteAgent, 0, len(agents))
	for _, agent := range agents {
		if pool != "" && strings.ToLower(strings.TrimSpace(agent.PoolName)) != pool {
			continue
		}
		if workspace != "" && strings.TrimSpace(agent.WorkspaceSessionID) != workspace {
			continue
		}
		if availability != "" && strings.ToLower(strings.TrimSpace(agent.Availability)) != availability {
			continue
		}
		out = append(out, agent)
	}
	return out
}

type remoteAgentTaskFilter struct {
	Agent  string
	Pool   string
	States map[string]struct{}
	Active bool
}

func filterRemoteAgentTasks(tasks []RemoteAgentTask, filter remoteAgentTaskFilter) []RemoteAgentTask {
	if len(tasks) == 0 {
		return nil
	}
	agent := strings.ToLower(strings.TrimSpace(filter.Agent))
	pool := strings.ToLower(strings.TrimSpace(filter.Pool))
	out := make([]RemoteAgentTask, 0, len(tasks))
	for _, task := range tasks {
		if agent != "" && strings.ToLower(strings.TrimSpace(task.AgentName)) != agent {
			continue
		}
		if pool != "" && strings.ToLower(strings.TrimSpace(task.PoolName)) != pool {
			continue
		}
		state := strings.ToLower(strings.TrimSpace(task.State))
		if filter.Active && state != "assigned" && state != "running" {
			continue
		}
		if len(filter.States) != 0 {
			if _, ok := filter.States[state]; !ok {
				continue
			}
		}
		out = append(out, task)
	}
	return out
}

func parseCSVSet(raw string) map[string]struct{} {
	values := map[string]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.ToLower(strings.TrimSpace(item))
		if item == "" {
			continue
		}
		values[item] = struct{}{}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func sortRemoteAgents(agents []RemoteAgent) {
	sort.Slice(agents, func(i, j int) bool {
		if strings.TrimSpace(agents[i].Name) == strings.TrimSpace(agents[j].Name) {
			return agents[i].ID < agents[j].ID
		}
		return strings.TrimSpace(agents[i].Name) < strings.TrimSpace(agents[j].Name)
	})
}

func sortRemoteAgentTasks(tasks []RemoteAgentTask) {
	sort.Slice(tasks, func(i, j int) bool {
		left := remoteAgentTaskSortKey(tasks[i])
		right := remoteAgentTaskSortKey(tasks[j])
		if left == right {
			return tasks[i].ID > tasks[j].ID
		}
		return left > right
	})
}

func remoteAgentTaskSortKey(task RemoteAgentTask) string {
	for _, candidate := range []string{task.LastTransitionAt, task.CompletedAt, task.CancelledAt, task.StartedAt, task.AssignedAt, task.CreatedAt} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return ""
}

func resolveRemoteAgent(ctx context.Context, client *Client, ref string, pool string, workspace string) (RemoteAgent, error) {
	agents, err := client.ListRemoteAgents(ctx)
	if err != nil {
		return RemoteAgent{}, err
	}
	ref = strings.TrimSpace(ref)
	pool = strings.ToLower(strings.TrimSpace(pool))
	workspace = strings.TrimSpace(workspace)
	var matches []RemoteAgent
	for _, agent := range agents {
		if strings.TrimSpace(agent.ID) != ref && !strings.EqualFold(strings.TrimSpace(agent.Name), ref) {
			continue
		}
		if pool != "" && strings.ToLower(strings.TrimSpace(agent.PoolName)) != pool {
			continue
		}
		if workspace != "" && strings.TrimSpace(agent.WorkspaceSessionID) != workspace {
			continue
		}
		matches = append(matches, agent)
	}
	if len(matches) == 0 {
		return RemoteAgent{}, fmt.Errorf("remote agent %q not found", ref)
	}
	if len(matches) > 1 {
		return RemoteAgent{}, fmt.Errorf("remote agent %q is ambiguous; specify --pool or --workspace", ref)
	}
	return matches[0], nil
}

func resolveRemoteAgentTaskTarget(ctx context.Context, client *Client, agentName string, agentID string, pool string, workspace string) (RemoteAgentTaskTarget, error) {
	if agentID != "" {
		return RemoteAgentTaskTarget{AgentID: agentID}, nil
	}
	agent, err := resolveRemoteAgent(ctx, client, agentName, pool, workspace)
	if err != nil {
		return RemoteAgentTaskTarget{}, err
	}
	return RemoteAgentTaskTarget{AgentID: agent.ID}, nil
}

func parseRequiredRemoteAgentTaskID(command string, args []string) (string, []string, error) {
	usage := fmt.Sprintf("usage: kocao remote-agents tasks %s <task-id> [--output table|json]", command)
	if len(args) == 0 || strings.HasPrefix(strings.TrimSpace(args[0]), "-") {
		return "", nil, fmt.Errorf("%s", usage)
	}
	taskID := strings.TrimSpace(args[0])
	if taskID == "" {
		return "", nil, fmt.Errorf("%s", usage)
	}
	return taskID, args[1:], nil
}

func writeRemoteAgentsTable(w io.Writer, agents []RemoteAgent) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "AGENT\tPOOL\tAVAILABILITY\tCURRENT TASK\tWORKSPACE\tLAST ACTIVITY\tAGENT ID"); err != nil {
		return err
	}
	for _, agent := range agents {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			valueOrDash(agent.Name),
			valueOrDash(agent.PoolName),
			valueOrDash(agent.Availability),
			valueOrDash(agent.CurrentTaskID),
			valueOrDash(agent.WorkspaceSessionID),
			valueOrDash(agent.LastActivityAt),
			valueOrDash(agent.ID),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeRemoteAgentSummary(w io.Writer, agent RemoteAgent) error {
	lines := []struct {
		label string
		value string
	}{
		{"Agent", valueOrDash(agent.Name)},
		{"Agent ID", valueOrDash(agent.ID)},
		{"Pool", valueOrDash(agent.PoolName)},
		{"Availability", valueOrDash(agent.Availability)},
		{"Current Task", valueOrDash(agent.CurrentTaskID)},
		{"Workspace", valueOrDash(agent.WorkspaceSessionID)},
		{"Runtime", valueOrDash(string(agent.Runtime))},
		{"Kind", valueOrDash(string(agent.Agent))},
		{"Last Activity", valueOrDash(agent.LastActivityAt)},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%-14s %s\n", line.label+":", line.value); err != nil {
			return err
		}
	}
	if agent.CurrentSession != nil {
		if _, err := fmt.Fprintf(w, "%-14s %s\n", "Session ID:", valueOrDash(agent.CurrentSession.SessionID)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%-14s %s\n", "Harness Run:", valueOrDash(agent.CurrentSession.HarnessRunID)); err != nil {
			return err
		}
	}
	return nil
}

func writeRemoteAgentTasksTable(w io.Writer, tasks []RemoteAgentTask) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TASK ID\tSTATE\tAGENT\tPOOL\tATTEMPT\tLAST TRANSITION"); err != nil {
		return err
	}
	for _, task := range tasks {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%d\t%s\n",
			valueOrDash(task.ID),
			valueOrDash(task.State),
			valueOrDash(task.AgentName),
			valueOrDash(task.PoolName),
			task.Attempt,
			valueOrDash(remoteAgentTaskSortKey(task)),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeRemoteAgentTaskSummary(w io.Writer, task RemoteAgentTask) error {
	lines := []struct {
		label string
		value string
	}{
		{"Task ID", valueOrDash(task.ID)},
		{"State", valueOrDash(task.State)},
		{"Agent", valueOrDash(task.AgentName)},
		{"Agent ID", valueOrDash(task.AgentID)},
		{"Pool", valueOrDash(task.PoolName)},
		{"Workspace", valueOrDash(task.WorkspaceSessionID)},
		{"Attempt", fmt.Sprintf("%d", task.Attempt)},
		{"Retries", fmt.Sprintf("%d", task.RetryCount)},
		{"Timeout", fmt.Sprintf("%ds", task.TimeoutSeconds)},
		{"Assigned", valueOrDash(task.AssignedAt)},
		{"Started", valueOrDash(task.StartedAt)},
		{"Completed", valueOrDash(task.CompletedAt)},
		{"Cancelled", valueOrDash(task.CancelledAt)},
	}
	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%-14s %s\n", line.label+":", line.value); err != nil {
			return err
		}
	}
	if strings.TrimSpace(task.Prompt) != "" {
		if _, err := fmt.Fprintf(w, "%-14s %s\n", "Prompt:", task.Prompt); err != nil {
			return err
		}
	}
	if task.Result != nil {
		if _, err := fmt.Fprintf(w, "%-14s %s\n", "Outcome:", valueOrDash(task.Result.Outcome)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%-14s %s\n", "Summary:", valueOrDash(task.Result.Summary)); err != nil {
			return err
		}
	}
	if task.CurrentSession != nil {
		if _, err := fmt.Fprintf(w, "%-14s %s\n", "Session ID:", valueOrDash(task.CurrentSession.SessionID)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w, "%-14s %d\n", "Transcript:", len(task.Transcript)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "%-14s %d\n", "Artifacts:", len(task.InputArtifacts)+len(task.OutputArtifacts)); err != nil {
		return err
	}
	return nil
}

func writeRemoteAgentTranscriptTable(w io.Writer, transcript []RemoteAgentTranscriptEntry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SEQ\tAT\tROLE\tKIND\tEVENT REF\tTEXT"); err != nil {
		return err
	}
	for _, entry := range transcript {
		if _, err := fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\n",
			entry.Sequence,
			valueOrDash(entry.At),
			valueOrDash(entry.Role),
			valueOrDash(entry.Kind),
			valueOrDash(entry.EventRef),
			valueOrDash(entry.Text),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeRemoteAgentArtifactsTable(w io.Writer, inputs []RemoteAgentArtifactRef, outputs []RemoteAgentArtifactRef) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SCOPE\tNAME\tKIND\tMEDIA TYPE\tSIZE\tPATH/URI\tCREATED"); err != nil {
		return err
	}
	for _, artifact := range inputs {
		if err := writeRemoteAgentArtifactRow(tw, "input", artifact); err != nil {
			return err
		}
	}
	for _, artifact := range outputs {
		if err := writeRemoteAgentArtifactRow(tw, "output", artifact); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func writeRemoteAgentArtifactRow(w io.Writer, scope string, artifact RemoteAgentArtifactRef) error {
	location := strings.TrimSpace(artifact.Path)
	if location == "" {
		location = strings.TrimSpace(artifact.URI)
	}
	_, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
		scope,
		valueOrDash(artifact.Name),
		valueOrDash(artifact.Kind),
		valueOrDash(artifact.MediaType),
		artifact.SizeBytes,
		valueOrDash(location),
		valueOrDash(artifact.CreatedAt),
	)
	return err
}

func remoteAgentTaskDisplayAgent(task RemoteAgentTask) string {
	if strings.TrimSpace(task.AgentName) != "" {
		if strings.TrimSpace(task.PoolName) != "" {
			return task.AgentName + " (pool " + task.PoolName + ")"
		}
		return task.AgentName
	}
	return valueOrDash(task.AgentID)
}
