package controllers

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/withakay/kocao/internal/auditlog"
	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	"github.com/withakay/kocao/internal/symphony/githubsource"
	"github.com/withakay/kocao/internal/symphony/runner"
	"github.com/withakay/kocao/internal/symphony/workflow"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clocktesting "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type stubSymphonySourceFactory struct {
	loader symphonySourceLoader
	err    error
}

func (s stubSymphonySourceFactory) New(token string) (symphonySourceLoader, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.loader, nil
}

type stubSymphonySourceLoader struct {
	snapshot githubsource.Snapshot
	err      error
	loads    int
}

func (s *stubSymphonySourceLoader) LoadProject(context.Context, githubsource.LoadOptions) (githubsource.Snapshot, error) {
	s.loads++
	if s.err != nil {
		return githubsource.Snapshot{}, s.err
	}
	return s.snapshot, nil
}

type stubWorkerExecutor struct {
	result symphonyWorkerResult
	err    error
	calls  int
}

func (s *stubWorkerExecutor) Execute(context.Context, symphonyWorkerExecution) (symphonyWorkerResult, error) {
	s.calls++
	if s.err != nil {
		return symphonyWorkerResult{}, s.err
	}
	return s.result, nil
}

func TestSymphonyProjectReconcile_ClaimsEligibleItemsAndSummarizesStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("demo")
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"},
		Data:       map[string][]byte{"token": []byte("ghp_test")},
	}
	loader := &stubSymphonySourceLoader{snapshot: githubsource.Snapshot{
		ResolvedFieldName:       "Status",
		UnsupportedRepositories: []string{"someone/else"},
		Candidates: []githubsource.CandidateItem{
			{ItemID: "PVT_item_1", Issue: githubIssue("withakay/kocao", 101, "First issue")},
			{ItemID: "PVT_item_2", Issue: githubIssue("withakay/kocao", 102, "Second issue")},
		},
		Skipped: []githubsource.SkippedItem{{ItemID: "PVT_item_9", Reason: githubsource.SkipReasonUnsupportedRepository, Message: "repo not allowed", Repository: "someone/else", ObservedAt: time.Unix(100, 0).UTC()}},
	}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}).WithObjects(project, secret).Build()
	audit := auditlog.New("", nil)
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(10, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}, Audit: audit}

	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if res.RequeueAfter != time.Minute {
		t.Fatalf("requeue after = %s, want %s", res.RequeueAfter, time.Minute)
	}
	if loader.loads != 1 {
		t.Fatalf("loads = %d, want 1", loader.loads)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.Status.Phase != operatorv1alpha1.SymphonyProjectPhaseReady {
		t.Fatalf("phase = %q", got.Status.Phase)
	}
	if len(got.Status.ActiveClaims) != 1 {
		t.Fatalf("active claims len = %d, want 1", len(got.Status.ActiveClaims))
	}
	if got.Status.ActiveClaims[0].ItemID != "PVT_item_1" {
		t.Fatalf("claimed item = %q", got.Status.ActiveClaims[0].ItemID)
	}
	if got.Status.ActiveClaims[0].RunRef.SessionName == "" || got.Status.ActiveClaims[0].RunRef.HarnessRunName == "" {
		t.Fatalf("expected run refs to be populated, got %#v", got.Status.ActiveClaims[0].RunRef)
	}
	if got.Status.EligibleItems != 2 || got.Status.RunningItems != 1 || got.Status.SkippedItems != 1 {
		t.Fatalf("counters = %#v", got.Status)
	}
	if got.Status.UnsupportedRepos[0] != "someone/else" {
		t.Fatalf("unsupported repos = %#v", got.Status.UnsupportedRepos)
	}

	var session operatorv1alpha1.Session
	if err := cl.Get(context.Background(), types.NamespacedName{Namespace: project.Namespace, Name: got.Status.ActiveClaims[0].RunRef.SessionName}, &session); err != nil {
		t.Fatalf("get session: %v", err)
	}
	if session.Spec.RepoURL != "https://github.com/withakay/kocao" {
		t.Fatalf("session repoURL = %q", session.Spec.RepoURL)
	}
	if session.Labels[LabelSymphonyProjectName] != project.Name || session.Labels[LabelSymphonyItemID] != "PVT_item_1" {
		t.Fatalf("session labels = %#v", session.Labels)
	}
	if session.Annotations[AnnotationAttachEnabled] != "false" {
		t.Fatalf("session attach annotation = %q", session.Annotations[AnnotationAttachEnabled])
	}

	var run operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), types.NamespacedName{Namespace: project.Namespace, Name: got.Status.ActiveClaims[0].RunRef.HarnessRunName}, &run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Spec.WorkspaceSessionName != session.Name {
		t.Fatalf("run workspaceSessionName = %q, want %q", run.Spec.WorkspaceSessionName, session.Name)
	}
	if run.Spec.RepoRevision != "main" {
		t.Fatalf("run repoRevision = %q, want main", run.Spec.RepoRevision)
	}
	if run.Labels[LabelGitHubIssueNumber] != "101" {
		t.Fatalf("run issue label = %#v", run.Labels)
	}
	if run.Annotations[AnnotationSymphonyIssueURL] == "" {
		t.Fatalf("run annotations = %#v", run.Annotations)
	}
	if conditionStatus(got.Status.Conditions, ConditionSource) != metav1.ConditionTrue {
		t.Fatalf("source condition = %#v", got.Status.Conditions)
	}
	if len(got.Finalizers) != 1 || got.Finalizers[0] != FinalizerName {
		t.Fatalf("finalizers = %#v", got.Finalizers)
	}
	events, err := audit.List(context.Background(), 10)
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if !hasAuditAction(events, "symphony.sync") {
		t.Fatalf("expected symphony.sync audit event, got %#v", events)
	}
	if !hasAuditAction(events, "symphony.claim") {
		t.Fatalf("expected symphony.claim audit event, got %#v", events)
	}
}

