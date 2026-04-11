package controllers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
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
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type symphonySourceLoader interface {
	LoadProject(context.Context, githubsource.LoadOptions) (githubsource.Snapshot, error)
}

type symphonySourceFactory interface {
	New(token string) (symphonySourceLoader, error)
}

type defaultSymphonySourceFactory struct{}

func (defaultSymphonySourceFactory) New(token string) (symphonySourceLoader, error) {
	return githubsource.NewClient(token, githubsource.Options{})
}

type symphonyWorkerExecution struct {
	ProjectName string
	Repository  operatorv1alpha1.SymphonyProjectRepositorySpec
	Claim       operatorv1alpha1.SymphonyProjectClaimStatus
	Issue       githubsource.Issue
	Title       string
}

type symphonyWorkerResult struct {
	WorkflowPath   string
	WorkspacePath  string
	SessionID      string
	ThreadID       string
	TurnID         string
	ApprovalPolicy string
	ThreadSandbox  string
	TurnSandbox    string
	LastEvent      string
	LastMessage    string
	InputTokens    int64
	OutputTokens   int64
	TotalTokens    int64
	SecondsRunning float64
}

type symphonyWorkerExecutor interface {
	Execute(context.Context, symphonyWorkerExecution) (symphonyWorkerResult, error)
}

type defaultSymphonyWorkerExecutor struct{}

const (
	defaultSymphonyApprovalPolicy    = "untrusted"
	defaultSymphonyThreadSandbox     = "workspace-write"
	defaultSymphonyTurnSandboxPolicy = "workspace-write"
)

var sensitiveTelemetryPattern = regexp.MustCompile(`(?i)(bearer\s+[a-z0-9._-]+|github_pat_[a-z0-9_]+|gh[pousr]_[a-z0-9]+|sk-[a-z0-9_-]+|token\s*[=:]\s*\S+|authorization\s*[=:]\s*\S+|password\s*[=:]\s*\S+)`)

func (defaultSymphonyWorkerExecutor) Execute(ctx context.Context, execReq symphonyWorkerExecution) (symphonyWorkerResult, error) {
	repoPath := strings.TrimSpace(execReq.Repository.LocalPath)
	if repoPath == "" {
		return symphonyWorkerResult{}, nil
	}
	resolvedRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return symphonyWorkerResult{}, err
	}
	workflowPath := workflow.ResolvePath(resolvedRepoPath, execReq.Repository.WorkflowPath)
	def, err := workflow.Load(workflowPath)
	if err != nil {
		return symphonyWorkerResult{}, err
	}
	cfg, err := def.TypedConfig(os.Getenv)
	if err != nil {
		return symphonyWorkerResult{}, err
	}
	if err := enforceWorkflowSecurity(def, cfg); err != nil {
		return symphonyWorkerResult{}, err
	}
	codexCfg := secureCodexConfig(cfg.Codex)
	prompt, err := def.Render(issueTemplateData(execReq.Issue), int32PtrToIntPtr(execReq.Claim.Attempt))
	if err != nil {
		return symphonyWorkerResult{}, err
	}
	workspacePath := filepath.Join(os.TempDir(), "kocao-symphony-workspaces", sanitizeDNSLabel(execReq.ProjectName), sanitizeDNSLabel(execReq.Issue.Repository+"-"+strconv.FormatInt(execReq.Issue.Number, 10)))
	if err := os.MkdirAll(workspacePath, 0o755); err != nil {
		return symphonyWorkerResult{}, err
	}
	started := time.Now().UTC()
	runResult, err := runner.Run(ctx, runner.Options{Workspace: workspacePath, Title: execReq.Title, Prompts: []string{prompt}, Config: codexCfg})
	if err != nil {
		return symphonyWorkerResult{}, err
	}
	lastTurnID := ""
	if len(runResult.Turns) != 0 {
		lastTurnID = runResult.Turns[len(runResult.Turns)-1].TurnID
	}
	return symphonyWorkerResult{
		WorkflowPath:   workflowPath,
		WorkspacePath:  workspacePath,
		SessionID:      sessionID(runResult.ThreadID, lastTurnID),
		ThreadID:       runResult.ThreadID,
		TurnID:         lastTurnID,
		ApprovalPolicy: codexCfg.ApprovalPolicy,
		ThreadSandbox:  codexCfg.ThreadSandbox,
		TurnSandbox:    codexCfg.TurnSandboxPolicy,
		LastEvent:      runResult.LastEvent,
		LastMessage:    sanitizeTelemetryMessage(runResult.LastMessage),
		InputTokens:    int64(runResult.Usage.InputTokens),
		OutputTokens:   int64(runResult.Usage.OutputTokens),
		TotalTokens:    int64(runResult.Usage.TotalTokens),
		SecondsRunning: time.Since(started).Seconds(),
	}, nil
}

type SymphonyProjectReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	Clock          clock.Clock
	SourceFactory  symphonySourceFactory
	WorkerExecutor symphonyWorkerExecutor
	Audit          *auditlog.Store
}

