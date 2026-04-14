package controlplaneapi

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type remoteAgentAvailability string

const (
	remoteAgentAvailabilityIdle    remoteAgentAvailability = "idle"
	remoteAgentAvailabilityBusy    remoteAgentAvailability = "busy"
	remoteAgentAvailabilityOffline remoteAgentAvailability = "offline"
)

type remoteAgentTaskState string

const (
	remoteAgentTaskStateAssigned  remoteAgentTaskState = "assigned"
	remoteAgentTaskStateRunning   remoteAgentTaskState = "running"
	remoteAgentTaskStateCompleted remoteAgentTaskState = "completed"
	remoteAgentTaskStateFailed    remoteAgentTaskState = "failed"
	remoteAgentTaskStateTimedOut  remoteAgentTaskState = "timed_out"
	remoteAgentTaskStateCancelled remoteAgentTaskState = "cancelled"
)

type remoteAgentArtifactKind string

const (
	remoteAgentArtifactKindFile   remoteAgentArtifactKind = "file"
	remoteAgentArtifactKindPatch  remoteAgentArtifactKind = "patch"
	remoteAgentArtifactKindBundle remoteAgentArtifactKind = "bundle"
	remoteAgentArtifactKindReport remoteAgentArtifactKind = "report"
)

type remoteAgentTranscriptRole string

const (
	remoteAgentTranscriptRoleSystem remoteAgentTranscriptRole = "system"
	remoteAgentTranscriptRoleUser   remoteAgentTranscriptRole = "user"
	remoteAgentTranscriptRoleAgent  remoteAgentTranscriptRole = "agent"
	remoteAgentTranscriptRoleTool   remoteAgentTranscriptRole = "tool"
)

type remoteAgentSessionBinding struct {
	HarnessRunID string                        `json:"harnessRunId,omitempty"`
	SessionID    string                        `json:"sessionId,omitempty"`
	PodName      string                        `json:"podName,omitempty"`
	Runtime      operatorv1alpha1.AgentRuntime `json:"runtime,omitempty"`
	Agent        operatorv1alpha1.AgentKind    `json:"agent,omitempty"`
}

type remoteAgentPool struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	DisplayName        string `json:"displayName,omitempty"`
	Description        string `json:"description,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
	CreatedAt          string `json:"createdAt,omitempty"`
	UpdatedAt          string `json:"updatedAt,omitempty"`
}

type remoteAgent struct {
	ID                 string                        `json:"id"`
	Name               string                        `json:"name"`
	DisplayName        string                        `json:"displayName,omitempty"`
	Description        string                        `json:"description,omitempty"`
	PoolID             string                        `json:"poolId,omitempty"`
	PoolName           string                        `json:"poolName,omitempty"`
	WorkspaceSessionID string                        `json:"workspaceSessionId,omitempty"`
	Runtime            operatorv1alpha1.AgentRuntime `json:"runtime,omitempty"`
	Agent              operatorv1alpha1.AgentKind    `json:"agent,omitempty"`
	Availability       remoteAgentAvailability       `json:"availability,omitempty"`
	CurrentTaskID      string                        `json:"currentTaskId,omitempty"`
	LastActivityAt     string                        `json:"lastActivityAt,omitempty"`
	CurrentSession     *remoteAgentSessionBinding    `json:"currentSession,omitempty"`
	CreatedAt          string                        `json:"createdAt,omitempty"`
	UpdatedAt          string                        `json:"updatedAt,omitempty"`
}

type remoteAgentArtifactRef struct {
	ID        string                  `json:"id"`
	Name      string                  `json:"name"`
	Kind      remoteAgentArtifactKind `json:"kind"`
	MediaType string                  `json:"mediaType,omitempty"`
	Path      string                  `json:"path,omitempty"`
	URI       string                  `json:"uri,omitempty"`
	Digest    string                  `json:"digest,omitempty"`
	SizeBytes int64                   `json:"sizeBytes,omitempty"`
	CreatedAt string                  `json:"createdAt,omitempty"`
}

type remoteAgentTranscriptEntry struct {
	Sequence int64                     `json:"sequence"`
	At       string                    `json:"at,omitempty"`
	Role     remoteAgentTranscriptRole `json:"role"`
	Kind     string                    `json:"kind,omitempty"`
	Text     string                    `json:"text,omitempty"`
	EventRef string                    `json:"eventRef,omitempty"`
}

type remoteAgentTaskResult struct {
	Summary             string `json:"summary,omitempty"`
	Outcome             string `json:"outcome,omitempty"`
	TranscriptEntries   int    `json:"transcriptEntries,omitempty"`
	OutputArtifactCount int    `json:"outputArtifactCount,omitempty"`
}

type remoteAgentTask struct {
	ID                 string                       `json:"id"`
	RequestedBy        string                       `json:"requestedBy,omitempty"`
	AgentID            string                       `json:"agentId,omitempty"`
	AgentName          string                       `json:"agentName,omitempty"`
	PoolID             string                       `json:"poolId,omitempty"`
	PoolName           string                       `json:"poolName,omitempty"`
	WorkspaceSessionID string                       `json:"workspaceSessionId,omitempty"`
	Prompt             string                       `json:"prompt,omitempty"`
	State              remoteAgentTaskState         `json:"state"`
	TimeoutSeconds     int32                        `json:"timeoutSeconds,omitempty"`
	Attempt            int                          `json:"attempt,omitempty"`
	RetryCount         int                          `json:"retryCount,omitempty"`
	CurrentSession     *remoteAgentSessionBinding   `json:"currentSession,omitempty"`
	CreatedAt          string                       `json:"createdAt,omitempty"`
	AssignedAt         string                       `json:"assignedAt,omitempty"`
	StartedAt          string                       `json:"startedAt,omitempty"`
	CompletedAt        string                       `json:"completedAt,omitempty"`
	CancelledAt        string                       `json:"cancelledAt,omitempty"`
	LastTransitionAt   string                       `json:"lastTransitionAt,omitempty"`
	Result             *remoteAgentTaskResult       `json:"result,omitempty"`
	InputArtifacts     []remoteAgentArtifactRef     `json:"inputArtifacts,omitempty"`
	OutputArtifacts    []remoteAgentArtifactRef     `json:"outputArtifacts,omitempty"`
	Transcript         []remoteAgentTranscriptEntry `json:"transcript,omitempty"`
}

func (t remoteAgentTask) isTerminal() bool {
	switch t.State {
	case remoteAgentTaskStateCompleted, remoteAgentTaskStateFailed, remoteAgentTaskStateTimedOut, remoteAgentTaskStateCancelled:
		return true
	default:
		return false
	}
}

type remoteAgentPoolCreateRequest struct {
	Name               string `json:"name"`
	DisplayName        string `json:"displayName,omitempty"`
	Description        string `json:"description,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
}