func TestSymphonyProjectReconcile_PausedSkipsSourcePolling(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("paused")
	project.Spec.Paused = true
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	loader := &stubSymphonySourceLoader{}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}).WithObjects(project, secret).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(20, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if loader.loads != 0 {
		t.Fatalf("loads = %d, want 0", loader.loads)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.Status.Phase != operatorv1alpha1.SymphonyProjectPhasePaused {
		t.Fatalf("phase = %q", got.Status.Phase)
	}
	if conditionStatus(got.Status.Conditions, ConditionLifecycle) != metav1.ConditionFalse {
		t.Fatalf("lifecycle condition = %#v", got.Status.Conditions)
	}
}

func TestSymphonyProjectReconcile_FailedRunBecomesRetryQueueEntry(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("retry")
	project.Spec.Runtime.MaxConcurrentItems = 2
	project.Status.RetryQueue = []operatorv1alpha1.SymphonyProjectRetryStatus{{
		ItemID:  "PVT_item_1",
		Issue:   operatorv1alpha1.SymphonyProjectIssueRefStatus{Repository: "withakay/kocao", Number: 201, Title: "Retry me", NodeID: "ISSUE_NODE", URL: "https://github.com/withakay/kocao/issues/1"},
		Attempt: 1,
		ReadyAt: &metav1.Time{Time: time.Unix(29, 0).UTC()},
	}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	session := &operatorv1alpha1.Session{TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"}, ObjectMeta: metav1.ObjectMeta{Name: symphonySessionName(project, operatorv1alpha1.SymphonyProjectClaimStatus{Issue: operatorv1alpha1.SymphonyProjectIssueRefStatus{Repository: "withakay/kocao", Number: 201}}), Namespace: "default"}, Spec: operatorv1alpha1.SessionSpec{RepoURL: "https://github.com/withakay/kocao"}}
	run := &operatorv1alpha1.HarnessRun{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "retry-run-1", Namespace: "default", Labels: map[string]string{
			LabelSymphonyProjectName: project.Name,
			LabelSymphonyProjectUID:  string(project.UID),
			LabelSymphonyItemID:      "PVT_item_1",
		}},
		Spec:   operatorv1alpha1.HarnessRunSpec{RepoURL: "https://github.com/withakay/kocao", Image: "ghcr.io/withakay/kocao-harness:latest"},
		Status: operatorv1alpha1.HarnessRunStatus{Phase: operatorv1alpha1.HarnessRunPhaseFailed, Conditions: []metav1.Condition{{Type: ConditionFailed, Reason: "PodFailed", Status: metav1.ConditionTrue}}},
	}
	loader := &stubSymphonySourceLoader{snapshot: githubsource.Snapshot{ResolvedFieldName: "Status", Candidates: []githubsource.CandidateItem{{ItemID: "PVT_item_1", Issue: githubIssue("withakay/kocao", 201, "Retry me")}}}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}, &operatorv1alpha1.Session{}).WithObjects(project, secret, session, run).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(30, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if len(got.Status.RetryQueue) != 1 {
		t.Fatalf("retry queue len = %d, want 1", len(got.Status.RetryQueue))
	}
	if got.Status.FailedItems != 1 {
		t.Fatalf("failed items = %d, want 1", got.Status.FailedItems)
	}
	if len(got.Status.RecentErrors) != 1 {
		t.Fatalf("recent errors len = %d, want 1", len(got.Status.RecentErrors))
	}
	if got.Status.RecentErrors[0].HarnessRunName != run.Name {
		t.Fatalf("recent error harness run = %q, want %q", got.Status.RecentErrors[0].HarnessRunName, run.Name)
	}
	if got.Status.RetryQueue[0].Reason != "PodFailed" {
		t.Fatalf("retry reason = %q", got.Status.RetryQueue[0].Reason)
	}
	if got.Status.RetryQueue[0].ReadyAt == nil || got.Status.RetryQueue[0].ReadyAt.Time.Sub(time.Unix(30, 0).UTC()) != time.Minute {
		t.Fatalf("retry readyAt = %#v", got.Status.RetryQueue[0].ReadyAt)
	}
}

func TestSymphonyProjectReconcile_WorkflowExecutionQueuesContinuationRetry(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("workflow-success")
	project.Spec.Repositories[0].LocalPath = "/tmp/kocao"
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	loader := &stubSymphonySourceLoader{snapshot: githubsource.Snapshot{ResolvedFieldName: "Status", Candidates: []githubsource.CandidateItem{{ItemID: "PVT_item_1", Issue: githubIssue("withakay/kocao", 501, "Workflow success")}}}}
	executor := &stubWorkerExecutor{result: symphonyWorkerResult{
		WorkflowPath:   "/tmp/kocao/WORKFLOW.md",
		WorkspacePath:  "/tmp/workspaces/PVT_item_1",
		SessionID:      "thread-1-turn-1",
		ThreadID:       "thread-1",
		TurnID:         "turn-1",
		ApprovalPolicy: defaultSymphonyApprovalPolicy,
		ThreadSandbox:  defaultSymphonyThreadSandbox,
		TurnSandbox:    defaultSymphonyTurnSandboxPolicy,
		LastEvent:      runner.EventTurnCompleted,
		LastMessage:    "workflow execution completed",
		InputTokens:    12,
		OutputTokens:   5,
		TotalTokens:    17,
		SecondsRunning: 3.5,
	}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}).WithObjects(project, secret).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(70, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}, WorkerExecutor: executor}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if executor.calls != 1 {
		t.Fatalf("worker calls = %d, want 1", executor.calls)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if len(got.Status.ActiveClaims) != 0 {
		t.Fatalf("active claims len = %d, want 0", len(got.Status.ActiveClaims))
	}
	if len(got.Status.RetryQueue) != 1 {
		t.Fatalf("retry queue len = %d, want 1", len(got.Status.RetryQueue))
	}
	if got.Status.RetryQueue[0].Reason != "Continuation" {
		t.Fatalf("retry reason = %q, want Continuation", got.Status.RetryQueue[0].Reason)
	}
	if got.Status.CompletedItems != 1 {
		t.Fatalf("completed items = %d, want 1", got.Status.CompletedItems)
	}
	if got.Status.TokenTotals.TotalTokens != 17 {
		t.Fatalf("token totals = %#v", got.Status.TokenTotals)
	}
	if len(got.Status.RecentEvents) != 1 || got.Status.RecentEvents[0].Event != runner.EventTurnCompleted {
		t.Fatalf("recent events = %#v", got.Status.RecentEvents)
	}

	var runList operatorv1alpha1.HarnessRunList
	if err := cl.List(context.Background(), &runList, client.InNamespace(project.Namespace)); err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runList.Items) != 1 {
		t.Fatalf("run count = %d, want 1", len(runList.Items))
	}
	run := runList.Items[0]
	if run.Status.Phase != operatorv1alpha1.HarnessRunPhaseSucceeded {
		t.Fatalf("run phase = %q, want succeeded", run.Status.Phase)
	}
	if run.Annotations[AnnotationSymphonySessionID] != "thread-1-turn-1" {
		t.Fatalf("run annotations = %#v", run.Annotations)
	}
	if run.Annotations[AnnotationSymphonyWorkflowPath] != "[redacted]" || run.Annotations[AnnotationSymphonyWorkspacePath] != "[redacted]" {
		t.Fatalf("expected redacted path annotations, got %#v", run.Annotations)
	}
	if run.Annotations[AnnotationSymphonyApprovalPolicy] != defaultSymphonyApprovalPolicy {
		t.Fatalf("approval policy annotation = %q", run.Annotations[AnnotationSymphonyApprovalPolicy])
	}
}

