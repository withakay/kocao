package controlplanecli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type WorkspaceSession struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	RepoURL     string `json:"repoURL,omitempty"`
	Phase       string `json:"phase,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

type HarnessRun struct {
	ID                 string `json:"id"`
	DisplayName        string `json:"displayName,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionID,omitempty"`
	RepoURL            string `json:"repoURL"`
	RepoRevision       string `json:"repoRevision,omitempty"`
	Image              string `json:"image"`
	Phase              string `json:"phase,omitempty"`
	PodName            string `json:"podName,omitempty"`
	GitHubBranch       string `json:"gitHubBranch,omitempty"`
	PullRequestURL     string `json:"pullRequestURL,omitempty"`
	PullRequestStatus  string `json:"pullRequestStatus,omitempty"`
}

type PodLogs struct {
	PodName   string `json:"podName"`
	Container string `json:"container,omitempty"`
	TailLines int64  `json:"tailLines"`
	Logs      string `json:"logs"`
}

type AttachTokenResponse struct {
	Token              string `json:"token"`
	ExpiresAt          string `json:"expiresAt"`
	WorkspaceSessionID string `json:"workspaceSessionID"`
	ClientID           string `json:"clientID"`
	Role               string `json:"role"`
}

type Client struct {
	baseURL    *url.URL
	token      string
	httpClient *http.Client
	verbose    bool
	logOutput  io.Writer
}

func NewClient(cfg Config) (*Client, error) {
	normalized, err := cfg.normalized()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(normalized.Token) == "" {
		return nil, ErrMissingToken
	}
	base, err := url.Parse(normalized.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	return &Client{
		baseURL:    base,
		token:      normalized.Token,
		httpClient: &http.Client{Timeout: normalized.Timeout},
		verbose:    normalized.Verbose,
		logOutput:  normalized.LogOutput,
	}, nil
}

func (c *Client) apiURL(routePath string, query url.Values) string {
	u := *c.baseURL
	u.Path = path.Join("/", strings.TrimPrefix(c.baseURL.Path, "/"), strings.TrimPrefix(routePath, "/"))
	if query != nil {
		u.RawQuery = query.Encode()
	} else {
		u.RawQuery = ""
	}
	return u.String()
}

func (c *Client) doJSON(ctx context.Context, method string, routePath string, query url.Values, body any, out any) error {
	var reqBody io.Reader
	requestURL := c.apiURL(routePath, query)
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
		c.debugf("-> %s %s body=%s", method, requestURL, truncateForLog(string(b), 240))
	} else {
		c.debugf("-> %s %s", method, requestURL)
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, reqBody)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	b, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	c.debugf("<- %s %s status=%d content-type=%q bytes=%d", method, requestURL, resp.StatusCode, contentType, len(b))
	if c.verbose && len(bytes.TrimSpace(b)) != 0 {
		c.debugf("<- body: %s", truncateForLog(string(b), 320))
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiErr := &APIError{StatusCode: resp.StatusCode, Body: string(b), Method: method, URL: requestURL, ContentType: contentType}
		var payload struct {
			Error string `json:"error"`
		}
		if len(bytes.TrimSpace(b)) != 0 && json.Unmarshal(b, &payload) == nil {
			apiErr.Message = strings.TrimSpace(payload.Error)
		}
		if apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(b))
		}
		return apiErr
	}

	if out == nil || len(bytes.TrimSpace(b)) == 0 {
		return nil
	}
	if err := json.Unmarshal(b, out); err != nil {
		return &DecodeError{
			Method:      method,
			URL:         requestURL,
			StatusCode:  resp.StatusCode,
			ContentType: contentType,
			BodyPreview: truncateForLog(string(b), 320),
			Cause:       err,
		}
	}
	return nil
}

func (c *Client) debugf(format string, args ...any) {
	if !c.verbose || c.logOutput == nil {
		return
	}
	_, _ = fmt.Fprintf(c.logOutput, "debug: "+format+"\n", args...)
}

func truncateForLog(s string, max int) string {
	v := strings.TrimSpace(s)
	if max <= 0 || len(v) <= max {
		return v
	}
	return v[:max] + "..."
}

func (c *Client) ListWorkspaceSessions(ctx context.Context) ([]WorkspaceSession, error) {
	var payload struct {
		WorkspaceSessions []WorkspaceSession `json:"workspaceSessions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/workspace-sessions", nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.WorkspaceSessions, nil
}

func (c *Client) GetWorkspaceSession(ctx context.Context, sessionID string) (WorkspaceSession, error) {
	var out WorkspaceSession
	route := "/api/v1/workspace-sessions/" + url.PathEscape(strings.TrimSpace(sessionID))
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return WorkspaceSession{}, err
	}
	return out, nil
}

func (c *Client) ListHarnessRuns(ctx context.Context, workspaceSessionID string) ([]HarnessRun, error) {
	query := url.Values{}
	if strings.TrimSpace(workspaceSessionID) != "" {
		query.Set("workspaceSessionID", strings.TrimSpace(workspaceSessionID))
	}
	var payload struct {
		HarnessRuns []HarnessRun `json:"harnessRuns"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/harness-runs", query, nil, &payload); err != nil {
		return nil, err
	}
	return payload.HarnessRuns, nil
}

func (c *Client) GetPodLogs(ctx context.Context, podName string, container string, tailLines int64) (PodLogs, error) {
	query := url.Values{}
	if strings.TrimSpace(container) != "" {
		query.Set("container", strings.TrimSpace(container))
	}
	if tailLines > 0 {
		query.Set("tailLines", fmt.Sprintf("%d", tailLines))
	}
	route := "/api/v1/pods/" + url.PathEscape(strings.TrimSpace(podName)) + "/logs"
	var out PodLogs
	if err := c.doJSON(ctx, http.MethodGet, route, query, nil, &out); err != nil {
		return PodLogs{}, err
	}
	return out, nil
}

func (c *Client) CreateAttachToken(ctx context.Context, workspaceSessionID string, role string) (AttachTokenResponse, error) {
	var out AttachTokenResponse
	body := map[string]string{"role": role}
	route := "/api/v1/workspace-sessions/" + url.PathEscape(strings.TrimSpace(workspaceSessionID)) + "/attach-token"
	if err := c.doJSON(ctx, http.MethodPost, route, nil, body, &out); err != nil {
		return AttachTokenResponse{}, err
	}
	return out, nil
}
