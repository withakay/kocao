package controllers

import (
	"context"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SessionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SessionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var sess operatorv1alpha1.Session
	if err := r.Get(ctx, req.NamespacedName, &sess); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	updated := sess.DeepCopy()
	changedMeta := false
	changedStatus := false

	if updated.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(updated, FinalizerName) {
			controllerutil.AddFinalizer(updated, FinalizerName)
			changedMeta = true
		}
		if err := ensureSessionWorkspacePVC(ctx, r.Client, r.Scheme, updated); err != nil {
			return ctrl.Result{}, err
		}
		now := metav1.Now()
		setCondition(&updated.Status.Conditions, metav1.Condition{Type: ConditionReady, Status: metav1.ConditionTrue, Reason: "Ready", Message: "session accepted", LastTransitionTime: now})
		updated.Status.Phase = operatorv1alpha1.SessionPhaseActive
		updated.Status.ObservedGeneration = updated.Generation
		changedStatus = true
	} else {
		updated.Status.Phase = operatorv1alpha1.SessionPhaseTerminating
		updated.Status.ObservedGeneration = updated.Generation
		changedStatus = true

		var runs operatorv1alpha1.HarnessRunList
		if err := r.List(ctx, &runs, client.InNamespace(updated.Namespace), client.MatchingLabels{LabelSessionName: updated.Name}); err != nil {
			return ctrl.Result{}, err
		}
		if len(runs.Items) != 0 {
			for i := range runs.Items {
				_ = r.Delete(ctx, &runs.Items[i])
			}
			var remaining operatorv1alpha1.HarnessRunList
			if err := r.List(ctx, &remaining, client.InNamespace(updated.Namespace), client.MatchingLabels{LabelSessionName: updated.Name}); err != nil {
				return ctrl.Result{}, err
			}
			if len(remaining.Items) != 0 {
				return ctrl.Result{RequeueAfter: 500 * time.Millisecond}, nil
			}
		}
		if controllerutil.ContainsFinalizer(updated, FinalizerName) {
			controllerutil.RemoveFinalizer(updated, FinalizerName)
			changedMeta = true
		}
	}

	if changedMeta {
		metaUpdated := updated.DeepCopy()
		metaUpdated.Status = sess.Status
		if err := r.Patch(ctx, metaUpdated, client.MergeFrom(&sess)); err != nil {
			return ctrl.Result{}, err
		}
	}
	if changedStatus {
		var latest operatorv1alpha1.Session
		if err := r.Get(ctx, req.NamespacedName, &latest); err != nil {
			return ctrl.Result{}, err
		}
		latest.Status = updated.Status
		if err := r.Status().Update(ctx, &latest); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *SessionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Session{}).
		Complete(r)
}