func TestSymphonyProjectReconcile_WorkflowFailureBecomesRetryAndRecentError(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("workflow-failure")
	project.Spec.Repositories[0].LocalPath = "/tmp/kocao"
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	loader := &stubSymphonySourceLoader{snapshot: githubsource.Snapshot{ResolvedFieldName: "Status", Candidates: []githubsource.CandidateItem{{ItemID: "PVT_item_1", Issue: githubIssue("withakay/kocao", 601, "Workflow failure")}}}}
	executor := &stubWorkerExecutor{err: &workflow.Error{Code: workflow.ErrCodeMissingWorkflowFile, Path: "/tmp/kocao/WORKFLOW.md", Err: errors.New("missing")}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}).WithObjects(project, secret).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(80, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}, WorkerExecutor: executor}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.Status.FailedItems != 1 {
		t.Fatalf("failed items = %d, want 1", got.Status.FailedItems)
	}
	if len(got.Status.RetryQueue) != 1 || got.Status.RetryQueue[0].Reason != "WorkflowMissing" {
		t.Fatalf("retry queue = %#v", got.Status.RetryQueue)
	}
	if len(got.Status.RecentErrors) != 1 || got.Status.RecentErrors[0].Reason != "WorkflowMissing" {
		t.Fatalf("recent errors = %#v", got.Status.RecentErrors)
	}
	if len(got.Status.RecentEvents) != 0 {
		t.Fatalf("recent events = %#v, want none", got.Status.RecentEvents)
	}
}