func (r *SymphonyProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var project operatorv1alpha1.SymphonyProject
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	clk := r.Clock
	if clk == nil {
		clk = clock.RealClock{}
	}
	sourceFactory := r.SourceFactory
	if sourceFactory == nil {
		sourceFactory = defaultSymphonySourceFactory{}
	}
	workerExecutor := r.WorkerExecutor
	if workerExecutor == nil {
		workerExecutor = defaultSymphonyWorkerExecutor{}
	}

	now := metav1.NewTime(clk.Now().UTC())
	updated := project.DeepCopy()
	updated.ApplyDefaults()
	changedMeta := false
	changedStatus := false

	if updated.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(updated, FinalizerName) {
			controllerutil.AddFinalizer(updated, FinalizerName)
			changedMeta = true
		}
	} else {
		if controllerutil.ContainsFinalizer(updated, FinalizerName) {
			controllerutil.RemoveFinalizer(updated, FinalizerName)
			changedMeta = true
		}
		if changedMeta {
			metaUpdated := updated.DeepCopy()
			metaUpdated.Status = project.Status
			if err := r.Patch(ctx, metaUpdated, client.MergeFrom(&project)); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if err := updated.Validate(); err != nil {
		r.setConfigError(updated, now, err)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus)
	}

	setCondition(&updated.Status.Conditions, metav1.Condition{Type: ConditionConfig, Status: metav1.ConditionTrue, Reason: "Valid", Message: "symphony project configuration is valid", LastTransitionTime: now})
	updated.Status.ObservedGeneration = updated.Generation

	pollInterval := time.Duration(updated.Spec.Source.PollIntervalSec) * time.Second
	if updated.Spec.Paused {
		updated.Status.Phase = operatorv1alpha1.SymphonyProjectPhasePaused
		updated.Status.LastError = ""
		updated.Status.NextSyncTime = &metav1.Time{Time: now.Time.Add(pollInterval)}
		setCondition(&updated.Status.Conditions, metav1.Condition{Type: ConditionSource, Status: metav1.ConditionFalse, Reason: "Paused", Message: "symphony project polling is paused", LastTransitionTime: now})
		setCondition(&updated.Status.Conditions, metav1.Condition{Type: ConditionLifecycle, Status: metav1.ConditionFalse, Reason: "Paused", Message: "symphony orchestration is paused", LastTransitionTime: now})
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}

	token, err := r.loadGitHubToken(ctx, updated)
	if err != nil {
		r.setConfigError(updated, now, err)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}

	loader, err := sourceFactory.New(token)
	if err != nil {
		r.setSourceError(updated, now, fmt.Errorf("build github source client: %w", err), pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}

	snapshot, err := loader.LoadProject(ctx, githubsource.LoadOptions{
		Project:        updated.Spec.Source.Project,
		FieldName:      updated.Spec.Source.FieldName,
		ActiveStates:   updated.Spec.Source.ActiveStates,
		TerminalStates: updated.Spec.Source.TerminalStates,
		Repositories:   updated.Spec.Repositories,
	})
	if err != nil {
		r.setSourceError(updated, now, err, pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}

	runsByItem, err := r.listProjectRuns(ctx, updated)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.releaseInactiveRuns(ctx, updated, snapshot, runsByItem); err != nil {
		r.setLifecycleError(updated, now, err, pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}

	reconcileProjectRuntime(updated, snapshot, runsByItem, now.Time)
	if err := r.materializeActiveClaims(ctx, updated, runsByItem, now.Time); err != nil {
		r.setLifecycleError(updated, now, err, pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}
	runsByItem, err = r.listProjectRuns(ctx, updated)
	if err != nil {
		r.setLifecycleError(updated, now, err, pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}
	if err := r.executeRunnableClaims(ctx, updated, runsByItem, workerExecutor, now.Time); err != nil {
		r.setLifecycleError(updated, now, err, pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}
	runsByItem, err = r.listProjectRuns(ctx, updated)
	if err != nil {
		r.setLifecycleError(updated, now, err, pollInterval)
		changedStatus = true
		return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
	}
	reconcileProjectRuntime(updated, snapshot, runsByItem, now.Time)
	updated.Status.Phase = operatorv1alpha1.SymphonyProjectPhaseReady
	updated.Status.ObservedGeneration = updated.Generation
	updated.Status.ResolvedFieldName = snapshot.ResolvedFieldName
	updated.Status.LastSyncTime = &now
	updated.Status.LastSuccessfulSync = &now
	updated.Status.NextSyncTime = &metav1.Time{Time: now.Time.Add(pollInterval)}
	updated.Status.LastError = ""
	setCondition(&updated.Status.Conditions, metav1.Condition{Type: ConditionSource, Status: metav1.ConditionTrue, Reason: "SyncSucceeded", Message: "github project snapshot loaded", LastTransitionTime: now})
	setCondition(&updated.Status.Conditions, metav1.Condition{Type: ConditionLifecycle, Status: metav1.ConditionTrue, Reason: "Polling", Message: "symphony orchestration is polling for work", LastTransitionTime: now})
	changedStatus = true

	return r.commit(ctx, &project, updated, changedMeta, changedStatus, ctrl.Result{RequeueAfter: pollInterval})
}

func (r *SymphonyProjectReconciler) materializeActiveClaims(ctx context.Context, project *operatorv1alpha1.SymphonyProject, runsByItem map[string]operatorv1alpha1.HarnessRun, now time.Time) error {
	repositories := map[string]operatorv1alpha1.SymphonyProjectRepositorySpec{}
	for _, repo := range project.Spec.Repositories {
		repositories[repo.RepositoryKey()] = repo
	}
	for i := range project.Status.ActiveClaims {
		claim := &project.Status.ActiveClaims[i]
		if strings.TrimSpace(claim.ItemID) == "" {
			continue
		}
		desiredRunName := symphonyRunName(project, *claim)
		if existingRun, ok := runsByItem[claim.ItemID]; ok && existingRun.Name == desiredRunName {
			if repo, ok := repositories[repositoryKeyFromStatus(claim.Issue)]; ok {
				session, err := r.ensureClaimSession(ctx, project, repo, *claim)
				if err != nil {
					return err
				}
				if _, err := r.ensureClaimRun(ctx, project, session, repo, *claim); err != nil {
					return err
				}
			}
			project.Status.ActiveClaims[i] = buildClaimStatus(githubsource.CandidateItem{ItemID: claim.ItemID, Issue: githubsource.Issue{Repository: claim.Issue.Repository, Number: claim.Issue.Number, NodeID: claim.Issue.NodeID, URL: claim.Issue.URL, Title: claim.Issue.Title}}, existingRun, *claim, now)
			continue
		}

		repo, ok := repositories[repositoryKeyFromStatus(claim.Issue)]
		if !ok {
			continue
		}
		session, err := r.ensureClaimSession(ctx, project, repo, *claim)
		if err != nil {
			return err
		}
		run, err := r.ensureClaimRun(ctx, project, session, repo, *claim)
		if err != nil {
			return err
		}
		claim.RunRef = operatorv1alpha1.SymphonyProjectRunRefStatus{SessionName: session.Name, HarnessRunName: run.Name}
		claim.Phase = firstNonEmpty(string(run.Status.Phase), string(operatorv1alpha1.HarnessRunPhasePending), claim.Phase)
		lastUpdated := metav1.NewTime(now)
		claim.LastUpdatedTime = &lastUpdated
	}
	return nil
}

func (r *SymphonyProjectReconciler) executeRunnableClaims(ctx context.Context, project *operatorv1alpha1.SymphonyProject, runsByItem map[string]operatorv1alpha1.HarnessRun, workerExecutor symphonyWorkerExecutor, now time.Time) error {
	repositories := map[string]operatorv1alpha1.SymphonyProjectRepositorySpec{}
	for _, repo := range project.Spec.Repositories {
		repositories[repo.RepositoryKey()] = repo
	}
	for _, claim := range project.Status.ActiveClaims {
		repo, ok := repositories[repositoryKeyFromStatus(claim.Issue)]
		if !ok || strings.TrimSpace(repo.LocalPath) == "" {
			continue
		}
		run, ok := runsByItem[claim.ItemID]
		if !ok || run.Name != symphonyRunName(project, claim) {
			continue
		}
		switch run.Status.Phase {
		case operatorv1alpha1.HarnessRunPhaseSucceeded, operatorv1alpha1.HarnessRunPhaseFailed:
			continue
		}
		result, execErr := workerExecutor.Execute(ctx, symphonyWorkerExecution{
			ProjectName: project.Name,
			Repository:  repo,
			Claim:       claim,
			Issue:       githubsource.Issue{Repository: claim.Issue.Repository, Number: claim.Issue.Number, NodeID: claim.Issue.NodeID, URL: claim.Issue.URL, Title: claim.Issue.Title},
			Title:       fmt.Sprintf("%s#%d: %s", claim.Issue.Repository, claim.Issue.Number, claim.Issue.Title),
		})
		if err := r.applyWorkerOutcome(ctx, &run, result, execErr, now); err != nil {
			return err
		}
	}
	return nil
}

func (r *SymphonyProjectReconciler) ensureClaimSession(ctx context.Context, project *operatorv1alpha1.SymphonyProject, repo operatorv1alpha1.SymphonyProjectRepositorySpec, claim operatorv1alpha1.SymphonyProjectClaimStatus) (*operatorv1alpha1.Session, error) {
	name := symphonySessionName(project, claim)
	desired := &operatorv1alpha1.Session{}
	err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: name}, desired)
	if err == nil {
		updated := desired.DeepCopy()
		if updated.Annotations == nil {
			updated.Annotations = map[string]string{}
		}
		changed := false
		for key, value := range map[string]string{
			AnnotationAttachEnabled:     "true",
			AnnotationEgressMode:        claimEgressMode(project.Spec.Runtime, repo),
			AnnotationSymphonyIssueURL:  claim.Issue.URL,
			AnnotationSymphonyIssueNode: claim.Issue.NodeID,
		} {
			if updated.Annotations[key] != value {
				updated.Annotations[key] = value
				changed = true
			}
		}
		if updated.Spec.DisplayName != symphonySessionDisplayName(claim) {
			updated.Spec.DisplayName = symphonySessionDisplayName(claim)
			changed = true
		}
		if updated.Spec.RepoURL != repositoryRepoURL(repo) {
			updated.Spec.RepoURL = repositoryRepoURL(repo)
			changed = true
		}
		if changed {
			if err := r.Patch(ctx, updated, client.MergeFrom(desired)); err != nil {
				return nil, err
			}
			return updated, nil
		}
		return desired, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	desired = &operatorv1alpha1.Session{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "Session"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: project.Namespace,
			Labels:    symphonyObjectLabels(project, claim),
			Annotations: map[string]string{
				AnnotationAttachEnabled:     "true",
				AnnotationEgressMode:        claimEgressMode(project.Spec.Runtime, repo),
				AnnotationSymphonyIssueURL:  claim.Issue.URL,
				AnnotationSymphonyIssueNode: claim.Issue.NodeID,
			},
		},
		Spec: operatorv1alpha1.SessionSpec{
			DisplayName: symphonySessionDisplayName(claim),
			RepoURL:     repositoryRepoURL(repo),
		},
	}
	if err := controllerutil.SetControllerReference(project, desired, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, desired); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: name}, desired); err != nil {
				return nil, err
			}
			return desired, nil
		}
		return nil, err
	}
	return desired, nil
}

