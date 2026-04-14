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

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

type WorkspaceSession struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName,omitempty"`
	RepoURL     string `json:"repoURL,omitempty"`
	Phase       string `json:"phase,omitempty"`
	CreatedAt   string `json:"createdAt,omitempty"`
}

type AgentSessionInfo struct {
	Runtime   string `json:"runtime,omitempty"`
	Agent     string `json:"agent,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	Phase     string `json:"phase,omitempty"`
}

type HarnessRun struct {
	ID                 string                                           `json:"id"`
	DisplayName        string                                           `json:"displayName,omitempty"`
	WorkspaceSessionID string                                           `json:"workspaceSessionID,omitempty"`
	RepoURL            string                                           `json:"repoURL"`
	RepoRevision       string                                           `json:"repoRevision,omitempty"`
	Image              string                                           `json:"image"`
	ImageProfile       *operatorv1alpha1.HarnessImageProfileStatus      `json:"imageProfile,omitempty"`
	StartupMetrics     *operatorv1alpha1.HarnessRunStartupMetricsStatus `json:"startupMetrics,omitempty"`
	Phase              string                                           `json:"phase,omitempty"`
	PodName            string                                           `json:"podName,omitempty"`
	AgentSession       *AgentSessionInfo                                `json:"agentSession,omitempty"`
	GitHubBranch       string                                           `json:"gitHubBranch,omitempty"`
	PullRequestURL     string                                           `json:"pullRequestURL,omitempty"`
	PullRequestStatus  string                                           `json:"pullRequestStatus,omitempty"`
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
	Mode               string `json:"mode,omitempty"`
}

type SymphonyProject struct {
	Name       string                                 `json:"name"`
	Namespace  string                                 `json:"namespace,omitempty"`
	CreatedAt  string                                 `json:"createdAt,omitempty"`
	Generation int64                                  `json:"generation,omitempty"`
	Paused     bool                                   `json:"paused"`
	Spec       operatorv1alpha1.SymphonyProjectSpec   `json:"spec"`
	Status     operatorv1alpha1.SymphonyProjectStatus `json:"status"`
}

type SymphonyProjectRequest struct {
	Name string                               `json:"name,omitempty"`
	Spec operatorv1alpha1.SymphonyProjectSpec `json:"spec"`
}

type RemoteAgentSessionBinding struct {
	HarnessRunID string `json:"harnessRunId,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
	PodName      string `json:"podName,omitempty"`
	Runtime      string `json:"runtime,omitempty"`
	Agent        string `json:"agent,omitempty"`
}

type RemoteAgentPool struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	DisplayName        string `json:"displayName,omitempty"`
	Description        string `json:"description,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
	CreatedAt          string `json:"createdAt,omitempty"`
	UpdatedAt          string `json:"updatedAt,omitempty"`
}

type RemoteAgentPoolCreateRequest struct {
	Name               string `json:"name"`
	DisplayName        string `json:"displayName,omitempty"`
	Description        string `json:"description,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
}

type RemoteAgent struct {
	ID                 string                     `json:"id"`
	Name               string                     `json:"name"`
	DisplayName        string                     `json:"displayName,omitempty"`
	Description        string                     `json:"description,omitempty"`
	PoolID             string                     `json:"poolId,omitempty"`
	PoolName           string                     `json:"poolName,omitempty"`
	WorkspaceSessionID string                     `json:"workspaceSessionId,omitempty"`
	Runtime            string                     `json:"runtime,omitempty"`
	Agent              string                     `json:"agent,omitempty"`
	Availability       string                     `json:"availability,omitempty"`
	CurrentTaskID      string                     `json:"currentTaskId,omitempty"`
	LastActivityAt     string                     `json:"lastActivityAt,omitempty"`
	CurrentSession     *RemoteAgentSessionBinding `json:"currentSession,omitempty"`
	CreatedAt          string                     `json:"createdAt,omitempty"`
	UpdatedAt          string                     `json:"updatedAt,omitempty"`
}

type RemoteAgentCreateRequest struct {
	Name               string                     `json:"name"`
	DisplayName        string                     `json:"displayName,omitempty"`
	Description        string                     `json:"description,omitempty"`
	PoolID             string                     `json:"poolId,omitempty"`
	PoolName           string                     `json:"poolName,omitempty"`
	WorkspaceSessionID string                     `json:"workspaceSessionId,omitempty"`
	Runtime            string                     `json:"runtime,omitempty"`
	Agent              string                     `json:"agent,omitempty"`
	CurrentSession     *RemoteAgentSessionBinding `json:"currentSession,omitempty"`
}