type remoteAgentCreateRequest struct {
	Name               string                        `json:"name"`
	DisplayName        string                        `json:"displayName,omitempty"`
	Description        string                        `json:"description,omitempty"`
	PoolID             string                        `json:"poolId,omitempty"`
	PoolName           string                        `json:"poolName,omitempty"`
	WorkspaceSessionID string                        `json:"workspaceSessionId,omitempty"`
	Runtime            operatorv1alpha1.AgentRuntime `json:"runtime,omitempty"`
	Agent              operatorv1alpha1.AgentKind    `json:"agent,omitempty"`
	CurrentSession     *remoteAgentSessionBinding    `json:"currentSession,omitempty"`
}

type remoteAgentTaskTarget struct {
	AgentID            string `json:"agentId,omitempty"`
	AgentName          string `json:"agentName,omitempty"`
	PoolName           string `json:"poolName,omitempty"`
	WorkspaceSessionID string `json:"workspaceSessionId,omitempty"`
}

type remoteAgentArtifactCreateRequest struct {
	Name      string                  `json:"name"`
	Kind      remoteAgentArtifactKind `json:"kind"`
	MediaType string                  `json:"mediaType,omitempty"`
	Path      string                  `json:"path,omitempty"`
	URI       string                  `json:"uri,omitempty"`
	Digest    string                  `json:"digest,omitempty"`
	SizeBytes int64                   `json:"sizeBytes,omitempty"`
}

type remoteAgentTaskCreateRequest struct {
	Target         remoteAgentTaskTarget              `json:"target"`
	Prompt         string                             `json:"prompt"`
	TimeoutSeconds int32                              `json:"timeoutSeconds,omitempty"`
	InputArtifacts []remoteAgentArtifactCreateRequest `json:"inputArtifacts,omitempty"`
}

type remoteAgentTaskCompleteRequest struct {
	Summary string `json:"summary,omitempty"`
	Outcome string `json:"outcome,omitempty"`
}

type remoteAgentOrchestrationStoreRecord struct {
	Type  string           `json:"type"`
	At    time.Time        `json:"at"`
	Pool  *remoteAgentPool `json:"pool,omitempty"`
	Agent *remoteAgent     `json:"agent,omitempty"`
	Task  *remoteAgentTask `json:"task,omitempty"`
}

var remoteAgentSensitiveValuePattern = regexp.MustCompile(`(?i)("?(?:api[_ -]?key|authorization|credential|password|secret|token)"?\s*[:=]\s*"?)([^"\s,;]+)`)
var remoteAgentBearerValuePattern = regexp.MustCompile(`(?i)(bearer\s+)([^"\s,;]+)`)

type RemoteAgentOrchestrationStore struct {
	mu     sync.Mutex
	path   string
	mem    []remoteAgentOrchestrationStoreRecord
	maxMem int
}

func newRemoteAgentOrchestrationStore(path string) *RemoteAgentOrchestrationStore {
	return &RemoteAgentOrchestrationStore{path: path, maxMem: 50_000}
}

func remoteAgentOrchestrationStorePath(auditPath string) string {
	if auditPath == "" {
		return ""
	}
	dir := filepath.Dir(auditPath)
	return filepath.Join(dir, "kocao.remote_agent_orchestration.jsonl")
}

func (s *RemoteAgentOrchestrationStore) append(record remoteAgentOrchestrationStoreRecord) {
	record = sanitizeRemoteAgentStoreRecord(record)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		s.mem = append(s.mem, record)
		if s.maxMem > 0 && len(s.mem) > s.maxMem {
			s.mem = s.mem[len(s.mem)-s.maxMem:]
		}
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_ = json.NewEncoder(f).Encode(record)
	_ = f.Sync()
}

