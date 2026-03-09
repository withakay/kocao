package controlplanecli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

func runSymphonyCommand(args []string, cfg Config, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		writeSymphonyUsage(stdout)
		return nil
	}

	ctx := context.Background()
	sub := strings.ToLower(strings.TrimSpace(args[0]))
	switch sub {
	case "ls", "list":
		return runSymphonyListCommand(ctx, cfg, args[1:], stdout, stderr)
	case "get", "inspect":
		return runSymphonyGetCommand(ctx, cfg, args[1:], stdout, stderr)
	case "create":
		return runSymphonyCreateCommand(ctx, cfg, args[1:], stdout, stderr)
	case "pause":
		return runSymphonyControlCommand(ctx, cfg, args[1:], stdout, stderr, "pause")
	case "resume":
		return runSymphonyControlCommand(ctx, cfg, args[1:], stdout, stderr, "resume")
	case "refresh":
		return runSymphonyControlCommand(ctx, cfg, args[1:], stdout, stderr, "refresh")
	case "help", "-h", "--help":
		writeSymphonyUsage(stdout)
		return nil
	default:
		return fmt.Errorf("unknown symphony subcommand %q", sub)
	}
}

func runSymphonyListCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("kocao symphony ls", flag.ContinueOnError)
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
	projects, err := client.ListSymphonyProjects(ctx)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(stdout, map[string]any{"symphonyProjects": projects})
	}
	return writeSymphonyTable(stdout, projects)
}

func runSymphonyGetCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao symphony get <project-name> [--json]")
	}
	name := strings.TrimSpace(args[0])
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("usage: kocao symphony get <project-name> [--json]")
	}
	fs := flag.NewFlagSet("kocao symphony get", flag.ContinueOnError)
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
	project, err := client.GetSymphonyProject(ctx, name)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(stdout, project)
	}
	_, _ = fmt.Fprintf(stdout, "Name:         %s\n", project.Name)
	_, _ = fmt.Fprintf(stdout, "Phase:        %s\n", valueOrDash(string(project.Status.Phase)))
	_, _ = fmt.Fprintf(stdout, "Paused:       %t\n", project.Paused)
	_, _ = fmt.Fprintf(stdout, "GitHub:       %s/%d\n", project.Spec.Source.Project.Owner, project.Spec.Source.Project.Number)
	_, _ = fmt.Fprintf(stdout, "Repositories: %d\n", len(project.Spec.Repositories))
	_, _ = fmt.Fprintf(stdout, "Active:       %d\n", project.Status.RunningItems)
	_, _ = fmt.Fprintf(stdout, "Retrying:     %d\n", project.Status.RetryingItems)
	_, _ = fmt.Fprintf(stdout, "Created At:   %s\n", valueOrDash(project.CreatedAt))
	return nil
}

func runSymphonyCreateCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer) error {
	fs := flag.NewFlagSet("kocao symphony create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	filePath := fs.String("file", "", "path to symphony project JSON payload")
	jsonOut := fs.Bool("json", false, "output JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*filePath) == "" {
		return fmt.Errorf("usage: kocao symphony create --file <path> [--json]")
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", fs.Arg(0))
	}
	b, err := os.ReadFile(strings.TrimSpace(*filePath))
	if err != nil {
		return fmt.Errorf("read symphony project file: %w", err)
	}
	var req SymphonyProjectRequest
	if err := json.Unmarshal(b, &req); err != nil {
		return fmt.Errorf("decode symphony project file: %w", err)
	}
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	project, err := client.CreateSymphonyProject(ctx, req)
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(stdout, project)
	}
	_, _ = fmt.Fprintf(stdout, "created symphony project %s\n", project.Name)
	return nil
}

func runSymphonyControlCommand(ctx context.Context, cfg Config, args []string, stdout io.Writer, stderr io.Writer, action string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: kocao symphony %s <project-name> [--json]", action)
	}
	name := strings.TrimSpace(args[0])
	if name == "" || strings.HasPrefix(name, "-") {
		return fmt.Errorf("usage: kocao symphony %s <project-name> [--json]", action)
	}
	fs := flag.NewFlagSet("kocao symphony "+action, flag.ContinueOnError)
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
	var project SymphonyProject
	switch action {
	case "pause":
		project, err = client.PauseSymphonyProject(ctx, name)
	case "resume":
		project, err = client.ResumeSymphonyProject(ctx, name)
	case "refresh":
		project, err = client.RefreshSymphonyProject(ctx, name)
	}
	if err != nil {
		return err
	}
	if *jsonOut {
		return writeJSON(stdout, project)
	}
	_, _ = fmt.Fprintf(stdout, "%s symphony project %s\n", action, project.Name)
	return nil
}

func writeSymphonyUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "kocao symphony")
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  kocao symphony ls [--json]")
	_, _ = fmt.Fprintln(w, "  kocao symphony get <project-name> [--json]")
	_, _ = fmt.Fprintln(w, "  kocao symphony create --file <path> [--json]")
	_, _ = fmt.Fprintln(w, "  kocao symphony pause <project-name> [--json]")
	_, _ = fmt.Fprintln(w, "  kocao symphony resume <project-name> [--json]")
	_, _ = fmt.Fprintln(w, "  kocao symphony refresh <project-name> [--json]")
}

func writeSymphonyTable(w io.Writer, projects []SymphonyProject) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "NAME\tPHASE\tPAUSED\tPROJECT\tACTIVE\tRETRY\tNEXT SYNC"); err != nil {
		return err
	}
	for _, project := range projects {
		nextSync := "-"
		if project.Status.NextSyncTime != nil {
			nextSync = project.Status.NextSyncTime.UTC().Format("2006-01-02T15:04:05Z")
		}
		projectRef := fmt.Sprintf("%s/%d", project.Spec.Source.Project.Owner, project.Spec.Source.Project.Number)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%t\t%s\t%d\t%d\t%s\n", project.Name, valueOrDash(string(project.Status.Phase)), project.Paused, projectRef, project.Status.RunningItems, project.Status.RetryingItems, nextSync); err != nil {
			return err
		}
	}
	return tw.Flush()
}
