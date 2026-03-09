package controllers

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	"github.com/withakay/kocao/internal/symphony/githubsource"
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

type SymphonyProjectReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Clock         clock.Clock
	SourceFactory symphonySourceFactory
}

func (r *SymphonyProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var project operatorv1alpha1.SymphonyProject
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}
	if r.SourceFactory == nil {
		r.SourceFactory = defaultSymphonySourceFactory{}
	}

	now := metav1.NewTime(r.Clock.Now().UTC())
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

	loader, err := r.SourceFactory.New(token)
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

	reconcileProjectRuntime(updated, snapshot, runsByItem, now.Time)
	if err := r.materializeActiveClaims(ctx, updated, runsByItem, now.Time); err != nil {
		return ctrl.Result{}, err
	}
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
		if existingRun, ok := runsByItem[claim.ItemID]; ok {
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

func (r *SymphonyProjectReconciler) ensureClaimSession(ctx context.Context, project *operatorv1alpha1.SymphonyProject, repo operatorv1alpha1.SymphonyProjectRepositorySpec, claim operatorv1alpha1.SymphonyProjectClaimStatus) (*operatorv1alpha1.Session, error) {
	name := symphonySessionName(project, claim)
	desired := &operatorv1alpha1.Session{}
	err := r.Get(ctx, client.ObjectKey{Namespace: project.Namespace, Name: name}, desired)
	if err == nil {
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
				AnnotationAttachEnabled:     "false",
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
	completed := int32(0)
	consumed := map[string]struct{}{}
	deferredRetry := map[string]operatorv1alpha1.SymphonyProjectRetryStatus{}
	readyRetryIDs := make([]string, 0)
	freshCandidates := make([]githubsource.CandidateItem, 0)

	for _, candidate := range snapshot.Candidates {
		run, hasRun := runsByItem[candidate.ItemID]
		if hasRun {
			switch run.Status.Phase {
			case operatorv1alpha1.HarnessRunPhaseSucceeded:
				completed++
				consumed[candidate.ItemID] = struct{}{}
				continue
			case operatorv1alpha1.HarnessRunPhasePending, operatorv1alpha1.HarnessRunPhaseStarting, operatorv1alpha1.HarnessRunPhaseRunning:
				claims = append(claims, buildClaimStatus(candidate, run, previousClaims[candidate.ItemID], now))
				consumed[candidate.ItemID] = struct{}{}
				continue
			case operatorv1alpha1.HarnessRunPhaseFailed:
				if retry, ok := buildRetryStatus(candidate, run, previousClaims[candidate.ItemID], previousRetries[candidate.ItemID], project.Spec.Runtime, now); ok {
					if retry.ReadyAt != nil && !retry.ReadyAt.Time.After(now) {
						readyRetryIDs = append(readyRetryIDs, candidate.ItemID)
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
		if len(claims) >= maxConcurrent {
			retry := previousRetries[itemID]
			deferredRetry[itemID] = retry
			continue
		}
		candidate, ok := findCandidate(snapshot.Candidates, itemID)
		if !ok {
			continue
		}
		claims = append(claims, buildClaimFromRetry(candidate, previousRetries[itemID], now))
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
	project.Status.ActiveClaims = claims
	project.Status.RetryQueue = retries
	project.Status.RecentSkips = snapshotSkipsToStatus(snapshot.Skipped, int(project.Spec.Runtime.RecentSkipLimit))
	project.Status.UnsupportedRepos = append([]string(nil), snapshot.UnsupportedRepositories...)
	project.Status.EligibleItems = int32(len(snapshot.Candidates))
	project.Status.RunningItems = int32(len(claims))
	project.Status.RetryingItems = int32(len(retries))
	project.Status.CompletedItems = completed
	project.Status.FailedItems = 0
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
	attempt := previousClaim.Attempt
	if attempt <= 0 {
		attempt = previousRetry.Attempt
	}
	if attempt <= 0 {
		attempt = 1
	}
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
		LabelGitHubRepository:    claim.Issue.Repository,
	}
	if claim.Issue.Number > 0 {
		labels[LabelGitHubIssueNumber] = strconv.FormatInt(claim.Issue.Number, 10)
	}
	return labels
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
	return res, nil
}

func (r *SymphonyProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.SymphonyProject{}).
		Owns(&operatorv1alpha1.HarnessRun{}).
		Complete(r)
}