func sanitizeRemoteAgentStoreRecord(record remoteAgentOrchestrationStoreRecord) remoteAgentOrchestrationStoreRecord {
	if record.Agent != nil {
		agent := *record.Agent
		agent.CurrentSession = sanitizeRemoteAgentSessionBinding(agent.CurrentSession)
		record.Agent = &agent
	}
	if record.Task != nil {
		task := *record.Task
		task.Prompt = sanitizeRemoteAgentText(task.Prompt)
		task.CurrentSession = sanitizeRemoteAgentSessionBinding(task.CurrentSession)
		if task.Result != nil {
			result := *task.Result
			result.Summary = sanitizeRemoteAgentText(result.Summary)
			result.Outcome = sanitizeRemoteAgentText(result.Outcome)
			task.Result = &result
		}
		if len(task.Transcript) != 0 {
			transcript := make([]remoteAgentTranscriptEntry, len(task.Transcript))
			for i, entry := range task.Transcript {
				entry.Text = sanitizeRemoteAgentText(entry.Text)
				entry.EventRef = sanitizeRemoteAgentText(entry.EventRef)
				transcript[i] = entry
			}
			task.Transcript = transcript
		}
		task.InputArtifacts = sanitizeRemoteAgentArtifacts(task.InputArtifacts)
		task.OutputArtifacts = sanitizeRemoteAgentArtifacts(task.OutputArtifacts)
		record.Task = &task
	}
	return record
}

func sanitizeRemoteAgentSessionBinding(binding *remoteAgentSessionBinding) *remoteAgentSessionBinding {
	if binding == nil {
		return nil
	}
	copyBinding := *binding
	copyBinding.HarnessRunID = strings.TrimSpace(copyBinding.HarnessRunID)
	copyBinding.SessionID = strings.TrimSpace(copyBinding.SessionID)
	copyBinding.PodName = strings.TrimSpace(copyBinding.PodName)
	return &copyBinding
}

func sanitizeRemoteAgentArtifacts(artifacts []remoteAgentArtifactRef) []remoteAgentArtifactRef {
	if len(artifacts) == 0 {
		return nil
	}
	out := make([]remoteAgentArtifactRef, len(artifacts))
	for i, artifact := range artifacts {
		artifact.Name = sanitizeRemoteAgentText(artifact.Name)
		artifact.Path = sanitizeRemoteAgentText(artifact.Path)
		artifact.URI = sanitizeRemoteAgentURI(artifact.URI)
		out[i] = artifact
	}
	return out
}

func sanitizeRemoteAgentText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = remoteAgentBearerValuePattern.ReplaceAllString(value, `${1}[redacted]`)
	return remoteAgentSensitiveValuePattern.ReplaceAllString(value, `${1}[redacted]`)
}

