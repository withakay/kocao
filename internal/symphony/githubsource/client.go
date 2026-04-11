package githubsource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

const (
	DefaultAPIURL = "https://api.github.com/graphql"

	SkipReasonArchived              = "archived"
	SkipReasonUnsupportedItem       = "unsupported_item"
	SkipReasonPullRequest           = "pull_request"
	SkipReasonUnsupportedRepository = "unsupported_repository"
	SkipReasonTerminalState         = "terminal_state"
	SkipReasonInactiveState         = "inactive_state"
	SkipReasonIssueClosed           = "issue_closed"
)

var errProjectNotFound = fmt.Errorf("github project not found")

type Options struct {
	APIURL     string
	HTTPClient *http.Client
}

type Client struct {
	apiURL     string
	httpClient *http.Client
	token      string
}

type LoadOptions struct {
	Project        operatorv1alpha1.GitHubProjectRef
	FieldName      string
	ActiveStates   []string
	TerminalStates []string
	Repositories   []operatorv1alpha1.SymphonyProjectRepositorySpec
}

type Snapshot struct {
	ProjectID               string
	ProjectTitle            string
	ResolvedFieldName       string
	Candidates              []CandidateItem
	Skipped                 []SkippedItem
	UnsupportedRepositories []string
}

type CandidateItem struct {
	ItemID string
	Status string
	Issue  Issue
}

type SkippedItem struct {
	ItemID      string
	Repository  string
	Status      string
	Reason      string
	Message     string
	Issue       *Issue
	ObservedAt  time.Time
	WasArchived bool
}

type Issue struct {
	NodeID      string
	Repository  string
	Number      int64
	Title       string
	Body        string
	Labels      []string
	URL         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ProjectItem string
}

func NewClient(token string, opts Options) (*Client, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("github token is required")
	}
	apiURL := strings.TrimSpace(opts.APIURL)
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{apiURL: apiURL, httpClient: httpClient, token: token}, nil
}

func (c *Client) LoadProject(ctx context.Context, opts LoadOptions) (Snapshot, error) {
	if c == nil {
		return Snapshot{}, fmt.Errorf("github source client is nil")
	}
	if strings.TrimSpace(opts.Project.Owner) == "" {
		return Snapshot{}, fmt.Errorf("project owner is required")
	}
	if opts.Project.Number <= 0 {
		return Snapshot{}, fmt.Errorf("project number must be greater than zero")
	}
	fieldName := strings.TrimSpace(opts.FieldName)
	if fieldName == "" {
		fieldName = "Status"
	}
	activeStates := makeStateSet(opts.ActiveStates)
	if len(activeStates) == 0 {
		return Snapshot{}, fmt.Errorf("at least one active state is required")
	}
	terminalStates := makeStateSet(opts.TerminalStates)
	if len(terminalStates) == 0 {
		return Snapshot{}, fmt.Errorf("at least one terminal state is required")
	}
	allowlist := makeRepositorySet(opts.Repositories)

	project, err := c.loadProjectItems(ctx, opts.Project.Owner, opts.Project.Number, fieldName)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := Snapshot{
		ProjectID:         project.ID,
		ProjectTitle:      project.Title,
		ResolvedFieldName: fieldName,
	}
	unsupportedSeen := map[string]struct{}{}

	for _, item := range project.Items {
		status := item.statusName()
		if item.IsArchived {
			snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
				ItemID:      item.ID,
				Status:      status,
				Reason:      SkipReasonArchived,
				Message:     "project item is archived",
				ObservedAt:  time.Now().UTC(),
				WasArchived: true,
			})
			continue
		}

		switch item.Content.TypeName {
		case "Issue":
			issue, err := toIssue(item)
			if err != nil {
				return Snapshot{}, fmt.Errorf("normalize project item %q: %w", item.ID, err)
			}
			if _, ok := allowlist[issue.Repository]; !ok {
				if _, seen := unsupportedSeen[issue.Repository]; !seen {
					unsupportedSeen[issue.Repository] = struct{}{}
					snapshot.UnsupportedRepositories = append(snapshot.UnsupportedRepositories, issue.Repository)
				}
				snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
					ItemID:     item.ID,
					Repository: issue.Repository,
					Status:     status,
					Reason:     SkipReasonUnsupportedRepository,
					Message:    fmt.Sprintf("repository %s is not configured for this Symphony project", issue.Repository),
					Issue:      &issue,
					ObservedAt: time.Now().UTC(),
				})
				continue
			}
			if item.Content.Closed || strings.EqualFold(item.Content.State, "CLOSED") {
				snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
					ItemID:     item.ID,
					Repository: issue.Repository,
					Status:     status,
					Reason:     SkipReasonIssueClosed,
					Message:    "issue is closed",
					Issue:      &issue,
					ObservedAt: time.Now().UTC(),
				})
				continue
			}
			if _, ok := terminalStates[normalizeState(status)]; ok {
				snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
					ItemID:     item.ID,
					Repository: issue.Repository,
					Status:     status,
					Reason:     SkipReasonTerminalState,
					Message:    fmt.Sprintf("status %q is configured as terminal", status),
					Issue:      &issue,
					ObservedAt: time.Now().UTC(),
				})
				continue
			}
			if _, ok := activeStates[normalizeState(status)]; !ok {
				snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
					ItemID:     item.ID,
					Repository: issue.Repository,
					Status:     status,
					Reason:     SkipReasonInactiveState,
					Message:    fmt.Sprintf("status %q is not configured as active", status),
					Issue:      &issue,
					ObservedAt: time.Now().UTC(),
				})
				continue
			}
			snapshot.Candidates = append(snapshot.Candidates, CandidateItem{ItemID: item.ID, Status: status, Issue: issue})
		case "PullRequest":
			repo := item.repositoryKey()
			snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
				ItemID:     item.ID,
				Repository: repo,
				Status:     status,
				Reason:     SkipReasonPullRequest,
				Message:    "project item is backed by a pull request",
				ObservedAt: time.Now().UTC(),
			})
		default:
			snapshot.Skipped = append(snapshot.Skipped, SkippedItem{
				ItemID:     item.ID,
				Status:     status,
				Reason:     SkipReasonUnsupportedItem,
				Message:    fmt.Sprintf("project item content type %q is not supported", item.Content.TypeName),
				ObservedAt: time.Now().UTC(),
			})
		}
	}

	sort.Strings(snapshot.UnsupportedRepositories)
	return snapshot, nil
}