func TestSecureCodexConfigAppliesDefaults(t *testing.T) {
	cfg := secureCodexConfig(workflow.CodexConfig{})
	if cfg.ApprovalPolicy != defaultSymphonyApprovalPolicy {
		t.Fatalf("approval policy = %q", cfg.ApprovalPolicy)
	}
	if cfg.ThreadSandbox != defaultSymphonyThreadSandbox {
		t.Fatalf("thread sandbox = %q", cfg.ThreadSandbox)
	}
	if cfg.TurnSandboxPolicy != defaultSymphonyTurnSandboxPolicy {
		t.Fatalf("turn sandbox = %q", cfg.TurnSandboxPolicy)
	}
}

func TestEnforceWorkflowSecurityRejectsHooks(t *testing.T) {
	err := enforceWorkflowSecurity(workflow.Definition{Path: "/tmp/WORKFLOW.md"}, workflow.Config{Hooks: workflow.HooksConfig{BeforeRun: "echo hi"}})
	if err == nil {
		t.Fatal("expected workflow hook security error")
	}
	workflowErr, ok := err.(*workflow.Error)
	if !ok {
		t.Fatalf("expected workflow error, got %T", err)
	}
	if workflowErr.Code != workflow.ErrCodeWorkflowValidationError {
		t.Fatalf("workflow error code = %q", workflowErr.Code)
	}
}