func sanitizeRemoteAgentURI(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return sanitizeRemoteAgentText(raw)
	}
	if parsed.User != nil {
		parsed.User = url.User("[redacted]")
	}
	query := parsed.Query()
	for key := range query {
		if agentSessionKeyLooksSensitive(key) {
			query.Set(key, "[redacted]")
			continue
		}
		values := query[key]
		for i := range values {
			values[i] = sanitizeRemoteAgentText(values[i])
		}
		query[key] = values
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func (s *RemoteAgentOrchestrationStore) SavePool(pool remoteAgentPool) {
	s.append(remoteAgentOrchestrationStoreRecord{Type: "pool", At: time.Now().UTC(), Pool: &pool})
}

func (s *RemoteAgentOrchestrationStore) SaveAgent(agent remoteAgent) {
	s.append(remoteAgentOrchestrationStoreRecord{Type: "agent", At: time.Now().UTC(), Agent: &agent})
}

func (s *RemoteAgentOrchestrationStore) SaveTask(task remoteAgentTask) {
	s.append(remoteAgentOrchestrationStoreRecord{Type: "task", At: time.Now().UTC(), Task: &task})
}

func (s *RemoteAgentOrchestrationStore) records() ([]remoteAgentOrchestrationStoreRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		out := make([]remoteAgentOrchestrationStoreRecord, len(s.mem))
		copy(out, s.mem)
		return out, nil
	}
	f, err := os.Open(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []remoteAgentOrchestrationStoreRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec remoteAgentOrchestrationStoreRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RemoteAgentOrchestrationStore) load() (map[string]remoteAgentPool, map[string]remoteAgent, map[string]remoteAgentTask, error) {
	records, err := s.records()
	if err != nil {
		return nil, nil, nil, err
	}
	pools := map[string]remoteAgentPool{}
	agents := map[string]remoteAgent{}
	tasks := map[string]remoteAgentTask{}
	for _, rec := range records {
		if rec.Pool != nil && rec.Pool.ID != "" {
			pools[rec.Pool.ID] = *rec.Pool
		}
		if rec.Agent != nil && rec.Agent.ID != "" {
			agents[rec.Agent.ID] = *rec.Agent
		}
		if rec.Task != nil && rec.Task.ID != "" {
			tasks[rec.Task.ID] = *rec.Task
		}
	}
	return pools, agents, tasks, nil
}

type RemoteAgentOrchestrationService struct {
	mu            sync.Mutex
	store         *RemoteAgentOrchestrationStore
	namespace     string
	k8s           client.Client
	agentSessions *AgentSessionService
	pools         map[string]remoteAgentPool
	agents        map[string]remoteAgent
	tasks         map[string]remoteAgentTask
}

func newRemoteAgentOrchestrationService(store *RemoteAgentOrchestrationStore, namespace string, k8s client.Client, agentSessions *AgentSessionService) *RemoteAgentOrchestrationService {
	service := &RemoteAgentOrchestrationService{
		store:         store,
		namespace:     namespace,
		k8s:           k8s,
		agentSessions: agentSessions,
		pools:         map[string]remoteAgentPool{},
		agents:        map[string]remoteAgent{},
		tasks:         map[string]remoteAgentTask{},
	}
	if store == nil {
		return service
	}
	if pools, agents, tasks, err := store.load(); err == nil {
		service.pools = pools
		service.agents = agents
		service.tasks = tasks
	}
	return service
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func normalizeRemoteAgentName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizePoolName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func validateRemoteAgentPoolRequest(req remoteAgentPoolCreateRequest) error {
	if normalizePoolName(req.Name) == "" {
		return &requestError{status: http.StatusBadRequest, msg: "name required"}
	}
	return nil
}

func validateRemoteAgentRequest(req remoteAgentCreateRequest) error {
	if normalizeRemoteAgentName(req.Name) == "" {
		return &requestError{status: http.StatusBadRequest, msg: "name required"}
	}
	return nil
}

func (s *RemoteAgentOrchestrationService) resolveSessionBinding(workspaceSessionID string, binding *remoteAgentSessionBinding) (*remoteAgentSessionBinding, error) {
	if binding == nil {
		return nil, nil
	}
	harnessRunID := strings.TrimSpace(binding.HarnessRunID)
	if harnessRunID == "" {
		return nil, &requestError{status: http.StatusBadRequest, msg: "currentSession.harnessRunId required"}
	}
	resolved := &remoteAgentSessionBinding{HarnessRunID: harnessRunID}
	if s.k8s == nil {
		return resolved, nil
	}
	var run operatorv1alpha1.HarnessRun
	if err := s.k8s.Get(context.Background(), client.ObjectKey{Namespace: s.namespace, Name: harnessRunID}, &run); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, &requestError{status: http.StatusBadRequest, msg: "currentSession.harnessRunId not found"}
		}
		return nil, &requestError{status: http.StatusInternalServerError, msg: "resolve currentSession.harnessRunId failed"}
	}
	if requestedWorkspace := strings.TrimSpace(workspaceSessionID); requestedWorkspace != "" && run.Spec.WorkspaceSessionName != "" && run.Spec.WorkspaceSessionName != requestedWorkspace {
		return nil, &requestError{status: http.StatusBadRequest, msg: "currentSession.harnessRunId does not belong to workspaceSessionId"}
	}
	if run.Status.PodName != "" {
		resolved.PodName = run.Status.PodName
	}
	if run.Status.AgentSession != nil {
		resolved.SessionID = strings.TrimSpace(run.Status.AgentSession.SessionID)
		resolved.Runtime = run.Status.AgentSession.Runtime
		resolved.Agent = run.Status.AgentSession.Agent
	}
	if s.agentSessions != nil && run.Spec.AgentSession != nil && run.Spec.AgentSession.Enabled() {
		state := s.agentSessions.GetState(&run)
		if state.SessionID != "" {
			resolved.SessionID = state.SessionID
		}
		if state.PodName != "" {
			resolved.PodName = state.PodName
		}
		if state.Runtime != "" {
			resolved.Runtime = state.Runtime
		}
		if state.Agent != "" {
			resolved.Agent = state.Agent
		}
	}
	if run.Spec.AgentSession != nil {
		if resolved.Runtime == "" {
			resolved.Runtime = run.Spec.AgentSession.Runtime
		}
		if resolved.Agent == "" {
			resolved.Agent = run.Spec.AgentSession.Agent
		}
	}
	return resolved, nil
}

func validateRemoteAgentTaskCreateRequest(req remoteAgentTaskCreateRequest) error {
	if strings.TrimSpace(req.Prompt) == "" {
		return &requestError{status: http.StatusBadRequest, msg: "prompt required"}
	}
	if strings.TrimSpace(req.Target.AgentID) == "" && strings.TrimSpace(req.Target.AgentName) == "" {
		return &requestError{status: http.StatusBadRequest, msg: "target.agentId or target.agentName required"}
	}
	if req.TimeoutSeconds < 0 {
		return &requestError{status: http.StatusBadRequest, msg: "timeoutSeconds must be >= 0"}
	}
	for _, artifact := range req.InputArtifacts {
		if strings.TrimSpace(artifact.Name) == "" {
			return &requestError{status: http.StatusBadRequest, msg: "inputArtifacts[].name required"}
		}
		if artifact.Kind == "" {
			return &requestError{status: http.StatusBadRequest, msg: "inputArtifacts[].kind required"}
		}
	}
	return nil
}