func (c *Client) loadProjectItems(ctx context.Context, owner string, number int64, fieldName string) (*graphQLProject, error) {
	var (
		cursor string
		result *graphQLProject
	)
	for {
		resp, err := c.query(ctx, graphQLRequest{
			Query: projectQuery,
			Variables: map[string]any{
				"owner":     owner,
				"number":    number,
				"cursor":    cursor,
				"fieldName": fieldName,
			},
		})
		if err != nil {
			return nil, err
		}
		project := resp.Data.project()
		if project == nil {
			return nil, errProjectNotFound
		}
		if result == nil {
			result = &graphQLProject{ID: project.ID, Title: project.Title}
		}
		result.Items = append(result.Items, project.Items...)
		if !project.PageInfo.HasNextPage {
			return result, nil
		}
		cursor = project.PageInfo.EndCursor
		if cursor == "" {
			return nil, fmt.Errorf("github project pagination ended without a cursor")
		}
	}
}

func (c *Client) query(ctx context.Context, reqBody graphQLRequest) (graphQLResponse, error) {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return graphQLResponse{}, fmt.Errorf("encode github graphql request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiURL, bytes.NewReader(body))
	if err != nil {
		return graphQLResponse{}, fmt.Errorf("build github graphql request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return graphQLResponse{}, fmt.Errorf("execute github graphql request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return graphQLResponse{}, fmt.Errorf("github graphql returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var decoded graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return graphQLResponse{}, fmt.Errorf("decode github graphql response: %w", err)
	}
	if len(decoded.Errors) != 0 {
		if decoded.Data.project() != nil && graphQLErrorsAreOwnerLookupOnly(decoded.Errors) {
			return decoded, nil
		}
		parts := make([]string, 0, len(decoded.Errors))
		for _, item := range decoded.Errors {
			msg := strings.TrimSpace(item.Message)
			if msg != "" {
				parts = append(parts, msg)
			}
		}
		if len(parts) == 0 {
			parts = append(parts, "unknown graphql error")
		}
		return graphQLResponse{}, fmt.Errorf("github graphql error: %s", strings.Join(parts, "; "))
	}
	return decoded, nil
}

func graphQLErrorsAreOwnerLookupOnly(items []graphQLError) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		message := strings.TrimSpace(strings.ToLower(item.Message))
		if !strings.Contains(message, "could not resolve to an organization with the login of") && !strings.Contains(message, "could not resolve to a user with the login of") {
			return false
		}
	}
	return true
}

func toIssue(item graphQLItem) (Issue, error) {
	createdAt, err := time.Parse(time.RFC3339, item.Content.CreatedAt)
	if err != nil {
		return Issue{}, fmt.Errorf("parse createdAt: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339, item.Content.UpdatedAt)
	if err != nil {
		return Issue{}, fmt.Errorf("parse updatedAt: %w", err)
	}
	labels := make([]string, 0, len(item.Content.Labels.Nodes))
	for _, label := range item.Content.Labels.Nodes {
		name := strings.TrimSpace(label.Name)
		if name != "" {
			labels = append(labels, name)
		}
	}
	repository := item.repositoryKey()
	if repository == "" {
		return Issue{}, fmt.Errorf("repository identity is missing")
	}
	return Issue{
		NodeID:      item.Content.ID,
		Repository:  repository,
		Number:      item.Content.Number,
		Title:       item.Content.Title,
		Body:        item.Content.Body,
		Labels:      labels,
		URL:         item.Content.URL,
		CreatedAt:   createdAt.UTC(),
		UpdatedAt:   updatedAt.UTC(),
		ProjectItem: item.ID,
	}, nil
}

func makeStateSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := normalizeState(value)
		if normalized != "" {
			set[normalized] = struct{}{}
		}
	}
	return set
}

func normalizeState(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func makeRepositorySet(repos []operatorv1alpha1.SymphonyProjectRepositorySpec) map[string]struct{} {
	set := make(map[string]struct{}, len(repos))
	for _, repo := range repos {
		if key := repo.RepositoryKey(); key != "" {
			set[key] = struct{}{}
		}
	}
	return set
}

type graphQLRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   graphQLData    `json:"data"`
	Errors []graphQLError `json:"errors"`
}

type graphQLError struct {
	Message string `json:"message"`
}

type graphQLData struct {
	Organization *graphQLProjectOwner `json:"organization"`
	User         *graphQLProjectOwner `json:"user"`
}

func (d graphQLData) project() *graphQLProject {
	if d.Organization != nil && d.Organization.Project != nil {
		return d.Organization.Project
	}
	if d.User != nil && d.User.Project != nil {
		return d.User.Project
	}
	return nil
}

type graphQLProjectOwner struct {
	Project *graphQLProject `json:"projectV2"`
}

type graphQLProject struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Items    []graphQLItem
	PageInfo graphQLPageInfo
}