func (r *SymphonyProjectReconciler) ensureClaimRun(ctx context.Context, project *operatorv1alpha1.SymphonyProject, session *operatorv1alpha1.Session, repo operatorv1alpha1.SymphonyProjectRepositorySpec, claim operatorv1alpha1.SymphonyProjectClaimStatus) (*operatorv1alpha1.HarnessRun, error) {
	name := symphonyRunName(project, claim)
	run := &operatorv1alpha1.HarnessRun{}
	err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: name}, run)
	if err == nil {
		updated := run.DeepCopy()
		changed := false
		if updated.Spec.WorkspaceSessionName != session.Name {
			updated.Spec.WorkspaceSessionName = session.Name
			changed = true
		}
		for _, apply := range []func(*operatorv1alpha1.HarnessRunSpec) bool{
			func(spec *operatorv1alpha1.HarnessRunSpec) bool {
				if spec.RepoURL != repositoryRepoURL(repo) {
					spec.RepoURL = repositoryRepoURL(repo)
					return true
				}
				return false
			},
			func(spec *operatorv1alpha1.HarnessRunSpec) bool {
				if spec.RepoRevision != claimRepoRevision(project.Spec.Runtime, repo) {
					spec.RepoRevision = claimRepoRevision(project.Spec.Runtime, repo)
					return true
				}
				return false
			},
			func(spec *operatorv1alpha1.HarnessRunSpec) bool {
				if spec.Image != project.Spec.Runtime.Image {
					spec.Image = project.Spec.Runtime.Image
					return true
				}
				return false
			},
			func(spec *operatorv1alpha1.HarnessRunSpec) bool {
				if spec.EgressMode != claimEgressMode(project.Spec.Runtime, repo) {
					spec.EgressMode = claimEgressMode(project.Spec.Runtime, repo)
					return true
				}
				return false
			},
		} {
			if apply(&updated.Spec) {
				changed = true
			}
		}
		if changed {
			if err := r.Patch(ctx, updated, client.MergeFrom(run)); err != nil {
				return nil, err
			}
			return updated, nil
		}
		return run, nil
	}
	if !apierrors.IsNotFound(err) {
		return nil, err
	}
	run = &operatorv1alpha1.HarnessRun{
		TypeMeta: metav1.TypeMeta{APIVersion: operatorv1alpha1.GroupVersion.String(), Kind: "HarnessRun"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   project.Namespace,
			Labels:      symphonyObjectLabels(project, claim),
			Annotations: symphonyObjectAnnotations(claim),
		},
		Spec: operatorv1alpha1.HarnessRunSpec{
			WorkspaceSessionName:    session.Name,
			RepoURL:                 repositoryRepoURL(repo),
			RepoRevision:            claimRepoRevision(project.Spec.Runtime, repo),
			Image:                   project.Spec.Runtime.Image,
			Command:                 append([]string(nil), project.Spec.Runtime.Command...),
			Args:                    append([]string(nil), project.Spec.Runtime.Args...),
			WorkingDir:              project.Spec.Runtime.WorkingDir,
			Env:                     append([]operatorv1alpha1.EnvVar(nil), project.Spec.Runtime.Env...),
			GitAuth:                 repo.GitAuth,
			AgentAuth:               repo.AgentAuth,
			EgressMode:              claimEgressMode(project.Spec.Runtime, repo),
			TTLSecondsAfterFinished: project.Spec.Runtime.TTLSecondsAfterFinished,
		},
	}
	if err := controllerutil.SetControllerReference(project, run, r.Scheme); err != nil {
		return nil, err
	}
	if err := r.Create(ctx, run); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: name}, run); err != nil {
				return nil, err
			}
			return run, nil
		}
		return nil, err
	}
	return run, nil
}

func (r *SymphonyProjectReconciler) applyWorkerOutcome(ctx context.Context, run *operatorv1alpha1.HarnessRun, result symphonyWorkerResult, execErr error, now time.Time) error {
	original := run.DeepCopy()
	updated := run.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	if result.SessionID != "" {
		updated.Annotations[AnnotationSymphonySessionID] = result.SessionID
	}
	if result.ThreadID != "" {
		updated.Annotations[AnnotationSymphonyThreadID] = result.ThreadID
	}
	if result.TurnID != "" {
		updated.Annotations[AnnotationSymphonyTurnID] = result.TurnID
	}
	if result.LastEvent != "" {
		updated.Annotations[AnnotationSymphonyLastEvent] = result.LastEvent
	}
	if sanitized := sanitizeTelemetryMessage(result.LastMessage); sanitized != "" {
		updated.Annotations[AnnotationSymphonyLastMessage] = sanitized
	}
	if result.WorkflowPath != "" || result.WorkspacePath != "" {
		updated.Annotations[AnnotationSymphonyWorkflowPath] = "[redacted]"
		updated.Annotations[AnnotationSymphonyWorkspacePath] = "[redacted]"
	}
	if policy, sandbox, turnSandbox := resultSecurityAnnotations(result); policy != "" || sandbox != "" || turnSandbox != "" {
		if policy != "" {
			updated.Annotations[AnnotationSymphonyApprovalPolicy] = policy
		}
		if sandbox != "" {
			updated.Annotations[AnnotationSymphonyThreadSandbox] = sandbox
		}
		if turnSandbox != "" {
			updated.Annotations[AnnotationSymphonyTurnSandbox] = turnSandbox
		}
	}
	if result.InputTokens > 0 {
		updated.Annotations[AnnotationSymphonyInputTokens] = strconv.FormatInt(result.InputTokens, 10)
	}
	if result.OutputTokens > 0 {
		updated.Annotations[AnnotationSymphonyOutputTokens] = strconv.FormatInt(result.OutputTokens, 10)
	}
	if result.TotalTokens > 0 {
		updated.Annotations[AnnotationSymphonyTotalTokens] = strconv.FormatInt(result.TotalTokens, 10)
	}
	if result.SecondsRunning > 0 {
		updated.Annotations[AnnotationSymphonyRuntimeSeconds] = strconv.FormatFloat(result.SecondsRunning, 'f', 3, 64)
	}
	startedAt := now
	if result.SecondsRunning > 0 {
		startedAt = now.Add(-time.Duration(result.SecondsRunning * float64(time.Second)))
	}
	started := metav1.NewTime(startedAt)
	completed := metav1.NewTime(now)
	updated.Status.StartTime = &started
	updated.Status.CompletionTime = &completed
	if execErr != nil {
		updated.Status.Phase = operatorv1alpha1.HarnessRunPhaseFailed
		updated.Status.Conditions = []metav1.Condition{{Type: ConditionFailed, Status: metav1.ConditionTrue, Reason: symphonyErrorReason(execErr), Message: execErr.Error(), LastTransitionTime: completed}}
	} else {
		updated.Status.Phase = operatorv1alpha1.HarnessRunPhaseSucceeded
		updated.Status.Conditions = []metav1.Condition{{Type: ConditionSucceeded, Status: metav1.ConditionTrue, Reason: "WorkflowCompleted", Message: firstNonEmpty(result.LastMessage, "workflow execution completed"), LastTransitionTime: completed}}
	}
	metaUpdated := updated.DeepCopy()
	metaUpdated.Status = original.Status
	if err := r.Patch(ctx, metaUpdated, client.MergeFrom(original)); err != nil {
		return err
	}
	var latest operatorv1alpha1.HarnessRun
	if err := r.Get(ctx, client.ObjectKeyFromObject(run), &latest); err != nil {
		return err
	}
	latest.Status = updated.Status
	return r.Status().Update(ctx, &latest)
}