func (s *RemoteAgentOrchestrationService) ListPools() []remoteAgentPool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]remoteAgentPool, 0, len(s.pools))
	for _, pool := range s.pools {
		out = append(out, pool)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *RemoteAgentOrchestrationService) ListAgents() []remoteAgent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]remoteAgent, 0, len(s.agents))
	for _, agent := range s.agents {
		out = append(out, agent)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (s *RemoteAgentOrchestrationService) ListTasks() []remoteAgentTask {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireTimedOutTasksLocked(time.Now().UTC())
	out := make([]remoteAgentTask, 0, len(s.tasks))
	for _, task := range s.tasks {
		out = append(out, task)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func (s *RemoteAgentOrchestrationService) CreatePool(req remoteAgentPoolCreateRequest) (remoteAgentPool, error) {
	if err := validateRemoteAgentPoolRequest(req); err != nil {
		return remoteAgentPool{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	name := normalizePoolName(req.Name)
	for _, existing := range s.pools {
		if normalizePoolName(existing.Name) == name {
			return remoteAgentPool{}, &requestError{status: http.StatusConflict, msg: "pool name already exists"}
		}
	}
	now := nowRFC3339()
	pool := remoteAgentPool{
		ID:                 newID(),
		Name:               strings.TrimSpace(req.Name),
		DisplayName:        strings.TrimSpace(req.DisplayName),
		Description:        strings.TrimSpace(req.Description),
		WorkspaceSessionID: strings.TrimSpace(req.WorkspaceSessionID),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	s.pools[pool.ID] = pool
	if s.store != nil {
		s.store.SavePool(pool)
	}
	return pool, nil
}

func (s *RemoteAgentOrchestrationService) resolvePoolLocked(poolID, poolName string) (remoteAgentPool, bool, error) {
	if strings.TrimSpace(poolID) != "" {
		pool, ok := s.pools[strings.TrimSpace(poolID)]
		if !ok {
			return remoteAgentPool{}, false, &requestError{status: http.StatusBadRequest, msg: "pool not found"}
		}
		return pool, true, nil
	}
	if normalizePoolName(poolName) == "" {
		return remoteAgentPool{}, false, nil
	}
	for _, pool := range s.pools {
		if normalizePoolName(pool.Name) == normalizePoolName(poolName) {
			return pool, true, nil
		}
	}
	return remoteAgentPool{}, false, &requestError{status: http.StatusBadRequest, msg: "pool not found"}
}

func (s *RemoteAgentOrchestrationService) CreateAgent(req remoteAgentCreateRequest) (remoteAgent, error) {
	if err := validateRemoteAgentRequest(req); err != nil {
		return remoteAgent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pool, hasPool, err := s.resolvePoolLocked(req.PoolID, req.PoolName)
	if err != nil {
		return remoteAgent{}, err
	}
	name := normalizeRemoteAgentName(req.Name)
	for _, existing := range s.agents {
		if normalizeRemoteAgentName(existing.Name) != name {
			continue
		}
		if existing.PoolID == pool.ID {
			return remoteAgent{}, &requestError{status: http.StatusConflict, msg: "agent name already exists in pool"}
		}
		if !hasPool && existing.PoolID == "" {
			return remoteAgent{}, &requestError{status: http.StatusConflict, msg: "agent name already exists"}
		}
	}
	now := nowRFC3339()
	currentSession, err := s.resolveSessionBinding(req.WorkspaceSessionID, req.CurrentSession)
	if err != nil {
		return remoteAgent{}, err
	}
	agent := remoteAgent{
		ID:                 newID(),
		Name:               strings.TrimSpace(req.Name),
		DisplayName:        strings.TrimSpace(req.DisplayName),
		Description:        strings.TrimSpace(req.Description),
		PoolID:             pool.ID,
		PoolName:           pool.Name,
		WorkspaceSessionID: strings.TrimSpace(req.WorkspaceSessionID),
		Runtime:            req.Runtime,
		Agent:              req.Agent,
		Availability:       remoteAgentAvailabilityIdle,
		CurrentSession:     currentSession,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if agent.CurrentSession != nil {
		agent.LastActivityAt = now
	}
	s.agents[agent.ID] = agent
	if s.store != nil {
		s.store.SaveAgent(agent)
	}
	return agent, nil
}

func (s *RemoteAgentOrchestrationService) GetAgent(id string) (remoteAgent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	agent, ok := s.agents[strings.TrimSpace(id)]
	return agent, ok
}

func (s *RemoteAgentOrchestrationService) resolveAgentLocked(target remoteAgentTaskTarget) (remoteAgent, error) {
	if id := strings.TrimSpace(target.AgentID); id != "" {
		agent, ok := s.agents[id]
		if !ok {
			return remoteAgent{}, &requestError{status: http.StatusNotFound, msg: "remote agent not found"}
		}
		return agent, nil
	}
	name := normalizeRemoteAgentName(target.AgentName)
	poolName := normalizePoolName(target.PoolName)
	workspaceID := strings.TrimSpace(target.WorkspaceSessionID)
	var matches []remoteAgent
	for _, agent := range s.agents {
		if normalizeRemoteAgentName(agent.Name) != name {
			continue
		}
		if poolName != "" && normalizePoolName(agent.PoolName) != poolName {
			continue
		}
		if workspaceID != "" && agent.WorkspaceSessionID != workspaceID {
			continue
		}
		matches = append(matches, agent)
	}
	if len(matches) == 0 {
		return remoteAgent{}, &requestError{status: http.StatusNotFound, msg: "remote agent not found"}
	}
	if len(matches) > 1 {
		return remoteAgent{}, &requestError{status: http.StatusConflict, msg: "remote agent target is ambiguous; specify poolName or agentId"}
	}
	return matches[0], nil
}

func makeRemoteAgentArtifact(req remoteAgentArtifactCreateRequest) remoteAgentArtifactRef {
	return remoteAgentArtifactRef{
		ID:        newID(),
		Name:      strings.TrimSpace(req.Name),
		Kind:      req.Kind,
		MediaType: strings.TrimSpace(req.MediaType),
		Path:      strings.TrimSpace(req.Path),
		URI:       strings.TrimSpace(req.URI),
		Digest:    strings.TrimSpace(req.Digest),
		SizeBytes: req.SizeBytes,
		CreatedAt: nowRFC3339(),
	}
}

func (s *RemoteAgentOrchestrationService) DispatchTask(requestedBy string, req remoteAgentTaskCreateRequest) (remoteAgentTask, error) {
	if err := validateRemoteAgentTaskCreateRequest(req); err != nil {
		return remoteAgentTask{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	agent, err := s.resolveAgentLocked(req.Target)
	if err != nil {
		return remoteAgentTask{}, err
	}
	if agent.Availability == remoteAgentAvailabilityBusy && strings.TrimSpace(agent.CurrentTaskID) != "" {
		return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: "remote agent is busy; wait for the active task to finish or cancel it"}
	}
	now := nowRFC3339()
	inputArtifacts := make([]remoteAgentArtifactRef, 0, len(req.InputArtifacts))
	for _, artifact := range req.InputArtifacts {
		inputArtifacts = append(inputArtifacts, makeRemoteAgentArtifact(artifact))
	}
	task := remoteAgentTask{
		ID:                 newID(),
		RequestedBy:        strings.TrimSpace(requestedBy),
		AgentID:            agent.ID,
		AgentName:          agent.Name,
		PoolID:             agent.PoolID,
		PoolName:           agent.PoolName,
		WorkspaceSessionID: agent.WorkspaceSessionID,
		Prompt:             strings.TrimSpace(req.Prompt),
		State:              remoteAgentTaskStateAssigned,
		TimeoutSeconds:     req.TimeoutSeconds,
		Attempt:            1,
		CurrentSession:     agent.CurrentSession,
		CreatedAt:          now,
		AssignedAt:         now,
		LastTransitionAt:   now,
		InputArtifacts:     inputArtifacts,
	}
	s.tasks[task.ID] = task
	agent.Availability = remoteAgentAvailabilityBusy
	agent.CurrentTaskID = task.ID
	agent.LastActivityAt = now
	agent.UpdatedAt = now
	s.agents[agent.ID] = agent
	if s.store != nil {
		s.store.SaveAgent(agent)
	}
	if s.store != nil {
		s.store.SaveTask(task)
	}
	return task, nil
}

func (s *RemoteAgentOrchestrationService) GetTask(id string) (remoteAgentTask, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireTimedOutTaskLocked(strings.TrimSpace(id), time.Now().UTC())
	task, ok := s.tasks[strings.TrimSpace(id)]
	return task, ok
}

func remoteAgentTaskDeadline(task remoteAgentTask) (time.Time, bool) {
	if task.TimeoutSeconds <= 0 {
		return time.Time{}, false
	}
	startedAt := strings.TrimSpace(task.StartedAt)
	if startedAt == "" {
		startedAt = strings.TrimSpace(task.AssignedAt)
	}
	if startedAt == "" {
		startedAt = strings.TrimSpace(task.CreatedAt)
	}
	if startedAt == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.Add(time.Duration(task.TimeoutSeconds) * time.Second), true
}

func (s *RemoteAgentOrchestrationService) expireTimedOutTasksLocked(now time.Time) {
	for id := range s.tasks {
		s.expireTimedOutTaskLocked(id, now)
	}
}

func (s *RemoteAgentOrchestrationService) expireTimedOutTaskLocked(taskID string, now time.Time) bool {
	task, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return false
	}
	if task.State != remoteAgentTaskStateAssigned && task.State != remoteAgentTaskStateRunning {
		return false
	}
	deadline, ok := remoteAgentTaskDeadline(task)
	if !ok || deadline.After(now) {
		return false
	}
	completedAt := now.UTC().Format(time.RFC3339)
	task.State = remoteAgentTaskStateTimedOut
	task.CompletedAt = completedAt
	task.CancelledAt = ""
	task.LastTransitionAt = completedAt
	task.Result = &remoteAgentTaskResult{
		Summary:             "task exceeded timeout",
		Outcome:             string(remoteAgentTaskStateTimedOut),
		TranscriptEntries:   len(task.Transcript),
		OutputArtifactCount: len(task.OutputArtifacts),
	}
	s.updateTaskLocked(task)
	if agent, ok := s.agents[task.AgentID]; ok && agent.CurrentTaskID == task.ID {
		agent.CurrentTaskID = ""
		agent.Availability = remoteAgentAvailabilityIdle
		agent.LastActivityAt = completedAt
		agent.UpdatedAt = completedAt
		s.updateAgentLocked(agent)
	}
	return true
}

func (s *RemoteAgentOrchestrationService) updateTaskLocked(task remoteAgentTask) {
	s.tasks[task.ID] = task
	if s.store != nil {
		s.store.SaveTask(task)
	}
}

func (s *RemoteAgentOrchestrationService) updateAgentLocked(agent remoteAgent) {
	s.agents[agent.ID] = agent
	if s.store != nil {
		s.store.SaveAgent(agent)
	}
}

func (s *RemoteAgentOrchestrationService) transitionTaskLocked(taskID string, allowed []remoteAgentTaskState, next remoteAgentTaskState) (remoteAgentTask, error) {
	task, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return remoteAgentTask{}, &requestError{status: http.StatusNotFound, msg: "remote agent task not found"}
	}
	for _, state := range allowed {
		if task.State == state {
			now := nowRFC3339()
			task.State = next
			task.LastTransitionAt = now
			switch next {
			case remoteAgentTaskStateAssigned:
				task.AssignedAt = now
			case remoteAgentTaskStateRunning:
				task.StartedAt = now
			case remoteAgentTaskStateCompleted, remoteAgentTaskStateFailed, remoteAgentTaskStateTimedOut:
				task.CompletedAt = now
			case remoteAgentTaskStateCancelled:
				task.CancelledAt = now
			}
			s.updateTaskLocked(task)
			return task, nil
		}
	}
	return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: fmt.Sprintf("task cannot transition from %s to %s", task.State, next)}
}

func (s *RemoteAgentOrchestrationService) StartTask(taskID string) (remoteAgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireTimedOutTaskLocked(strings.TrimSpace(taskID), time.Now().UTC())
	task, err := s.transitionTaskLocked(taskID, []remoteAgentTaskState{remoteAgentTaskStateAssigned}, remoteAgentTaskStateRunning)
	if err != nil {
		return remoteAgentTask{}, err
	}
	if agent, ok := s.agents[task.AgentID]; ok {
		agent.Availability = remoteAgentAvailabilityBusy
		agent.CurrentTaskID = task.ID
		agent.LastActivityAt = task.StartedAt
		agent.UpdatedAt = task.StartedAt
		s.updateAgentLocked(agent)
	}
	return task, nil
}

func (s *RemoteAgentOrchestrationService) CancelTask(taskID string) (remoteAgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireTimedOutTaskLocked(strings.TrimSpace(taskID), time.Now().UTC())
	task, err := s.transitionTaskLocked(taskID, []remoteAgentTaskState{remoteAgentTaskStateAssigned, remoteAgentTaskStateRunning}, remoteAgentTaskStateCancelled)
	if err != nil {
		return remoteAgentTask{}, err
	}
	if agent, ok := s.agents[task.AgentID]; ok && agent.CurrentTaskID == task.ID {
		agent.CurrentTaskID = ""
		agent.Availability = remoteAgentAvailabilityIdle
		agent.LastActivityAt = task.CancelledAt
		agent.UpdatedAt = task.CancelledAt
		s.updateAgentLocked(agent)
	}
	return task, nil
}

func (s *RemoteAgentOrchestrationService) CompleteTask(taskID string, result remoteAgentTaskCompleteRequest) (remoteAgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expireTimedOutTaskLocked(strings.TrimSpace(taskID), time.Now().UTC())
	task, err := s.transitionTaskLocked(taskID, []remoteAgentTaskState{remoteAgentTaskStateAssigned, remoteAgentTaskStateRunning}, remoteAgentTaskStateCompleted)
	if err != nil {
		return remoteAgentTask{}, err
	}
	task.Result = &remoteAgentTaskResult{
		Summary:             strings.TrimSpace(result.Summary),
		Outcome:             strings.TrimSpace(result.Outcome),
		TranscriptEntries:   len(task.Transcript),
		OutputArtifactCount: len(task.OutputArtifacts),
	}
	if task.Result.Outcome == "" {
		task.Result.Outcome = "completed"
	}
	task.LastTransitionAt = task.CompletedAt
	s.updateTaskLocked(task)
	if agent, ok := s.agents[task.AgentID]; ok && agent.CurrentTaskID == task.ID {
		agent.CurrentTaskID = ""
		agent.Availability = remoteAgentAvailabilityIdle
		agent.LastActivityAt = task.CompletedAt
		agent.UpdatedAt = task.CompletedAt
		s.updateAgentLocked(agent)
	}
	return task, nil
}

func (s *RemoteAgentOrchestrationService) RetryTask(taskID string) (remoteAgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID = strings.TrimSpace(taskID)
	s.expireTimedOutTaskLocked(taskID, time.Now().UTC())
	task, ok := s.tasks[taskID]
	if !ok {
		return remoteAgentTask{}, &requestError{status: http.StatusNotFound, msg: "remote agent task not found"}
	}
	if !task.isTerminal() || task.State == remoteAgentTaskStateCompleted {
		return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: "task retry requires a failed, timed out, or cancelled task"}
	}
	agent, ok := s.agents[task.AgentID]
	if !ok {
		return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: "task retry requires a registered remote agent"}
	}
	if strings.TrimSpace(agent.CurrentTaskID) != "" && agent.CurrentTaskID != task.ID {
		return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: "remote agent is busy; wait for the active task to finish or cancel it"}
	}
	now := nowRFC3339()
	task.State = remoteAgentTaskStateAssigned
	task.Attempt++
	task.RetryCount++
	task.CurrentSession = agent.CurrentSession
	task.AssignedAt = now
	task.StartedAt = ""
	task.CompletedAt = ""
	task.CancelledAt = ""
	task.LastTransitionAt = now
	task.Result = nil
	task.Transcript = nil
	task.OutputArtifacts = nil
	s.updateTaskLocked(task)
	agent.CurrentTaskID = task.ID
	agent.Availability = remoteAgentAvailabilityBusy
	agent.LastActivityAt = now
	agent.UpdatedAt = now
	s.updateAgentLocked(agent)
	return task, nil
}

func (s *RemoteAgentOrchestrationService) AppendTranscript(taskID string, entry remoteAgentTranscriptEntry) (remoteAgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID = strings.TrimSpace(taskID)
	s.expireTimedOutTaskLocked(taskID, time.Now().UTC())
	task, ok := s.tasks[taskID]
	if !ok {
		return remoteAgentTask{}, &requestError{status: http.StatusNotFound, msg: "remote agent task not found"}
	}
	if task.State != remoteAgentTaskStateAssigned && task.State != remoteAgentTaskStateRunning {
		return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: "task transcript is immutable outside active states"}
	}
	entry.Text = strings.TrimSpace(entry.Text)
	entry.EventRef = strings.TrimSpace(entry.EventRef)
	entry.Sequence = int64(len(task.Transcript) + 1)
	if strings.TrimSpace(entry.At) == "" {
		entry.At = nowRFC3339()
	}
	task.Transcript = append(task.Transcript, entry)
	if task.Result != nil {
		task.Result.TranscriptEntries = len(task.Transcript)
	}
	task.LastTransitionAt = entry.At
	s.updateTaskLocked(task)
	return task, nil
}

func (s *RemoteAgentOrchestrationService) AddOutputArtifact(taskID string, req remoteAgentArtifactCreateRequest) (remoteAgentTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID = strings.TrimSpace(taskID)
	s.expireTimedOutTaskLocked(taskID, time.Now().UTC())
	task, ok := s.tasks[taskID]
	if !ok {
		return remoteAgentTask{}, &requestError{status: http.StatusNotFound, msg: "remote agent task not found"}
	}
	if task.State != remoteAgentTaskStateAssigned && task.State != remoteAgentTaskStateRunning {
		return remoteAgentTask{}, &requestError{status: http.StatusConflict, msg: "task artifacts are immutable outside active states"}
	}
	task.OutputArtifacts = append(task.OutputArtifacts, makeRemoteAgentArtifact(req))
	if task.Result != nil {
		task.Result.OutputArtifactCount = len(task.OutputArtifacts)
	}
	task.LastTransitionAt = nowRFC3339()
	s.updateTaskLocked(task)
	return task, nil
}

func (s *RemoteAgentOrchestrationService) TaskTranscript(taskID string) ([]remoteAgentTranscriptEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID = strings.TrimSpace(taskID)
	s.expireTimedOutTaskLocked(taskID, time.Now().UTC())
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, &requestError{status: http.StatusNotFound, msg: "remote agent task not found"}
	}
	out := make([]remoteAgentTranscriptEntry, len(task.Transcript))
	copy(out, task.Transcript)
	return out, nil
}

func (s *RemoteAgentOrchestrationService) TaskArtifacts(taskID string) ([]remoteAgentArtifactRef, []remoteAgentArtifactRef, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	taskID = strings.TrimSpace(taskID)
	s.expireTimedOutTaskLocked(taskID, time.Now().UTC())
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, nil, &requestError{status: http.StatusNotFound, msg: "remote agent task not found"}
	}
	inputs := make([]remoteAgentArtifactRef, len(task.InputArtifacts))
	outputs := make([]remoteAgentArtifactRef, len(task.OutputArtifacts))
	copy(inputs, task.InputArtifacts)
	copy(outputs, task.OutputArtifacts)
	return inputs, outputs, nil
}