type RemoteAgentArtifactRef struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	MediaType string `json:"mediaType,omitempty"`
	Path      string `json:"path,omitempty"`
	URI       string `json:"uri,omitempty"`
	Digest    string `json:"digest,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
}

type RemoteAgentArtifactCreateRequest struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	MediaType string `json:"mediaType,omitempty"`
	Path      string `json:"path,omitempty"`
	URI       string `json:"uri,omitempty"`
	Digest    string `json:"digest,omitempty"`
	SizeBytes int64  `json:"sizeBytes,omitempty"`
}

type RemoteAgentTranscriptEntry struct {
	Sequence int64  `json:"sequence"`
	At       string `json:"at,omitempty"`
	Role     string `json:"role"`
	Kind     string `json:"kind,omitempty"`
	Text     string `json:"text,omitempty"`
	EventRef string `json:"eventRef,omitempty"`
}

type RemoteAgentTaskResult struct {
	Summary             string `json:"summary,omitempty"`
	Outcome             string `json:"outcome,omitempty"`
	TranscriptEntries   int    `json:"transcriptEntries,omitempty"`
	OutputArtifactCount int    `json:"outputArtifactCount,omitempty"`
}

type RemoteAgentTaskTarget struct {
	AgentID            string `json:"agentId,omitempty"`
	AgentName          string `json:"agentName,omitempty"`
	PoolName           string `json:"poolName,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
}

type RemoteAgentTask struct {
	ID                 string                       `json:"id"`
	RequestedBy        string                       `json:"requestedBy,omitempty"`
	AgentID            string                       `json:"agentId,omitempty"`
	AgentName          string                       `json:"agentName,omitempty"`
	PoolID             string                       `json:"poolId,omitempty"`
	PoolName           string                       `json:"poolName,omitempty"`
	WorkspaceSessionID string                       `json:"workspaceSessionId,omitempty"`
	Prompt             string                       `json:"prompt,omitempty"`
	State              string                       `json:"state"`
	TimeoutSeconds     int32                        `json:"timeoutSeconds,omitempty"`
	Attempt            int                          `json:"attempt,omitempty"`
	RetryCount         int                          `json:"retryCount,omitempty"`
	CurrentSession     *RemoteAgentSessionBinding   `json:"currentSession,omitempty"`
	CreatedAt          string                       `json:"createdAt,omitempty"`
	AssignedAt         string                       `json:"assignedAt,omitempty"`
	StartedAt          string                       `json:"startedAt,omitempty"`
	CompletedAt        string                       `json:"completedAt,omitempty"`
	CancelledAt        string                       `json:"cancelledAt,omitempty"`
	LastTransitionAt   string                       `json:"lastTransitionAt,omitempty"`
	Result             *RemoteAgentTaskResult       `json:"result,omitempty"`
	InputArtifacts     []RemoteAgentArtifactRef     `json:"inputArtifacts,omitempty"`
	OutputArtifacts    []RemoteAgentArtifactRef     `json:"outputArtifacts,omitempty"`
	Transcript         []RemoteAgentTranscriptEntry `json:"transcript,omitempty"`
}

type RemoteAgentTaskCreateRequest struct {
	Target         RemoteAgentTaskTarget              `json:"target"`
	Prompt         string                             `json:"prompt"`
	TimeoutSeconds int32                              `json:"timeoutSeconds,omitempty"`
	InputArtifacts []RemoteAgentArtifactCreateRequest `json:"inputArtifacts,omitempty"`
}

type RemoteAgentTaskTranscript struct {
	TaskID     string                       `json:"taskId"`
	Transcript []RemoteAgentTranscriptEntry `json:"transcript"`
}

type RemoteAgentTaskArtifacts struct {
	TaskID          string                   `json:"taskId"`
	InputArtifacts  []RemoteAgentArtifactRef `json:"inputArtifacts"`
	OutputArtifacts []RemoteAgentArtifactRef `json:"outputArtifacts"`
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

func (c *Client) ListRemoteAgentPools(ctx context.Context) ([]RemoteAgentPool, error) {
	var payload struct {
		RemoteAgentPools []RemoteAgentPool `json:"remoteAgentPools"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/remote-agent-pools", nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.RemoteAgentPools, nil
}

func (c *Client) CreateRemoteAgentPool(ctx context.Context, req RemoteAgentPoolCreateRequest) (RemoteAgentPool, error) {
	var out RemoteAgentPool
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/remote-agent-pools", nil, req, &out); err != nil {
		return RemoteAgentPool{}, err
	}
	return out, nil
}

func (c *Client) ListRemoteAgents(ctx context.Context) ([]RemoteAgent, error) {
	var payload struct {
		RemoteAgents []RemoteAgent `json:"remoteAgents"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/remote-agents", nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.RemoteAgents, nil
}

func (c *Client) CreateRemoteAgent(ctx context.Context, req RemoteAgentCreateRequest) (RemoteAgent, error) {
	var out RemoteAgent
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/remote-agents", nil, req, &out); err != nil {
		return RemoteAgent{}, err
	}
	return out, nil
}

func (c *Client) GetRemoteAgent(ctx context.Context, agentID string) (RemoteAgent, error) {
	var out RemoteAgent
	route := "/api/v1/remote-agents/" + url.PathEscape(strings.TrimSpace(agentID))
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return RemoteAgent{}, err
	}
	return out, nil
}