func (r *SymphonyProjectReconciler) loadGitHubToken(ctx context.Context, project *operatorv1alpha1.SymphonyProject) (string, error) {
	secretName := strings.TrimSpace(project.Spec.Source.TokenSecretRef.Name)
	secretKey := strings.TrimSpace(project.Spec.Source.TokenSecretRef.Key)
	if secretKey == "" {
		secretKey = "token"
	}
	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: secretName}, &secret); err != nil {
		return "", fmt.Errorf("load github token secret %s: %w", secretName, err)
	}
	tokenBytes, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("github token secret %s missing key %q", secretName, secretKey)
	}
	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return "", fmt.Errorf("github token secret %s key %q is empty", secretName, secretKey)
	}
	return token, nil
}

func (r *SymphonyProjectReconciler) listProjectRuns(ctx context.Context, project *operatorv1alpha1.SymphonyProject) (map[string]operatorv1alpha1.HarnessRun, error) {
	var runs operatorv1alpha1.HarnessRunList
	if err := r.List(ctx, &runs, client.InNamespace(project.Namespace), client.MatchingLabels{
		LabelSymphonyProjectName: project.Name,
		LabelSymphonyProjectUID:  string(project.UID),
	}); err != nil {
		return nil, err
	}
	byItem := map[string]operatorv1alpha1.HarnessRun{}
	for _, run := range runs.Items {
		itemID := strings.TrimSpace(run.Labels[LabelSymphonyItemID])
		if itemID == "" {
			continue
		}
		current, exists := byItem[itemID]
		if !exists || newerHarnessRun(run, current) {
			byItem[itemID] = run
		}
	}
	return byItem, nil
}

func (r *SymphonyProjectReconciler) releaseInactiveRuns(ctx context.Context, project *operatorv1alpha1.SymphonyProject, snapshot githubsource.Snapshot, runsByItem map[string]operatorv1alpha1.HarnessRun) error {
	activeItems := make(map[string]struct{}, len(snapshot.Candidates))
	for _, candidate := range snapshot.Candidates {
		activeItems[candidate.ItemID] = struct{}{}
	}
	for itemID, run := range runsByItem {
		if _, ok := activeItems[itemID]; ok {
			continue
		}
		if err := r.deleteClaimSession(ctx, project.Namespace, run.Spec.WorkspaceSessionName); err != nil {
			return err
		}
		switch run.Status.Phase {
		case operatorv1alpha1.HarnessRunPhasePending, operatorv1alpha1.HarnessRunPhaseStarting, operatorv1alpha1.HarnessRunPhaseRunning:
			if err := r.Delete(ctx, &run); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
			delete(runsByItem, itemID)
		}
	}
	return nil
}

func (r *SymphonyProjectReconciler) deleteClaimSession(ctx context.Context, namespace, name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	session := &operatorv1alpha1.Session{}
	err := r.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, session)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := r.Delete(ctx, session); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func newerHarnessRun(left, right operatorv1alpha1.HarnessRun) bool {
	if !left.CreationTimestamp.Equal(&right.CreationTimestamp) {
		return left.CreationTimestamp.After(right.CreationTimestamp.Time)
	}
	return left.Name > right.Name
}