func (a *API) handleRemoteAgentPoolsList(w http.ResponseWriter, _ *http.Request) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"remoteAgentPools": a.RemoteAgentOrchestration.ListPools()})
}

func (a *API) handleRemoteAgentPoolsCreate(w http.ResponseWriter, r *http.Request) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	var req remoteAgentPoolCreateRequest
	if err := readJSON(w, r, &req); err != nil {
		writeJSONError(w, err)
		return
	}
	pool, err := a.RemoteAgentOrchestration.CreatePool(req)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pool)
}

func (a *API) handleRemoteAgentsList(w http.ResponseWriter, _ *http.Request) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"remoteAgents": a.RemoteAgentOrchestration.ListAgents()})
}

func (a *API) handleRemoteAgentsCreate(w http.ResponseWriter, r *http.Request) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	var req remoteAgentCreateRequest
	if err := readJSON(w, r, &req); err != nil {
		writeJSONError(w, err)
		return
	}
	agent, err := a.RemoteAgentOrchestration.CreateAgent(req)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (a *API) handleRemoteAgentGet(w http.ResponseWriter, _ *http.Request, id string) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	agent, ok := a.RemoteAgentOrchestration.GetAgent(id)
	if !ok {
		writeError(w, http.StatusNotFound, "remote agent not found")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (a *API) handleRemoteAgentTasksList(w http.ResponseWriter, _ *http.Request) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"remoteAgentTasks": a.RemoteAgentOrchestration.ListTasks()})
}

