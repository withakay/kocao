package controlplanecli

import (
	"context"
	"sort"
	"strings"
	"time"
)

type SessionStatus struct {
	Session WorkspaceSession `json:"session"`
	Run     *HarnessRun      `json:"run,omitempty"`
}

func BuildSessionStatus(ctx context.Context, client *Client, sessionID string) (SessionStatus, error) {
	session, err := client.GetWorkspaceSession(ctx, sessionID)
	if err != nil {
		return SessionStatus{}, err
	}
	runs, err := client.ListHarnessRuns(ctx, sessionID)
	if err != nil {
		return SessionStatus{}, err
	}
	return SessionStatus{Session: session, Run: selectPreferredRun(runs)}, nil
}

func sortSessionsNewestFirst(in []WorkspaceSession) {
	sort.Slice(in, func(i, j int) bool {
		ti := parseRFC3339(in[i].CreatedAt)
		tj := parseRFC3339(in[j].CreatedAt)
		if !ti.Equal(tj) {
			return ti.After(tj)
		}
		a := in[i].DisplayName
		if strings.TrimSpace(a) == "" {
			a = in[i].ID
		}
		b := in[j].DisplayName
		if strings.TrimSpace(b) == "" {
			b = in[j].ID
		}
		return strings.ToLower(a) < strings.ToLower(b)
	})
}

func parseRFC3339(raw string) time.Time {
	v := strings.TrimSpace(raw)
	if v == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return time.Time{}
	}
	return t
}

func selectPreferredRun(runs []HarnessRun) *HarnessRun {
	if len(runs) == 0 {
		return nil
	}
	copyRuns := make([]HarnessRun, len(runs))
	copy(copyRuns, runs)
	sort.Slice(copyRuns, func(i, j int) bool {
		pi := runPhasePriority(copyRuns[i].Phase)
		pj := runPhasePriority(copyRuns[j].Phase)
		if pi != pj {
			return pi > pj
		}
		hasPodI := strings.TrimSpace(copyRuns[i].PodName) != ""
		hasPodJ := strings.TrimSpace(copyRuns[j].PodName) != ""
		if hasPodI != hasPodJ {
			return hasPodI
		}
		return copyRuns[i].ID > copyRuns[j].ID
	})
	return &copyRuns[0]
}

func runPhasePriority(phase string) int {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "running":
		return 5
	case "starting":
		return 4
	case "pending":
		return 3
	case "succeeded":
		return 2
	case "failed":
		return 1
	default:
		return 0
	}
}