func (p *graphQLProject) UnmarshalJSON(data []byte) error {
	type alias struct {
		ID    string                 `json:"id"`
		Title string                 `json:"title"`
		Items *graphQLItemConnection `json:"items"`
	}
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	p.ID = decoded.ID
	p.Title = decoded.Title
	if decoded.Items != nil {
		p.Items = decoded.Items.Nodes
		p.PageInfo = decoded.Items.PageInfo
	}
	return nil
}

type graphQLItemConnection struct {
	Nodes    []graphQLItem   `json:"nodes"`
	PageInfo graphQLPageInfo `json:"pageInfo"`
}

type graphQLPageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

type graphQLItem struct {
	ID         string             `json:"id"`
	IsArchived bool               `json:"isArchived"`
	FieldValue *graphQLFieldValue `json:"fieldValueByName"`
	Content    graphQLItemContent `json:"content"`
}

func (i graphQLItem) statusName() string {
	if i.FieldValue == nil {
		return ""
	}
	return strings.TrimSpace(i.FieldValue.Name)
}

func (i graphQLItem) repositoryKey() string {
	owner := strings.TrimSpace(strings.ToLower(i.Content.Repository.Owner.Login))
	name := strings.TrimSpace(strings.ToLower(i.Content.Repository.Name))
	if owner == "" || name == "" {
		return ""
	}
	return owner + "/" + name
}

type graphQLFieldValue struct {
	TypeName string `json:"__typename"`
	Name     string `json:"name"`
}

type graphQLItemContent struct {
	TypeName   string                 `json:"__typename"`
	ID         string                 `json:"id"`
	Number     int64                  `json:"number"`
	Title      string                 `json:"title"`
	Body       string                 `json:"body"`
	URL        string                 `json:"url"`
	State      string                 `json:"state"`
	Closed     bool                   `json:"closed"`
	CreatedAt  string                 `json:"createdAt"`
	UpdatedAt  string                 `json:"updatedAt"`
	Repository graphQLRepository      `json:"repository"`
	Labels     graphQLLabelConnection `json:"labels"`
}

type graphQLRepository struct {
	Name  string                 `json:"name"`
	Owner graphQLRepositoryOwner `json:"owner"`
}

type graphQLRepositoryOwner struct {
	Login string `json:"login"`
}

type graphQLLabelConnection struct {
	Nodes []graphQLLabel `json:"nodes"`
}

type graphQLLabel struct {
	Name string `json:"name"`
}

const projectQuery = `query SymphonyProjectItems($owner: String!, $number: Int!, $cursor: String, $fieldName: String!) {
  organization(login: $owner) {
    projectV2(number: $number) {
      id
      title
      items(first: ` + "50" + `, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          isArchived
          fieldValueByName(name: $fieldName) {
            __typename
            ... on ProjectV2ItemFieldSingleSelectValue {
              name
            }
          }
          content {
            __typename
            ... on Issue {
              id
              number
              title
              body
              url
              state
              closed
              createdAt
              updatedAt
              repository {
                name
                owner {
                  login
                }
              }
              labels(first: 20) {
                nodes {
                  name
                }
              }
            }
            ... on PullRequest {
              id
              number
              title
              url
              repository {
                name
                owner {
                  login
                }
              }
            }
          }
        }
      }
    }
  }
  user(login: $owner) {
    projectV2(number: $number) {
      id
      title
      items(first: ` + "50" + `, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          isArchived
          fieldValueByName(name: $fieldName) {
            __typename
            ... on ProjectV2ItemFieldSingleSelectValue {
              name
            }
          }
          content {
            __typename
            ... on Issue {
              id
              number
              title
              body
              url
              state
              closed
              createdAt
              updatedAt
              repository {
                name
                owner {
                  login
                }
              }
              labels(first: 20) {
                nodes {
                  name
                }
              }
            }
            ... on PullRequest {
              id
              number
              title
              url
              repository {
                name
                owner {
                  login
                }
              }
            }
          }
        }
      }
    }
  }
}`
