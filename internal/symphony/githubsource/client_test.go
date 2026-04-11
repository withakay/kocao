package githubsource

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

func TestLoadProjectNormalizesCandidatesAndSkips(t *testing.T) {
	t.Helper()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if got := r.Header.Get("Authorization"); got != "Bearer github-token" {
			t.Fatalf("authorization header = %q", got)
		}
		writeGraphQLResponse(t, w, map[string]any{
			"data": map[string]any{
				"organization": map[string]any{
					"projectV2": map[string]any{
						"id":    "PVT_project_1",
						"title": "Symphony",
						"items": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
							"nodes": []map[string]any{
								issueItem("PVT_item_1", false, "Todo", "ISSUE_1", 101, "withakay", "kocao", false, "OPEN"),
								issueItem("PVT_item_2", false, "Done", "ISSUE_2", 102, "withakay", "kocao", false, "OPEN"),
								issueItem("PVT_item_3", false, "Todo", "ISSUE_3", 103, "someone", "else", false, "OPEN"),
								pullRequestItem("PVT_item_4", "Todo", "withakay", "kocao"),
								archivedIssueItem("PVT_item_5", "Todo"),
								issueItem("PVT_item_6", false, "Todo", "ISSUE_6", 104, "withakay", "kocao", true, "CLOSED"),
								unsupportedItem("PVT_item_7", "Todo", "DraftIssue"),
							},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := NewClient("github-token", Options{APIURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	snapshot, err := client.LoadProject(context.Background(), LoadOptions{
		Project:        operatorv1alpha1.GitHubProjectRef{Owner: "withakay", Number: 7},
		FieldName:      "Status",
		ActiveStates:   []string{"Todo", "In Progress"},
		TerminalStates: []string{"Done"},
		Repositories: []operatorv1alpha1.SymphonyProjectRepositorySpec{
			{Owner: "withakay", Name: "kocao"},
		},
	})
	if err != nil {
		t.Fatalf("LoadProject error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if snapshot.ProjectID != "PVT_project_1" {
		t.Fatalf("ProjectID = %q", snapshot.ProjectID)
	}
	if len(snapshot.Candidates) != 1 {
		t.Fatalf("candidates len = %d, want 1", len(snapshot.Candidates))
	}
	candidate := snapshot.Candidates[0]
	if candidate.ItemID != "PVT_item_1" {
		t.Fatalf("candidate item = %q", candidate.ItemID)
	}
	if candidate.Issue.Repository != "withakay/kocao" {
		t.Fatalf("candidate repository = %q", candidate.Issue.Repository)
	}
	if candidate.Issue.Number != 101 {
		t.Fatalf("candidate issue number = %d", candidate.Issue.Number)
	}
	if len(candidate.Issue.Labels) != 2 || candidate.Issue.Labels[0] != "bug" || candidate.Issue.Labels[1] != "queued" {
		t.Fatalf("candidate labels = %#v", candidate.Issue.Labels)
	}
	if candidate.Issue.CreatedAt.Format(time.RFC3339) != "2026-03-09T10:00:00Z" {
		t.Fatalf("candidate createdAt = %s", candidate.Issue.CreatedAt.Format(time.RFC3339))
	}

	if len(snapshot.Skipped) != 6 {
		t.Fatalf("skipped len = %d, want 6", len(snapshot.Skipped))
	}
	assertSkipReason(t, snapshot.Skipped, "PVT_item_2", SkipReasonTerminalState)
	assertSkipReason(t, snapshot.Skipped, "PVT_item_3", SkipReasonUnsupportedRepository)
	assertSkipReason(t, snapshot.Skipped, "PVT_item_4", SkipReasonPullRequest)
	assertSkipReason(t, snapshot.Skipped, "PVT_item_5", SkipReasonArchived)
	assertSkipReason(t, snapshot.Skipped, "PVT_item_6", SkipReasonIssueClosed)
	assertSkipReason(t, snapshot.Skipped, "PVT_item_7", SkipReasonUnsupportedItem)
	if len(snapshot.UnsupportedRepositories) != 1 || snapshot.UnsupportedRepositories[0] != "someone/else" {
		t.Fatalf("unsupported repositories = %#v", snapshot.UnsupportedRepositories)
	}
}

func TestLoadProjectPaginatesUserProjects(t *testing.T) {
	t.Helper()
	var cursors []any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		cursors = append(cursors, req.Variables["cursor"])
		if req.Variables["cursor"] == nil || req.Variables["cursor"] == "" {
			writeGraphQLResponse(t, w, map[string]any{
				"data": map[string]any{
					"user": map[string]any{
						"projectV2": map[string]any{
							"id":    "PVT_project_2",
							"title": "Paged Symphony",
							"items": map[string]any{
								"pageInfo": map[string]any{"hasNextPage": true, "endCursor": "cursor-1"},
								"nodes": []map[string]any{
									issueItem("PVT_item_1", false, "Todo", "ISSUE_1", 1, "withakay", "kocao", false, "OPEN"),
								},
							},
						},
					},
				},
				"errors": []map[string]any{{"message": "Could not resolve to an Organization with the login of 'withakay'."}},
			})
			return
		}
		writeGraphQLResponse(t, w, map[string]any{
			"data": map[string]any{
				"user": map[string]any{
					"projectV2": map[string]any{
						"id":    "PVT_project_2",
						"title": "Paged Symphony",
						"items": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
							"nodes": []map[string]any{
								issueItem("PVT_item_2", false, "Todo", "ISSUE_2", 2, "withakay", "kocao", false, "OPEN"),
							},
						},
					},
				},
			},
			"errors": []map[string]any{{"message": "Could not resolve to an Organization with the login of 'withakay'."}},
		})
	}))
	defer srv.Close()

	client, err := NewClient("github-token", Options{APIURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	snapshot, err := client.LoadProject(context.Background(), LoadOptions{
		Project:        operatorv1alpha1.GitHubProjectRef{Owner: "withakay", Number: 8},
		ActiveStates:   []string{"Todo"},
		TerminalStates: []string{"Done"},
		Repositories: []operatorv1alpha1.SymphonyProjectRepositorySpec{
			{Owner: "withakay", Name: "kocao"},
		},
	})
	if err != nil {
		t.Fatalf("LoadProject error = %v", err)
	}
	if len(cursors) != 2 {
		t.Fatalf("cursors len = %d, want 2", len(cursors))
	}
	if len(snapshot.Candidates) != 2 {
		t.Fatalf("candidates len = %d, want 2", len(snapshot.Candidates))
	}
	if snapshot.Candidates[1].Issue.Number != 2 {
		t.Fatalf("second candidate issue number = %d", snapshot.Candidates[1].Issue.Number)
	}
}

func TestLoadProjectIgnoresUserLookupErrorForOrganizationProjects(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeGraphQLResponse(t, w, map[string]any{
			"data": map[string]any{
				"organization": map[string]any{
					"projectV2": map[string]any{
						"id":    "PVT_project_org",
						"title": "Org Symphony",
						"items": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
							"nodes": []map[string]any{
								issueItem("PVT_item_org", false, "Todo", "ISSUE_ORG", 88, "acme", "platform", false, "OPEN"),
							},
						},
					},
				},
			},
			"errors": []map[string]any{{"message": "Could not resolve to a User with the login of 'acme'."}},
		})
	}))
	defer srv.Close()

	client, err := NewClient("github-token", Options{APIURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	snapshot, err := client.LoadProject(context.Background(), LoadOptions{
		Project:        operatorv1alpha1.GitHubProjectRef{Owner: "acme", Number: 9},
		ActiveStates:   []string{"Todo"},
		TerminalStates: []string{"Done"},
		Repositories:   []operatorv1alpha1.SymphonyProjectRepositorySpec{{Owner: "acme", Name: "platform"}},
	})
	if err != nil {
		t.Fatalf("LoadProject error = %v", err)
	}
	if len(snapshot.Candidates) != 1 || snapshot.Candidates[0].Issue.Repository != "acme/platform" {
		t.Fatalf("unexpected candidates = %#v", snapshot.Candidates)
	}
}

func TestLoadProjectReturnsNonOwnerGraphQLErrors(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeGraphQLResponse(t, w, map[string]any{
			"data": map[string]any{
				"user": map[string]any{
					"projectV2": map[string]any{
						"id":    "PVT_project_2",
						"title": "Paged Symphony",
						"items": map[string]any{
							"pageInfo": map[string]any{"hasNextPage": false, "endCursor": ""},
							"nodes":    []map[string]any{},
						},
					},
				},
			},
			"errors": []map[string]any{{"message": "Resource not accessible by personal access token"}},
		})
	}))
	defer srv.Close()

	client, err := NewClient("github-token", Options{APIURL: srv.URL, HTTPClient: srv.Client()})
	if err != nil {
		t.Fatalf("NewClient error = %v", err)
	}

	_, err = client.LoadProject(context.Background(), LoadOptions{
		Project:        operatorv1alpha1.GitHubProjectRef{Owner: "withakay", Number: 8},
		ActiveStates:   []string{"Todo"},
		TerminalStates: []string{"Done"},
		Repositories:   []operatorv1alpha1.SymphonyProjectRepositorySpec{{Owner: "withakay", Name: "kocao"}},
	})
	if err == nil {
		t.Fatal("expected graphql accessibility error")
	}
	if got := err.Error(); !strings.Contains(got, "github graphql error") || !strings.Contains(got, "Resource not accessible by personal access token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func assertSkipReason(t *testing.T, skipped []SkippedItem, itemID, want string) {
	t.Helper()
	for _, item := range skipped {
		if item.ItemID == itemID {
			if item.Reason != want {
				t.Fatalf("skip reason for %s = %q, want %q", itemID, item.Reason, want)
			}
			return
		}
	}
	t.Fatalf("skip item %s not found", itemID)
}

func issueItem(itemID string, archived bool, status, issueID string, number int64, owner, repo string, closed bool, state string) map[string]any {
	return map[string]any{
		"id":         itemID,
		"isArchived": archived,
		"fieldValueByName": map[string]any{
			"__typename": "ProjectV2ItemFieldSingleSelectValue",
			"name":       status,
		},
		"content": map[string]any{
			"__typename": "Issue",
			"id":         issueID,
			"number":     number,
			"title":      "Issue title",
			"body":       "Issue body",
			"url":        "https://github.com/" + owner + "/" + repo + "/issues/" + jsonNumber(number),
			"state":      state,
			"closed":     closed,
			"createdAt":  "2026-03-09T10:00:00Z",
			"updatedAt":  "2026-03-09T11:00:00Z",
			"repository": map[string]any{"name": repo, "owner": map[string]any{"login": owner}},
			"labels":     map[string]any{"nodes": []map[string]any{{"name": "bug"}, {"name": "queued"}}},
		},
	}
}

func archivedIssueItem(itemID, status string) map[string]any {
	item := issueItem(itemID, true, status, "ISSUE_ARCHIVED", 999, "withakay", "kocao", false, "OPEN")
	item["content"] = nil
	return item
}

func pullRequestItem(itemID, status, owner, repo string) map[string]any {
	return map[string]any{
		"id":         itemID,
		"isArchived": false,
		"fieldValueByName": map[string]any{
			"__typename": "ProjectV2ItemFieldSingleSelectValue",
			"name":       status,
		},
		"content": map[string]any{
			"__typename": "PullRequest",
			"id":         "PR_1",
			"number":     44,
			"title":      "PR title",
			"url":        "https://github.com/" + owner + "/" + repo + "/pull/44",
			"repository": map[string]any{"name": repo, "owner": map[string]any{"login": owner}},
		},
	}
}

func unsupportedItem(itemID, status, typeName string) map[string]any {
	return map[string]any{
		"id":         itemID,
		"isArchived": false,
		"fieldValueByName": map[string]any{
			"__typename": "ProjectV2ItemFieldSingleSelectValue",
			"name":       status,
		},
		"content": map[string]any{
			"__typename": typeName,
		},
	}
}

func writeGraphQLResponse(t *testing.T, w http.ResponseWriter, payload map[string]any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

func jsonNumber(v int64) string {
	return strconv.FormatInt(v, 10)
}