func (c *Client) ListRemoteAgentTasks(ctx context.Context) ([]RemoteAgentTask, error) {
	var payload struct {
		RemoteAgentTasks []RemoteAgentTask `json:"remoteAgentTasks"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/remote-agent-tasks", nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.RemoteAgentTasks, nil
}

func (c *Client) CreateRemoteAgentTask(ctx context.Context, req RemoteAgentTaskCreateRequest) (RemoteAgentTask, error) {
	var out RemoteAgentTask
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/remote-agent-tasks", nil, req, &out); err != nil {
		return RemoteAgentTask{}, err
	}
	return out, nil
}

func (c *Client) GetRemoteAgentTask(ctx context.Context, taskID string) (RemoteAgentTask, error) {
	var out RemoteAgentTask
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID))
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return RemoteAgentTask{}, err
	}
	return out, nil
}

func (c *Client) CancelRemoteAgentTask(ctx context.Context, taskID string) (RemoteAgentTask, error) {
	return c.controlRemoteAgentTask(ctx, taskID, "cancel")
}

func (c *Client) RetryRemoteAgentTask(ctx context.Context, taskID string) (RemoteAgentTask, error) {
	return c.controlRemoteAgentTask(ctx, taskID, "retry")
}

func (c *Client) GetRemoteAgentTaskTranscript(ctx context.Context, taskID string) (RemoteAgentTaskTranscript, error) {
	var out RemoteAgentTaskTranscript
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/transcript"
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return RemoteAgentTaskTranscript{}, err
	}
	return out, nil
}

func (c *Client) GetRemoteAgentTaskArtifacts(ctx context.Context, taskID string) (RemoteAgentTaskArtifacts, error) {
	var out RemoteAgentTaskArtifacts
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/artifacts"
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return RemoteAgentTaskArtifacts{}, err
	}
	return out, nil
}

func (c *Client) controlRemoteAgentTask(ctx context.Context, taskID string, action string) (RemoteAgentTask, error) {
	var out RemoteAgentTask
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/" + action
	if err := c.doJSON(ctx, http.MethodPost, route, nil, nil, &out); err != nil {
		return RemoteAgentTask{}, err
	}
	return out, nil
}

func (c *Client) CreateAttachToken(ctx context.Context, workspaceSessionID string, role string, mode string) (AttachTokenResponse, error) {
	var out AttachTokenResponse
	body := map[string]string{"role": role, "mode": mode}
	route := "/api/v1/workspace-sessions/" + url.PathEscape(strings.TrimSpace(workspaceSessionID)) + "/attach-token"
	if err := c.doJSON(ctx, http.MethodPost, route, nil, body, &out); err != nil {
		return AttachTokenResponse{}, err
	}
	return out, nil
}

func (c *Client) ListSymphonyProjects(ctx context.Context) ([]SymphonyProject, error) {
	var payload struct {
		SymphonyProjects []SymphonyProject `json:"symphonyProjects"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/symphony-projects", nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.SymphonyProjects, nil
}

func (c *Client) GetSymphonyProject(ctx context.Context, name string) (SymphonyProject, error) {
	var out SymphonyProject
	route := "/api/v1/symphony-projects/" + url.PathEscape(strings.TrimSpace(name))
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return SymphonyProject{}, err
	}
	return out, nil
}

func (c *Client) CreateSymphonyProject(ctx context.Context, req SymphonyProjectRequest) (SymphonyProject, error) {
	var out SymphonyProject
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/symphony-projects", nil, req, &out); err != nil {
		return SymphonyProject{}, err
	}
	return out, nil
}

func (c *Client) UpdateSymphonyProject(ctx context.Context, name string, req SymphonyProjectRequest) (SymphonyProject, error) {
	var out SymphonyProject
	route := "/api/v1/symphony-projects/" + url.PathEscape(strings.TrimSpace(name))
	if err := c.doJSON(ctx, http.MethodPatch, route, nil, req, &out); err != nil {
		return SymphonyProject{}, err
	}
	return out, nil
}

func (c *Client) PauseSymphonyProject(ctx context.Context, name string) (SymphonyProject, error) {
	return c.controlSymphonyProject(ctx, name, "pause")
}

func (c *Client) ResumeSymphonyProject(ctx context.Context, name string) (SymphonyProject, error) {
	return c.controlSymphonyProject(ctx, name, "resume")
}

func (c *Client) RefreshSymphonyProject(ctx context.Context, name string) (SymphonyProject, error) {
	return c.controlSymphonyProject(ctx, name, "refresh")
}

func (c *Client) controlSymphonyProject(ctx context.Context, name string, action string) (SymphonyProject, error) {
	var out SymphonyProject
	route := "/api/v1/symphony-projects/" + url.PathEscape(strings.TrimSpace(name)) + "/" + action
	if err := c.doJSON(ctx, http.MethodPost, route, nil, nil, &out); err != nil {
		return SymphonyProject{}, err
	}
	return out, nil
}
