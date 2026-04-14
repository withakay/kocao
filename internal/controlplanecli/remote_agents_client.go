package controlplanecli

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
)

type RemoteAgentSessionBinding struct {
	HarnessRunID string                        `json:"harnessRunId,omitempty"`
	SessionID    string                        `json:"sessionId,omitempty"`
	PodName      string                        `json:"podName,omitempty"`
	Runtime      operatorv1alpha1.AgentRuntime `json:"runtime,omitempty"`
	Agent        operatorv1alpha1.AgentKind    `json:"agent,omitempty"`
}

type RemoteAgent struct {
	ID                 string                        `json:"id"`
	Name               string                        `json:"name"`
	DisplayName        string                        `json:"displayName,omitempty"`
	Description        string                        `json:"description,omitempty"`
	PoolID             string                        `json:"poolId,omitempty"`
	PoolName           string                        `json:"poolName,omitempty"`
	WorkspaceSessionID string                        `json:"workspaceSessionId,omitempty"`
	Runtime            operatorv1alpha1.AgentRuntime `json:"runtime,omitempty"`
	Agent              operatorv1alpha1.AgentKind    `json:"agent,omitempty"`
	Availability       string                        `json:"availability,omitempty"`
	CurrentTaskID      string                        `json:"currentTaskId,omitempty"`
	LastActivityAt     string                        `json:"lastActivityAt,omitempty"`
	CurrentSession     *RemoteAgentSessionBinding    `json:"currentSession,omitempty"`
	CreatedAt          string                        `json:"createdAt,omitempty"`
	UpdatedAt          string                        `json:"updatedAt,omitempty"`
}

type RemoteAgentArtifact struct {
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
	InputArtifacts     []RemoteAgentArtifact        `json:"inputArtifacts,omitempty"`
	OutputArtifacts    []RemoteAgentArtifact        `json:"outputArtifacts,omitempty"`
	Transcript         []RemoteAgentTranscriptEntry `json:"transcript,omitempty"`
}

type RemoteAgentTaskTarget struct {
	AgentID            string `json:"agentId,omitempty"`
	AgentName          string `json:"agentName,omitempty"`
	PoolName           string `json:"poolName,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
}

type RemoteAgentTaskCreateRequest struct {
	Target         RemoteAgentTaskTarget `json:"target"`
	Prompt         string                `json:"prompt"`
	TimeoutSeconds int32                 `json:"timeoutSeconds,omitempty"`
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

func (c *Client) ListRemoteAgentTasks(ctx context.Context) ([]RemoteAgentTask, error) {
	var payload struct {
		RemoteAgentTasks []RemoteAgentTask `json:"remoteAgentTasks"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/remote-agent-tasks", nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.RemoteAgentTasks, nil
}

func (c *Client) GetRemoteAgentTask(ctx context.Context, taskID string) (RemoteAgentTask, error) {
	var out RemoteAgentTask
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID))
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &out); err != nil {
		return RemoteAgentTask{}, err
	}
	return out, nil
}

func (c *Client) DispatchRemoteAgentTask(ctx context.Context, req RemoteAgentTaskCreateRequest) (RemoteAgentTask, error) {
	var out RemoteAgentTask
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/remote-agent-tasks", nil, req, &out); err != nil {
		return RemoteAgentTask{}, err
	}
	return out, nil
}

func (c *Client) CancelRemoteAgentTask(ctx context.Context, taskID string) (RemoteAgentTask, error) {
	var out RemoteAgentTask
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/cancel"
	if err := c.doJSON(ctx, http.MethodPost, route, nil, nil, &out); err != nil {
		return RemoteAgentTask{}, err
	}
	return out, nil
}

func (c *Client) GetRemoteAgentTaskTranscript(ctx context.Context, taskID string) ([]RemoteAgentTranscriptEntry, error) {
	var payload struct {
		Transcript []RemoteAgentTranscriptEntry `json:"transcript"`
	}
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/transcript"
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &payload); err != nil {
		return nil, err
	}
	return payload.Transcript, nil
}

func (c *Client) GetRemoteAgentTaskArtifacts(ctx context.Context, taskID string) ([]RemoteAgentArtifact, []RemoteAgentArtifact, error) {
	var payload struct {
		InputArtifacts  []RemoteAgentArtifact `json:"inputArtifacts"`
		OutputArtifacts []RemoteAgentArtifact `json:"outputArtifacts"`
	}
	route := "/api/v1/remote-agent-tasks/" + url.PathEscape(strings.TrimSpace(taskID)) + "/artifacts"
	if err := c.doJSON(ctx, http.MethodGet, route, nil, nil, &payload); err != nil {
		return nil, nil, err
	}
	return payload.InputArtifacts, payload.OutputArtifacts, nil
}