func TestSanitizeTelemetryMessageRedactsSecrets(t *testing.T) {
	message := sanitizeTelemetryMessage("authorization=Bearer secret-token token=abc123 github_pat_deadbeef")
	if strings.Contains(message, "secret-token") || strings.Contains(message, "abc123") || strings.Contains(message, "github_pat_deadbeef") {
		t.Fatalf("expected secret-safe message, got %q", message)
	}
	if !strings.Contains(message, "[redacted]") {
		t.Fatalf("expected redaction marker, got %q", message)
	}
}

func TestDefaultSymphonyWorkerExecutorExecutesWorkflowContract(t *testing.T) {
	repoDir := t.TempDir()
	workflowDoc := strings.Join([]string{
		"---",
		"codex:",
		fmt.Sprintf("  command: %q", helperControllerCommand(t, "workflow-success")),
		"---",
		"You are working on {{.issue.title}} in {{.issue.repository}}.",
	}, "\n")
	if err := os.WriteFile(filepath.Join(repoDir, "WORKFLOW.md"), []byte(workflowDoc), 0o600); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	exec := defaultSymphonyWorkerExecutor{}
	result, err := exec.Execute(context.Background(), symphonyWorkerExecution{
		ProjectName: "demo",
		Repository:  operatorv1alpha1.SymphonyProjectRepositorySpec{Owner: "withakay", Name: "kocao", LocalPath: repoDir},
		Claim:       operatorv1alpha1.SymphonyProjectClaimStatus{Attempt: 1},
		Issue:       githubIssue("withakay/kocao", 777, "Workflow integration"),
		Title:       "withakay/kocao#777: Workflow integration",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if result.SessionID != "thread-1-turn-1" {
		t.Fatalf("session id = %q", result.SessionID)
	}
	if result.ApprovalPolicy != defaultSymphonyApprovalPolicy {
		t.Fatalf("approval policy = %q", result.ApprovalPolicy)
	}
	if result.LastEvent != runner.EventTurnCompleted {
		t.Fatalf("last event = %q", result.LastEvent)
	}
	if result.TotalTokens != 15 {
		t.Fatalf("token totals = %d", result.TotalTokens)
	}
}

func hasAuditAction(events []auditlog.Event, action string) bool {
	for _, event := range events {
		if event.Action == action {
			return true
		}
	}
	return false
}

func helperControllerCommand(t *testing.T, mode string) string {
	t.Helper()
	return fmt.Sprintf("GO_WANT_HELPER_PROCESS=1 %s -test.run=TestControllerWorkerHelperProcess -- %s", os.Args[0], mode)
}

func TestControllerWorkerHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || len(os.Args) < 4 {
		return
	}
	mode := os.Args[3]
	reader := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()
	for reader.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(reader.Bytes(), &msg); err != nil {
			continue
		}
		method := strings.TrimSpace(fmt.Sprint(msg["method"]))
		switch method {
		case "initialize":
			writeControllerHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"serverInfo": map[string]any{"name": "fake"}}})
		case "initialized":
		case "thread/start":
			writeControllerHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"thread": map[string]any{"id": "thread-1"}}})
		case "turn/start":
			if mode == "workflow-success" {
				writeControllerHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "id": msg["id"], "result": map[string]any{"turn": map[string]any{"id": "turn-1"}}})
				writeControllerHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "method": "thread/tokenUsage/updated", "params": map[string]any{"total_token_usage": map[string]any{"input_tokens": 10, "output_tokens": 5, "total_tokens": 15}}})
				writeControllerHelperJSON(writer, map[string]any{"jsonrpc": "2.0", "method": "turn/completed", "params": map[string]any{"turnId": "turn-1", "summary": "completed"}})
			}
		}
	}
	os.Exit(0)
}