func reconcileProjectRuntime(project *operatorv1alpha1.SymphonyProject, snapshot githubsource.Snapshot, runsByItem map[string]operatorv1alpha1.HarnessRun, now time.Time) {
	previousClaims := mapClaimsByItem(project.Status.ActiveClaims)
	previousRetries := mapRetriesByItem(project.Status.RetryQueue)
	limit := int(project.Spec.Runtime.ActiveStatusItemLimit)
	if limit <= 0 {
		limit = operatorv1alpha1.DefaultSymphonyActiveStateLimit
	}
	retryLimit := int(project.Spec.Runtime.RecentErrorLimit)
	if retryLimit <= 0 {
		retryLimit = operatorv1alpha1.DefaultSymphonyRecentErrorLimit
	}
	maxConcurrent := int(project.Spec.Runtime.MaxConcurrentItems)
	if maxConcurrent <= 0 {
		maxConcurrent = operatorv1alpha1.DefaultSymphonyMaxConcurrentItems
	}

	claims := make([]operatorv1alpha1.SymphonyProjectClaimStatus, 0, minInt(limit, maxConcurrent))
	retries := make([]operatorv1alpha1.SymphonyProjectRetryStatus, 0, retryLimit)
	errors := make([]operatorv1alpha1.SymphonyProjectErrorStatus, 0, retryLimit)
	events := make([]operatorv1alpha1.SymphonyProjectEventStatus, 0, retryLimit)
	completed := int32(0)
	failed := int32(0)
	totals := operatorv1alpha1.SymphonyProjectTokenTotalsStatus{}
	consumed := map[string]struct{}{}
	deferredRetry := map[string]operatorv1alpha1.SymphonyProjectRetryStatus{}
	readyRetries := map[string]operatorv1alpha1.SymphonyProjectRetryStatus{}
	readyRetryIDs := make([]string, 0)
	freshCandidates := make([]githubsource.CandidateItem, 0)

	for _, candidate := range snapshot.Candidates {
		if retry, ok := previousRetries[candidate.ItemID]; ok {
			if _, claimed := previousClaims[candidate.ItemID]; !claimed {
				if retry.ReadyAt != nil && !retry.ReadyAt.Time.After(now) {
					if len(claims) < maxConcurrent {
						claims = append(claims, buildClaimFromRetry(candidate, retry, now))
						consumed[candidate.ItemID] = struct{}{}
					} else {
						deferredRetry[candidate.ItemID] = retry
					}
					continue
				}
				deferredRetry[candidate.ItemID] = retry
				continue
			}
		}
		run, hasRun := runsByItem[candidate.ItemID]
		if hasRun {
			totals.InputTokens += runAnnotationInt64(run, AnnotationSymphonyInputTokens)
			totals.OutputTokens += runAnnotationInt64(run, AnnotationSymphonyOutputTokens)
			totals.TotalTokens += runAnnotationInt64(run, AnnotationSymphonyTotalTokens)
			totals.SecondsRunning += runAnnotationFloat64(run, AnnotationSymphonyRuntimeSeconds)
			if event, ok := buildEventStatus(candidate, run); ok {
				events = append(events, event)
			}
			switch run.Status.Phase {
			case operatorv1alpha1.HarnessRunPhaseSucceeded:
				completed++
				if retry, ok := buildContinuationRetryStatus(candidate, previousClaims[candidate.ItemID], previousRetries[candidate.ItemID], now); ok {
					if retry.ReadyAt != nil && !retry.ReadyAt.Time.After(now) {
						if len(claims) < maxConcurrent {
							claims = append(claims, buildClaimFromRetry(candidate, retry, now))
							consumed[candidate.ItemID] = struct{}{}
						} else {
							deferredRetry[candidate.ItemID] = retry
						}
					} else {
						deferredRetry[candidate.ItemID] = retry
					}
				}
				continue
			case operatorv1alpha1.HarnessRunPhasePending, operatorv1alpha1.HarnessRunPhaseStarting, operatorv1alpha1.HarnessRunPhaseRunning:
				claims = append(claims, buildClaimStatus(candidate, run, previousClaims[candidate.ItemID], now))
				consumed[candidate.ItemID] = struct{}{}
				continue
			case operatorv1alpha1.HarnessRunPhaseFailed:
				failed++
				errors = append(errors, buildErrorStatus(candidate, run, previousClaimOrRetryAttempt(previousClaims[candidate.ItemID], previousRetries[candidate.ItemID]), now))
				if retry, ok := buildRetryStatus(candidate, run, previousClaims[candidate.ItemID], previousRetries[candidate.ItemID], project.Spec.Runtime, now); ok {
					if retry.ReadyAt != nil && !retry.ReadyAt.Time.After(now) {
						if len(claims) < maxConcurrent {
							claims = append(claims, buildClaimFromRetry(candidate, retry, now))
							consumed[candidate.ItemID] = struct{}{}
						} else {
							deferredRetry[candidate.ItemID] = retry
						}
					} else {
						deferredRetry[candidate.ItemID] = retry
					}
				}
				continue
			}
		}
		if existing, ok := previousClaims[candidate.ItemID]; ok {
			claims = append(claims, buildClaimWithoutRun(candidate, existing, now))
			consumed[candidate.ItemID] = struct{}{}
			continue
		}
		if retry, ok := previousRetries[candidate.ItemID]; ok {
			if retry.ReadyAt != nil && !retry.ReadyAt.Time.After(now) {
				readyRetries[candidate.ItemID] = retry
				readyRetryIDs = append(readyRetryIDs, candidate.ItemID)
			} else {
				deferredRetry[candidate.ItemID] = retry
			}
			continue
		}
		freshCandidates = append(freshCandidates, candidate)
	}

	sort.Strings(readyRetryIDs)
	for _, itemID := range readyRetryIDs {
		retry, ok := readyRetries[itemID]
		if !ok {
			retry = previousRetries[itemID]
		}
		if len(claims) >= maxConcurrent {
			deferredRetry[itemID] = retry
			continue
		}
		candidate, ok := findCandidate(snapshot.Candidates, itemID)
		if !ok {
			continue
		}
		claims = append(claims, buildClaimFromRetry(candidate, retry, now))
		consumed[itemID] = struct{}{}
	}

	for _, candidate := range freshCandidates {
		if len(claims) >= maxConcurrent {
			break
		}
		if _, ok := consumed[candidate.ItemID]; ok {
			continue
		}
		claims = append(claims, buildFreshClaim(candidate, now))
		consumed[candidate.ItemID] = struct{}{}
	}

	retryIDs := make([]string, 0, len(deferredRetry))
	for itemID := range deferredRetry {
		retryIDs = append(retryIDs, itemID)
	}
	sort.Strings(retryIDs)
	for _, itemID := range retryIDs {
		retries = append(retries, deferredRetry[itemID])
	}

	claims = truncateClaims(claims, limit)
	retries = truncateRetries(retries, retryLimit)
	errors = truncateErrors(errors, retryLimit)
	events = truncateEvents(events, retryLimit)
	project.Status.ActiveClaims = claims
	project.Status.RetryQueue = retries
	project.Status.RecentErrors = errors
	project.Status.RecentEvents = events
	project.Status.TokenTotals = totals
	project.Status.RecentSkips = snapshotSkipsToStatus(snapshot.Skipped, int(project.Spec.Runtime.RecentSkipLimit))
	project.Status.UnsupportedRepos = append([]string(nil), snapshot.UnsupportedRepositories...)
	project.Status.EligibleItems = int32(len(snapshot.Candidates))
	project.Status.RunningItems = int32(len(claims))
	project.Status.RetryingItems = int32(len(retries))
	project.Status.CompletedItems = completed
	project.Status.FailedItems = failed
	project.Status.SkippedItems = int32(len(snapshot.Skipped))
}

func buildClaimStatus(candidate githubsource.CandidateItem, run operatorv1alpha1.HarnessRun, previous operatorv1alpha1.SymphonyProjectClaimStatus, now time.Time) operatorv1alpha1.SymphonyProjectClaimStatus {
	claim := buildClaimWithoutRun(candidate, previous, now)
	claim.Phase = string(run.Status.Phase)
	claim.RunRef = operatorv1alpha1.SymphonyProjectRunRefStatus{
		SessionName:    strings.TrimSpace(run.Spec.WorkspaceSessionName),
		HarnessRunName: run.Name,
	}
	nowMeta := metav1.NewTime(now)
	claim.LastUpdatedTime = &nowMeta
	return claim
}

func buildClaimWithoutRun(candidate githubsource.CandidateItem, previous operatorv1alpha1.SymphonyProjectClaimStatus, now time.Time) operatorv1alpha1.SymphonyProjectClaimStatus {
	attempt := previous.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	claimedAt := previous.ClaimedAt
	if claimedAt == nil {
		t := metav1.NewTime(now)
		claimedAt = &t
	}
	lastUpdated := metav1.NewTime(now)
	return operatorv1alpha1.SymphonyProjectClaimStatus{
		ItemID:          candidate.ItemID,
		Issue:           issueStatus(candidate.Issue),
		Attempt:         attempt,
		Phase:           firstNonEmpty(previous.Phase, "Claimed"),
		ClaimedAt:       claimedAt,
		LastUpdatedTime: &lastUpdated,
		RunRef:          previous.RunRef,
	}
}

func buildRetryStatus(candidate githubsource.CandidateItem, run operatorv1alpha1.HarnessRun, previousClaim operatorv1alpha1.SymphonyProjectClaimStatus, previousRetry operatorv1alpha1.SymphonyProjectRetryStatus, runtime operatorv1alpha1.SymphonyProjectRuntimeSpec, now time.Time) (operatorv1alpha1.SymphonyProjectRetryStatus, bool) {
	attempt := previousClaimOrRetryAttempt(previousClaim, previousRetry)
	readyAt := previousRetry.ReadyAt
	if readyAt == nil || previousRetry.Attempt != attempt {
		delay := retryDelay(runtime, attempt)
		t := metav1.NewTime(now.Add(delay))
		readyAt = &t
	}
	lastError := metav1.NewTime(now)
	return operatorv1alpha1.SymphonyProjectRetryStatus{
		ItemID:        candidate.ItemID,
		Issue:         issueStatus(candidate.Issue),
		Attempt:       attempt,
		Reason:        firstNonEmpty(conditionReason(run.Status.Conditions, ConditionFailed), "HarnessRunFailed"),
		ReadyAt:       readyAt,
		LastErrorTime: &lastError,
	}, true
}

