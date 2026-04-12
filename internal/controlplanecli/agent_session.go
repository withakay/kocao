package controlplanecli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AgentSession represents an agent session associated with a harness run.
type AgentSession struct {
	SessionID   string    `json:"sessionId"`
	RunID       string    `json:"runId"`
	DisplayName string    `json:"displayName"`
	Runtime     string    `json:"runtime"`
	Agent       string    `json:"agent"`
	Phase       string    `json:"phase"`
	WorkspaceID string    `json:"workspaceSessionId"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

// AgentSessionEvent represents a single event from an agent session event stream.
type AgentSessionEvent struct {
	Seq       int             `json:"seq"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// PromptRequest is the request body for sending a prompt to an agent session.
type PromptRequest struct {
	Prompt string `json:"prompt"`
}

// PromptResponse is the response from sending a prompt to an agent session.
type PromptResponse struct {
	Events []AgentSessionEvent `json:"events"`
}

// ListAgentSessions returns all agent sessions for the given workspace session.
func (c *Client) ListAgentSessions(ctx context.Context, workspaceID string) ([]AgentSession, error) {
	route := "/api/v1/workspace-sessions/" + url.PathEscape(strings.TrimSpace(workspaceID)) + "/agent-sessions"
	var payload struct {
		AgentSessions []AgentSession `json:"agentSessions"`
	}
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.AgentSessions, nil
}

// GetAgentSession returns the agent session for the given harness run.
func (c *Client) GetAgentSession(ctx context.Context, runID string) (*AgentSession, error) {
	route := "/api/v1/harness-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/agent-session"
	var out AgentSession
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateAgentSession creates a new agent session for the given harness run.
func (c *Client) CreateAgentSession(ctx context.Context, runID string) (*AgentSession, error) {
	route := "/api/v1/harness-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/agent-session"
	var out AgentSession
	if err := c.doJSON(ctx, http.MethodPost, route, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StopAgentSession stops the agent session for the given harness run.
func (c *Client) StopAgentSession(ctx context.Context, runID string) error {
	route := "/api/v1/harness-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/agent-session/stop"
	return c.doJSON(ctx, http.MethodPost, route, nil, nil, nil)
}

// SendPrompt sends a prompt to the agent session and returns the response events.
func (c *Client) SendPrompt(ctx context.Context, runID string, prompt string) (*PromptResponse, error) {
	route := "/api/v1/harness-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/agent-session/prompt"
	req := PromptRequest{Prompt: prompt}
	var out PromptResponse
	if err := c.doJSON(ctx, http.MethodPost, route, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StreamEvents opens an SSE stream for agent session events and returns the
// raw response body. The caller is responsible for closing the returned reader.
func (c *Client) StreamEvents(ctx context.Context, runID string) (io.ReadCloser, error) {
	route := "/api/v1/harness-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/agent-session/events/stream"
	requestURL := c.apiURL(route, nil)
	c.debugf("-> GET %s (stream)", requestURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		defer func() { _ = resp.Body.Close() }()
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
		apiErr := &APIError{
			StatusCode:  resp.StatusCode,
			Body:        string(b),
			Method:      http.MethodGet,
			URL:         requestURL,
			ContentType: strings.TrimSpace(resp.Header.Get("Content-Type")),
		}
		var payload struct {
			Error string `json:"error"`
		}
		if len(bytes.TrimSpace(b)) != 0 && json.Unmarshal(b, &payload) == nil {
			apiErr.Message = strings.TrimSpace(payload.Error)
		}
		if apiErr.Message == "" {
			apiErr.Message = strings.TrimSpace(string(b))
		}
		return nil, apiErr
	}

	c.debugf("<- GET %s status=%d content-type=%q (streaming)", requestURL, resp.StatusCode, resp.Header.Get("Content-Type"))
	return resp.Body, nil
}

// GetEvents returns all events for the given agent session.
func (c *Client) GetEvents(ctx context.Context, runID string) ([]AgentSessionEvent, error) {
	route := "/api/v1/harness-runs/" + url.PathEscape(strings.TrimSpace(runID)) + "/agent-session/events"
	var payload struct {
		Events []AgentSessionEvent `json:"events"`
	}
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Events, nil
}

// createWorkspaceSessionRequest is the request body for creating a workspace session.
type createWorkspaceSessionRequest struct {
	DisplayName string `json:"displayName,omitempty"`
	RepoURL     string `json:"repoURL,omitempty"`
}

// createHarnessRunRequest is the request body for creating a harness run.
type createHarnessRunRequest struct {
	RepoURL      string `json:"repoURL"`
	RepoRevision string `json:"repoRevision,omitempty"`
	Image        string `json:"image"`
}

// CreateWorkspaceSession creates a new workspace session.
func (c *Client) CreateWorkspaceSession(ctx context.Context, displayName string, repoURL string) (*WorkspaceSession, error) {
	req := createWorkspaceSessionRequest{
		DisplayName: displayName,
		RepoURL:     repoURL,
	}
	var out WorkspaceSession
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/workspace-sessions", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateHarnessRun creates a new harness run under the given workspace session.
func (c *Client) CreateHarnessRun(ctx context.Context, workspaceSessionID string, repoURL string, repoRevision string, image string) (*HarnessRun, error) {
	route := "/api/v1/workspace-sessions/" + url.PathEscape(strings.TrimSpace(workspaceSessionID)) + "/harness-runs"
	req := createHarnessRunRequest{
		RepoURL:      repoURL,
		RepoRevision: repoRevision,
		Image:        image,
	}
	var out HarnessRun
	if err := c.doJSON(ctx, http.MethodPost, route, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// StartAgent orchestrates the full resource chain for starting an agent:
// creates a workspace session (if workspaceID is empty), creates a harness run
// with an agent session, and returns the run ID.
func (c *Client) StartAgent(ctx context.Context, workspaceID, repoURL, repoRevision, agent, image string) (runID string, err error) {
	wsID := strings.TrimSpace(workspaceID)
	if wsID == "" {
		ws, err := c.CreateWorkspaceSession(ctx, "", repoURL)
		if err != nil {
			return "", fmt.Errorf("create workspace session: %w", err)
		}
		wsID = ws.ID
	}

	run, err := c.CreateHarnessRun(ctx, wsID, repoURL, repoRevision, image)
	if err != nil {
		return "", fmt.Errorf("create harness run: %w", err)
	}

	if _, err := c.CreateAgentSession(ctx, run.ID); err != nil {
		return "", fmt.Errorf("create agent session: %w", err)
	}

	return run.ID, nil
}