func writeControllerHelperJSON(w *bufio.Writer, payload map[string]any) {
	b, _ := json.Marshal(payload)
	_, _ = w.Write(append(b, '\n'))
	_ = w.Flush()
}

func TestSymphonyProjectReconcile_ReadyRetryCreatesNextAttemptRunAndReusesSession(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("reuse")
	project.Spec.Runtime.MaxConcurrentItems = 1
	project.Status.RetryQueue = []operatorv1alpha1.SymphonyProjectRetryStatus{{
		ItemID:  "PVT_item_1",
		Issue:   operatorv1alpha1.SymphonyProjectIssueRefStatus{Repository: "withakay/kocao", Number: 301, Title: "Retry me", NodeID: "ISSUE_NODE", URL: "https://github.com/withakay/kocao/issues/1"},
		Attempt: 1,
		ReadyAt: &metav1.Time{Time: time.Unix(49, 0).UTC()},
	}}
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	claim := operatorv1alpha1.SymphonyProjectClaimStatus{Issue: operatorv1alpha1.SymphonyProjectIssueRefStatus{Repository: "withakay/kocao", Number: 301}, Attempt: 2}
	sessionName := symphonySessionName(project, claim)
	session := &operatorv1alpha1.Session{TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"}, ObjectMeta: metav1.ObjectMeta{Name: sessionName, Namespace: "default"}, Spec: operatorv1alpha1.SessionSpec{RepoURL: "https://github.com/withakay/kocao"}}
	loader := &stubSymphonySourceLoader{snapshot: githubsource.Snapshot{ResolvedFieldName: "Status", Candidates: []githubsource.CandidateItem{{ItemID: "PVT_item_1", Issue: githubIssue("withakay/kocao", 301, "Retry me")}}}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}).WithObjects(project, secret, session).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(50, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if len(got.Status.ActiveClaims) != 1 {
		t.Fatalf("active claims len = %d, want 1", len(got.Status.ActiveClaims))
	}
	if got.Status.ActiveClaims[0].Attempt != 2 {
		t.Fatalf("attempt = %d, want 2", got.Status.ActiveClaims[0].Attempt)
	}
	if got.Status.ActiveClaims[0].RunRef.SessionName != sessionName {
		t.Fatalf("session ref = %q, want %q", got.Status.ActiveClaims[0].RunRef.SessionName, sessionName)
	}
	var run operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), types.NamespacedName{Namespace: project.Namespace, Name: got.Status.ActiveClaims[0].RunRef.HarnessRunName}, &run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Spec.WorkspaceSessionName != sessionName {
		t.Fatalf("run workspaceSessionName = %q, want %q", run.Spec.WorkspaceSessionName, sessionName)
	}
	if run.Labels[LabelSymphonyItemID] != "PVT_item_1" {
		t.Fatalf("run labels = %#v", run.Labels)
	}
}