func buildErrorStatus(candidate githubsource.CandidateItem, run operatorv1alpha1.HarnessRun, attempt int32, now time.Time) operatorv1alpha1.SymphonyProjectErrorStatus {
	lastError := metav1.NewTime(now)
	return operatorv1alpha1.SymphonyProjectErrorStatus{
		ItemID:         candidate.ItemID,
		Issue:          issueStatus(candidate.Issue),
		Attempt:        attempt,
		Reason:         firstNonEmpty(conditionReason(run.Status.Conditions, ConditionFailed), "HarnessRunFailed"),
		LastErrorTime:  &lastError,
		HarnessRunName: run.Name,
	}
}

func buildEventStatus(candidate githubsource.CandidateItem, run operatorv1alpha1.HarnessRun) (operatorv1alpha1.SymphonyProjectEventStatus, bool) {
	eventName := strings.TrimSpace(run.Annotations[AnnotationSymphonyLastEvent])
	if eventName == "" {
		return operatorv1alpha1.SymphonyProjectEventStatus{}, false
	}
	observedAt := run.Status.CompletionTime
	if observedAt == nil {
		observedAt = run.Status.StartTime
	}
	return operatorv1alpha1.SymphonyProjectEventStatus{
		ItemID:         candidate.ItemID,
		Issue:          issueStatus(candidate.Issue),
		SessionID:      strings.TrimSpace(run.Annotations[AnnotationSymphonySessionID]),
		ThreadID:       strings.TrimSpace(run.Annotations[AnnotationSymphonyThreadID]),
		TurnID:         strings.TrimSpace(run.Annotations[AnnotationSymphonyTurnID]),
		Event:          eventName,
		Message:        strings.TrimSpace(run.Annotations[AnnotationSymphonyLastMessage]),
		ObservedTime:   observedAt,
		HarnessRunName: run.Name,
	}, true
}

func buildContinuationRetryStatus(candidate githubsource.CandidateItem, previousClaim operatorv1alpha1.SymphonyProjectClaimStatus, previousRetry operatorv1alpha1.SymphonyProjectRetryStatus, now time.Time) (operatorv1alpha1.SymphonyProjectRetryStatus, bool) {
	attempt := previousClaim.Attempt
	if attempt <= 0 {
		attempt = 1
	}
	readyAt := previousRetry.ReadyAt
	if readyAt == nil || !strings.EqualFold(strings.TrimSpace(previousRetry.Reason), "Continuation") || previousRetry.Attempt != attempt {
		t := metav1.NewTime(now.Add(time.Second))
		readyAt = &t
	}
	return operatorv1alpha1.SymphonyProjectRetryStatus{
		ItemID:  candidate.ItemID,
		Issue:   issueStatus(candidate.Issue),
		Attempt: attempt,
		Reason:  "Continuation",
		ReadyAt: readyAt,
	}, true
}

func previousClaimOrRetryAttempt(previousClaim operatorv1alpha1.SymphonyProjectClaimStatus, previousRetry operatorv1alpha1.SymphonyProjectRetryStatus) int32 {
	attempt := previousClaim.Attempt
	if attempt <= 0 {
		attempt = previousRetry.Attempt
	}
	if attempt <= 0 {
		attempt = 1
	}
	return attempt
}

func buildClaimFromRetry(candidate githubsource.CandidateItem, retry operatorv1alpha1.SymphonyProjectRetryStatus, now time.Time) operatorv1alpha1.SymphonyProjectClaimStatus {
	claimedAt := metav1.NewTime(now)
	lastUpdated := metav1.NewTime(now)
	return operatorv1alpha1.SymphonyProjectClaimStatus{
		ItemID:          candidate.ItemID,
		Issue:           issueStatus(candidate.Issue),
		Attempt:         retry.Attempt + 1,
		Phase:           "Claimed",
		ClaimedAt:       &claimedAt,
		LastUpdatedTime: &lastUpdated,
	}
}

func buildFreshClaim(candidate githubsource.CandidateItem, now time.Time) operatorv1alpha1.SymphonyProjectClaimStatus {
	claimedAt := metav1.NewTime(now)
	return operatorv1alpha1.SymphonyProjectClaimStatus{
		ItemID:          candidate.ItemID,
		Issue:           issueStatus(candidate.Issue),
		Attempt:         1,
		Phase:           "Claimed",
		ClaimedAt:       &claimedAt,
		LastUpdatedTime: &claimedAt,
	}
}