func (a *API) handleRemoteAgentTasksCreate(w http.ResponseWriter, r *http.Request) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	var req remoteAgentTaskCreateRequest
	if err := readJSON(w, r, &req); err != nil {
		writeJSONError(w, err)
		return
	}
	task, err := a.RemoteAgentOrchestration.DispatchTask(principal(r.Context()), req)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, task)
}

func (a *API) handleRemoteAgentTaskGet(w http.ResponseWriter, _ *http.Request, id string) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	task, ok := a.RemoteAgentOrchestration.GetTask(id)
	if !ok {
		writeError(w, http.StatusNotFound, "remote agent task not found")
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleRemoteAgentTaskCancel(w http.ResponseWriter, _ *http.Request, id string) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	task, err := a.RemoteAgentOrchestration.CancelTask(id)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleRemoteAgentTaskRetry(w http.ResponseWriter, _ *http.Request, id string) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	task, err := a.RemoteAgentOrchestration.RetryTask(id)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (a *API) handleRemoteAgentTaskTranscriptGet(w http.ResponseWriter, _ *http.Request, id string) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	transcript, err := a.RemoteAgentOrchestration.TaskTranscript(id)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"taskId": id, "transcript": transcript})
}

func (a *API) handleRemoteAgentTaskArtifactsGet(w http.ResponseWriter, _ *http.Request, id string) {
	if a.RemoteAgentOrchestration == nil {
		writeError(w, http.StatusNotImplemented, "remote agent orchestration service not configured")
		return
	}
	inputs, outputs, err := a.RemoteAgentOrchestration.TaskArtifacts(id)
	if err != nil {
		writeJSONError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"taskId": id, "inputArtifacts": inputs, "outputArtifacts": outputs})
}