func TestSymphonyProjectReconcile_ReleasesActiveRunWhenItemLeavesBoard(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("released")
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	sessionName := symphonySessionName(project, operatorv1alpha1.SymphonyProjectClaimStatus{Issue: operatorv1alpha1.SymphonyProjectIssueRefStatus{Repository: "withakay/kocao", Number: 401}})
	session := &operatorv1alpha1.Session{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      sessionName,
			Namespace: "default",
			Labels: map[string]string{
				LabelSymphonyProjectName: project.Name,
				LabelSymphonyProjectUID:  string(project.UID),
				LabelSymphonyItemID:      "PVT_item_1",
			},
		},
		Spec: operatorv1alpha1.SessionSpec{RepoURL: "https://github.com/withakay/kocao"},
	}
	run := &operatorv1alpha1.HarnessRun{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{Name: "active-run", Namespace: "default", Labels: map[string]string{
			LabelSymphonyProjectName: project.Name,
			LabelSymphonyProjectUID:  string(project.UID),
			LabelSymphonyItemID:      "PVT_item_1",
		}},
		Spec:   operatorv1alpha1.HarnessRunSpec{WorkspaceSessionName: sessionName, RepoURL: "https://github.com/withakay/kocao", Image: "ghcr.io/withakay/kocao-harness:latest"},
		Status: operatorv1alpha1.HarnessRunStatus{Phase: operatorv1alpha1.HarnessRunPhaseRunning},
	}
	issue := githubIssue("withakay/kocao", 401, "Done")
	loader := &stubSymphonySourceLoader{snapshot: githubsource.Snapshot{ResolvedFieldName: "Status", Skipped: []githubsource.SkippedItem{{ItemID: "PVT_item_1", Issue: &issue, Repository: "withakay/kocao", Reason: githubsource.SkipReasonInactiveState, Message: "item left active states", ObservedAt: time.Unix(60, 0).UTC()}}}}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}).WithObjects(project, secret, session, run).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(60, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if len(got.Status.ActiveClaims) != 0 {
		t.Fatalf("active claims len = %d, want 0", len(got.Status.ActiveClaims))
	}
	if got.Status.RunningItems != 0 {
		t.Fatalf("running items = %d, want 0", got.Status.RunningItems)
	}

	var deleted operatorv1alpha1.HarnessRun
	if err := cl.Get(context.Background(), types.NamespacedName{Namespace: project.Namespace, Name: run.Name}, &deleted); !apierrors.IsNotFound(err) {
		t.Fatalf("expected active run deletion, got err=%v", err)
	}

	var deletedSession operatorv1alpha1.Session
	if err := cl.Get(context.Background(), types.NamespacedName{Namespace: project.Namespace, Name: sessionName}, &deletedSession); !apierrors.IsNotFound(err) {
		t.Fatalf("expected session deletion, got err=%v", err)
	}
}

func TestSymphonyProjectReconcile_SourceFailuresSurfaceErrorStatus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = operatorv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	project := newSymphonyProject("errored")
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
	loader := &stubSymphonySourceLoader{err: errors.New("github unavailable")}
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}).WithObjects(project, secret).Build()
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(40, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}}

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(project)})
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	var got operatorv1alpha1.SymphonyProject
	if err := cl.Get(context.Background(), client.ObjectKeyFromObject(project), &got); err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got.Status.Phase != operatorv1alpha1.SymphonyProjectPhaseError {
		t.Fatalf("phase = %q", got.Status.Phase)
	}
	if got.Status.LastError == "" {
		t.Fatal("expected last error to be set")
	}
	if conditionStatus(got.Status.Conditions, ConditionSource) != metav1.ConditionFalse {
		t.Fatalf("conditions = %#v", got.Status.Conditions)
	}
}

func newSymphonyProject(name string) *operatorv1alpha1.SymphonyProject {
	return &operatorv1alpha1.SymphonyProject{
		TypeMeta:   metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "SymphonyProject"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID("uid-" + name)},
		Spec: operatorv1alpha1.SymphonyProjectSpec{
			Source: operatorv1alpha1.SymphonyProjectSourceSpec{
				Project:        operatorv1alpha1.GitHubProjectRef{Owner: "withakay", Number: 7},
				TokenSecretRef: operatorv1alpha1.SecretKeyRef{Name: "github-token"},
				ActiveStates:   []string{"Todo"},
				TerminalStates: []string{"Done"},
			},
			Repositories: []operatorv1alpha1.SymphonyProjectRepositorySpec{{Owner: "withakay", Name: "kocao", RepoURL: "https://github.com/withakay/kocao"}},
			Runtime: operatorv1alpha1.SymphonyProjectRuntimeSpec{
				Image:               "ghcr.io/withakay/kocao-harness:latest",
				MaxConcurrentItems:  1,
				DefaultRepoRevision: "main",
			},
		},
	}
}

func githubIssue(repository string, number int64, title string) githubsource.Issue {
	return githubsource.Issue{Repository: repository, Number: number, Title: title, NodeID: "ISSUE_NODE", URL: "https://github.com/" + repository + "/issues/1"}
}

func conditionStatus(conditions []metav1.Condition, conditionType string) metav1.ConditionStatus {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status
		}
	}
	return metav1.ConditionUnknown
}