func retryDelay(runtime operatorv1alpha1.SymphonyProjectRuntimeSpec, attempt int32) time.Duration {
	base := time.Duration(runtime.RetryBaseDelaySeconds) * time.Second
	maxDelay := time.Duration(runtime.RetryMaxDelaySeconds) * time.Second
	if base <= 0 {
		base = time.Duration(operatorv1alpha1.DefaultSymphonyRetryBaseDelay) * time.Second
	}
	if maxDelay < base {
		maxDelay = time.Duration(operatorv1alpha1.DefaultSymphonyRetryMaxDelay) * time.Second
	}
	delay := base
	for step := int32(1); step < attempt; step++ {
		delay *= 2
		if delay >= maxDelay {
			return maxDelay
		}
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func snapshotSkipsToStatus(skipped []githubsource.SkippedItem, limit int) []operatorv1alpha1.SymphonyProjectSkipStatus {
	if limit <= 0 {
		limit = operatorv1alpha1.DefaultSymphonyRecentSkipLimit
	}
	items := make([]operatorv1alpha1.SymphonyProjectSkipStatus, 0, minInt(limit, len(skipped)))
	for _, item := range skipped {
		entry := operatorv1alpha1.SymphonyProjectSkipStatus{
			ItemID:       item.ItemID,
			Repository:   item.Repository,
			Reason:       item.Reason,
			Message:      item.Message,
			ObservedTime: &metav1.Time{Time: item.ObservedAt},
		}
		if item.Issue != nil {
			entry.Issue = issueStatus(*item.Issue)
		}
		items = append(items, entry)
		if len(items) == limit {
			break
		}
	}
	return items
}

func issueStatus(issue githubsource.Issue) operatorv1alpha1.SymphonyProjectIssueRefStatus {
	return operatorv1alpha1.SymphonyProjectIssueRefStatus{
		Repository: issue.Repository,
		Number:     issue.Number,
		NodeID:     issue.NodeID,
		URL:        issue.URL,
		Title:      issue.Title,
	}
}

func repositoryKeyFromStatus(issue operatorv1alpha1.SymphonyProjectIssueRefStatus) string {
	ownerRepo := strings.TrimSpace(strings.ToLower(issue.Repository))
	if ownerRepo == "" {
		return ""
	}
	return ownerRepo
}

func repositoryRepoURL(repo operatorv1alpha1.SymphonyProjectRepositorySpec) string {
	if value := strings.TrimSpace(repo.RepoURL); value != "" {
		return value
	}
	return fmt.Sprintf("https://github.com/%s/%s", strings.TrimSpace(repo.Owner), strings.TrimSpace(repo.Name))
}

func claimRepoRevision(runtime operatorv1alpha1.SymphonyProjectRuntimeSpec, repo operatorv1alpha1.SymphonyProjectRepositorySpec) string {
	if value := strings.TrimSpace(repo.Branch); value != "" {
		return value
	}
	return strings.TrimSpace(runtime.DefaultRepoRevision)
}

func claimEgressMode(runtime operatorv1alpha1.SymphonyProjectRuntimeSpec, repo operatorv1alpha1.SymphonyProjectRepositorySpec) string {
	if value := strings.TrimSpace(repo.EgressMode); value != "" {
		return value
	}
	return strings.TrimSpace(runtime.DefaultEgressMode)
}

func symphonySessionName(project *operatorv1alpha1.SymphonyProject, claim operatorv1alpha1.SymphonyProjectClaimStatus) string {
	base := sanitizeDNSLabel(strings.Join([]string{"sym", project.Name, claim.Issue.Repository, strconv.FormatInt(claim.Issue.Number, 10)}, "-"))
	if len(base) > 54 {
		base = strings.Trim(sanitizeDNSLabel(base[:54]), "-")
	}
	if base == "" {
		base = "sym-session"
	}
	return base
}

func symphonyRunName(project *operatorv1alpha1.SymphonyProject, claim operatorv1alpha1.SymphonyProjectClaimStatus) string {
	base := sanitizeDNSLabel(strings.Join([]string{"sym", project.Name, claim.Issue.Repository, strconv.FormatInt(claim.Issue.Number, 10), "a", strconv.Itoa(int(claim.Attempt))}, "-"))
	if len(base) > 63 {
		base = strings.Trim(sanitizeDNSLabel(base[:63]), "-")
	}
	if base == "" {
		base = "sym-run"
	}
	return base
}

func symphonySessionDisplayName(claim operatorv1alpha1.SymphonyProjectClaimStatus) string {
	return sanitizeDNSLabel(strings.Join([]string{claim.Issue.Repository, strconv.FormatInt(claim.Issue.Number, 10)}, "-"))
}

func symphonyObjectLabels(project *operatorv1alpha1.SymphonyProject, claim operatorv1alpha1.SymphonyProjectClaimStatus) map[string]string {
	labels := map[string]string{
		LabelSymphonyProjectName: project.Name,
		LabelSymphonyProjectUID:  string(project.UID),
		LabelSymphonyItemID:      claim.ItemID,
		LabelGitHubRepository:    sanitizeLabelValue(claim.Issue.Repository),
	}
	if claim.Issue.Number > 0 {
		labels[LabelGitHubIssueNumber] = strconv.FormatInt(claim.Issue.Number, 10)
	}
	return labels
}

func sanitizeLabelValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	b := strings.Builder{}
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	cleaned := strings.Trim(b.String(), "-._")
	for strings.Contains(cleaned, "--") {
		cleaned = strings.ReplaceAll(cleaned, "--", "-")
	}
	if cleaned == "" {
		return "unknown"
	}
	if len(cleaned) > 63 {
		cleaned = strings.Trim(cleaned[:63], "-._")
		if cleaned == "" {
			return "unknown"
		}
	}
	return cleaned
}

func symphonyObjectAnnotations(claim operatorv1alpha1.SymphonyProjectClaimStatus) map[string]string {
	annotations := map[string]string{}
	if claim.Issue.URL != "" {
		annotations[AnnotationSymphonyIssueURL] = claim.Issue.URL
	}
	if claim.Issue.NodeID != "" {
		annotations[AnnotationSymphonyIssueNode] = claim.Issue.NodeID
	}
	return annotations
}

func mapClaimsByItem(items []operatorv1alpha1.SymphonyProjectClaimStatus) map[string]operatorv1alpha1.SymphonyProjectClaimStatus {
	mapped := make(map[string]operatorv1alpha1.SymphonyProjectClaimStatus, len(items))
	for _, item := range items {
		mapped[item.ItemID] = item
	}
	return mapped
}

func mapRetriesByItem(items []operatorv1alpha1.SymphonyProjectRetryStatus) map[string]operatorv1alpha1.SymphonyProjectRetryStatus {
	mapped := make(map[string]operatorv1alpha1.SymphonyProjectRetryStatus, len(items))
	for _, item := range items {
		mapped[item.ItemID] = item
	}
	return mapped
}

func findCandidate(items []githubsource.CandidateItem, itemID string) (githubsource.CandidateItem, bool) {
	for _, item := range items {
		if item.ItemID == itemID {
			return item, true
		}
	}
	return githubsource.CandidateItem{}, false
}

func truncateClaims(items []operatorv1alpha1.SymphonyProjectClaimStatus, limit int) []operatorv1alpha1.SymphonyProjectClaimStatus {
	if len(items) <= limit {
		return items
	}
	return append([]operatorv1alpha1.SymphonyProjectClaimStatus(nil), items[:limit]...)
}

func truncateRetries(items []operatorv1alpha1.SymphonyProjectRetryStatus, limit int) []operatorv1alpha1.SymphonyProjectRetryStatus {
	if len(items) <= limit {
		return items
	}
	return append([]operatorv1alpha1.SymphonyProjectRetryStatus(nil), items[:limit]...)
}

func truncateEvents(items []operatorv1alpha1.SymphonyProjectEventStatus, limit int) []operatorv1alpha1.SymphonyProjectEventStatus {
	if len(items) <= limit {
		return items
	}
	return append([]operatorv1alpha1.SymphonyProjectEventStatus(nil), items[:limit]...)
}

func truncateErrors(items []operatorv1alpha1.SymphonyProjectErrorStatus, limit int) []operatorv1alpha1.SymphonyProjectErrorStatus {
	if len(items) <= limit {
		return items
	}
	return append([]operatorv1alpha1.SymphonyProjectErrorStatus(nil), items[:limit]...)
}

func conditionReason(conditions []metav1.Condition, conditionType string) string {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return strings.TrimSpace(condition.Reason)
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func issueTemplateData(issue githubsource.Issue) map[string]any {
	return map[string]any{
		"repository":  issue.Repository,
		"number":      issue.Number,
		"nodeId":      issue.NodeID,
		"url":         issue.URL,
		"title":       issue.Title,
		"description": issue.Body,
		"labels":      append([]string(nil), issue.Labels...),
		"projectItem": issue.ProjectItem,
	}
}

func enforceWorkflowSecurity(def workflow.Definition, cfg workflow.Config) error {
	if strings.TrimSpace(cfg.Hooks.AfterCreate) != "" || strings.TrimSpace(cfg.Hooks.BeforeRun) != "" || strings.TrimSpace(cfg.Hooks.AfterRun) != "" || strings.TrimSpace(cfg.Hooks.BeforeRemove) != "" {
		return &workflow.Error{Code: workflow.ErrCodeWorkflowValidationError, Path: def.Path, Err: fmt.Errorf("workflow hooks are disabled in kocao symphony workers")}
	}
	return nil
}

func secureCodexConfig(cfg workflow.CodexConfig) workflow.CodexConfig {
	if strings.TrimSpace(cfg.ApprovalPolicy) == "" {
		cfg.ApprovalPolicy = defaultSymphonyApprovalPolicy
	}
	if strings.TrimSpace(cfg.ThreadSandbox) == "" {
		cfg.ThreadSandbox = defaultSymphonyThreadSandbox
	}
	if strings.TrimSpace(cfg.TurnSandboxPolicy) == "" {
		cfg.TurnSandboxPolicy = defaultSymphonyTurnSandboxPolicy
	}
	return cfg
}

func sanitizeTelemetryMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return ""
	}
	if sensitiveTelemetryPattern.MatchString(message) {
		return "[redacted]"
	}
	return message
}

func resultSecurityAnnotations(result symphonyWorkerResult) (string, string, string) {
	return strings.TrimSpace(result.ApprovalPolicy), strings.TrimSpace(result.ThreadSandbox), strings.TrimSpace(result.TurnSandbox)
}

func int32PtrToIntPtr(value int32) *int {
	if value <= 0 {
		return nil
	}
	converted := int(value)
	return &converted
}

func sessionID(threadID, turnID string) string {
	if strings.TrimSpace(threadID) == "" || strings.TrimSpace(turnID) == "" {
		return ""
	}
	return strings.TrimSpace(threadID) + "-" + strings.TrimSpace(turnID)
}

