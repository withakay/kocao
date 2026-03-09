package controllers

import (
	"context"
	"errors"
	"testing"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	"github.com/withakay/kocao/internal/symphony/githubsource"
	corev1 "k8s.io/api/core/v1"
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
	r := &SymphonyProjectReconciler{Client: cl, Scheme: scheme, Clock: clocktesting.NewFakeClock(time.Unix(10, 0).UTC()), SourceFactory: stubSymphonySourceFactory{loader: loader}}

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
	if got.Status.EligibleItems != 2 || got.Status.RunningItems != 1 || got.Status.SkippedItems != 1 {
		t.Fatalf("counters = %#v", got.Status)
	}
	if got.Status.UnsupportedRepos[0] != "someone/else" {
		t.Fatalf("unsupported repos = %#v", got.Status.UnsupportedRepos)
	}
	if conditionStatus(got.Status.Conditions, ConditionSource) != metav1.ConditionTrue {
		t.Fatalf("source condition = %#v", got.Status.Conditions)
	}
	if len(got.Finalizers) != 1 || got.Finalizers[0] != FinalizerName {
		t.Fatalf("finalizers = %#v", got.Finalizers)
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
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "github-token", Namespace: "default"}, Data: map[string][]byte{"token": []byte("ghp_test")}}
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
	cl := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&operatorv1alpha1.SymphonyProject{}, &operatorv1alpha1.HarnessRun{}).WithObjects(project, secret, run).Build()
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
	if got.Status.RetryQueue[0].Reason != "PodFailed" {
		t.Fatalf("retry reason = %q", got.Status.RetryQueue[0].Reason)
	}
	if got.Status.RetryQueue[0].ReadyAt == nil || got.Status.RetryQueue[0].ReadyAt.Time.Sub(time.Unix(30, 0).UTC()) != time.Minute {
		t.Fatalf("retry readyAt = %#v", got.Status.RetryQueue[0].ReadyAt)
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
				Image:              "ghcr.io/withakay/kocao-harness:latest",
				MaxConcurrentItems: 1,
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