func symphonyErrorReason(err error) string {
	if err == nil {
		return ""
	}
	var workflowErr *workflow.Error
	if errors.As(err, &workflowErr) {
		switch workflowErr.Code {
		case workflow.ErrCodeMissingWorkflowFile:
			return "WorkflowMissing"
		case workflow.ErrCodeTemplateRenderError:
			return "WorkflowRenderFailed"
		case workflow.ErrCodeWorkflowValidationError:
			return "WorkflowInvalid"
		}
	}
	return "WorkerExecutionFailed"
}

func runAnnotationInt64(run operatorv1alpha1.HarnessRun, key string) int64 {
	value := strings.TrimSpace(run.Annotations[key])
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func runAnnotationFloat64(run operatorv1alpha1.HarnessRun, key string) float64 {
	value := strings.TrimSpace(run.Annotations[key])
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (r *SymphonyProjectReconciler) setConfigError(project *operatorv1alpha1.SymphonyProject, now metav1.Time, err error) {
	project.Status.Phase = operatorv1alpha1.SymphonyProjectPhaseError
	project.Status.LastError = err.Error()
	project.Status.ObservedGeneration = project.Generation
	setCondition(&project.Status.Conditions, metav1.Condition{Type: ConditionConfig, Status: metav1.ConditionFalse, Reason: "Invalid", Message: err.Error(), LastTransitionTime: now})
	setCondition(&project.Status.Conditions, metav1.Condition{Type: ConditionSource, Status: metav1.ConditionFalse, Reason: "Blocked", Message: "symphony source sync is blocked by invalid configuration", LastTransitionTime: now})
	setCondition(&project.Status.Conditions, metav1.Condition{Type: ConditionLifecycle, Status: metav1.ConditionFalse, Reason: "Blocked", Message: "symphony orchestration is blocked by invalid configuration", LastTransitionTime: now})
}

func (r *SymphonyProjectReconciler) setSourceError(project *operatorv1alpha1.SymphonyProject, now metav1.Time, err error, pollInterval time.Duration) {
	project.Status.Phase = operatorv1alpha1.SymphonyProjectPhaseError
	project.Status.LastError = err.Error()
	project.Status.LastSyncTime = &now
	project.Status.NextSyncTime = &metav1.Time{Time: now.Time.Add(pollInterval)}
	project.Status.ObservedGeneration = project.Generation
	setCondition(&project.Status.Conditions, metav1.Condition{Type: ConditionSource, Status: metav1.ConditionFalse, Reason: "SyncFailed", Message: err.Error(), LastTransitionTime: now})
	setCondition(&project.Status.Conditions, metav1.Condition{Type: ConditionLifecycle, Status: metav1.ConditionFalse, Reason: "Degraded", Message: "symphony orchestration is waiting for the next successful sync", LastTransitionTime: now})
}

func (r *SymphonyProjectReconciler) setLifecycleError(project *operatorv1alpha1.SymphonyProject, now metav1.Time, err error, pollInterval time.Duration) {
	project.Status.Phase = operatorv1alpha1.SymphonyProjectPhaseError
	project.Status.LastError = err.Error()
	project.Status.LastSyncTime = &now
	project.Status.NextSyncTime = &metav1.Time{Time: now.Time.Add(pollInterval)}
	setCondition(&project.Status.Conditions, metav1.Condition{Type: ConditionLifecycle, Status: metav1.ConditionFalse, Reason: "ExecutionFailed", Message: err.Error(), LastTransitionTime: now})
}

func (r *SymphonyProjectReconciler) commit(ctx context.Context, original, updated *operatorv1alpha1.SymphonyProject, changedMeta, changedStatus bool, result ...ctrl.Result) (ctrl.Result, error) {
	res := ctrl.Result{}
	if len(result) != 0 {
		res = result[0]
	}
	if changedMeta {
		metaUpdated := updated.DeepCopy()
		metaUpdated.Status = original.Status
		if err := r.Patch(ctx, metaUpdated, client.MergeFrom(original)); err != nil {
			return ctrl.Result{}, err
		}
	}
	if changedStatus {
		var latest operatorv1alpha1.SymphonyProject
		if err := r.Get(ctx, client.ObjectKeyFromObject(original), &latest); err != nil {
			return ctrl.Result{}, err
		}
		latest.Status = updated.Status
		if err := r.Status().Update(ctx, &latest); err != nil {
			return ctrl.Result{}, err
		}
	}
	r.emitAuditEvents(ctx, original, updated)
	return res, nil
}

func (r *SymphonyProjectReconciler) emitAuditEvents(ctx context.Context, original, updated *operatorv1alpha1.SymphonyProject) {
	if r.Audit == nil {
		return
	}
	if original.Spec.Paused != updated.Spec.Paused {
		action := "symphony.resume"
		if updated.Spec.Paused {
			action = "symphony.pause"
		}
		auditlog.AppendSymphony(ctx, r.Audit, "operator", action, updated.Name, "allowed", map[string]any{"paused": updated.Spec.Paused})
	}
	if original.Status.LastSuccessfulSync == nil || (updated.Status.LastSuccessfulSync != nil && !updated.Status.LastSuccessfulSync.Equal(original.Status.LastSuccessfulSync)) {
		if updated.Status.LastSuccessfulSync != nil {
			auditlog.AppendSymphony(ctx, r.Audit, "operator", "symphony.sync", updated.Name, "allowed", map[string]any{"runningItems": updated.Status.RunningItems, "retryingItems": updated.Status.RetryingItems})
		}
	}
	beforeClaims := mapClaimsByItem(original.Status.ActiveClaims)
	afterClaims := mapClaimsByItem(updated.Status.ActiveClaims)
	for itemID, claim := range afterClaims {
		if _, ok := beforeClaims[itemID]; ok {
			continue
		}
		auditlog.AppendSymphony(ctx, r.Audit, "operator", "symphony.claim", updated.Name, "allowed", map[string]any{"itemID": claim.ItemID, "repository": claim.Issue.Repository, "issueNumber": claim.Issue.Number, "attempt": claim.Attempt})
	}
	beforeRetries := mapRetriesByItem(original.Status.RetryQueue)
	afterRetries := mapRetriesByItem(updated.Status.RetryQueue)
	for itemID, retry := range afterRetries {
		if previous, ok := beforeRetries[itemID]; ok && previous.Attempt == retry.Attempt && previous.Reason == retry.Reason {
			continue
		}
		auditlog.AppendSymphony(ctx, r.Audit, "operator", "symphony.retry", updated.Name, "allowed", map[string]any{"itemID": retry.ItemID, "repository": retry.Issue.Repository, "issueNumber": retry.Issue.Number, "attempt": retry.Attempt, "reason": retry.Reason})
	}
	for itemID, claim := range beforeClaims {
		if _, ok := afterClaims[itemID]; ok {
			continue
		}
		if _, retrying := afterRetries[itemID]; retrying {
			continue
		}
		auditlog.AppendSymphony(ctx, r.Audit, "operator", "symphony.release", updated.Name, "allowed", map[string]any{"itemID": claim.ItemID, "repository": claim.Issue.Repository, "issueNumber": claim.Issue.Number, "phase": claim.Phase})
	}
}

func (r *SymphonyProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}
	if r.SourceFactory == nil {
		r.SourceFactory = defaultSymphonySourceFactory{}
	}
	if r.WorkerExecutor == nil {
		r.WorkerExecutor = defaultSymphonyWorkerExecutor{}
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.SymphonyProject{}).
		Owns(&operatorv1alpha1.HarnessRun{}).
		Complete(r)
}
