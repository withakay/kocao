package controllers

import (
	"context"
	"strings"
	"time"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type HarnessRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock  clock.Clock
}

func (r *HarnessRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var run operatorv1alpha1.HarnessRun
	if err := r.Get(ctx, req.NamespacedName, &run); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if r.Clock == nil {
		r.Clock = clock.RealClock{}
	}

	updated := run.DeepCopy()
	changedMeta := false
	changedStatus := false

	// Ensure session association label for list/filter queries.
	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}
	if run.Spec.SessionName != "" && updated.Labels[LabelSessionName] != run.Spec.SessionName {
		updated.Labels[LabelSessionName] = run.Spec.SessionName
		changedMeta = true
	}

	// Ensure finalizer.
	if updated.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(updated, FinalizerName) {
			controllerutil.AddFinalizer(updated, FinalizerName)
			changedMeta = true
		}
	}

	// Session reference (if provided) must exist.
	if updated.Spec.SessionName != "" {
		var sess operatorv1alpha1.Session
		err := r.Get(ctx, client.ObjectKey{Namespace: updated.Namespace, Name: updated.Spec.SessionName}, &sess)
		if err != nil {
			if apierrors.IsNotFound(err) {
				now := metav1.NewTime(r.Clock.Now())
				setCondition(&updated.Status.Conditions, metav1.Condition{
					Type:               ConditionSession,
					Status:             metav1.ConditionFalse,
					Reason:             "SessionNotFound",
					Message:            "referenced session does not exist",
					LastTransitionTime: now,
				})
				updated.Status.ObservedGeneration = updated.Generation
				updated.Status.Phase = operatorv1alpha1.HarnessRunPhasePending
				changedStatus = true
				if changedMeta {
					metaUpdated := updated.DeepCopy()
					metaUpdated.Status = run.Status
					if err := r.Patch(ctx, metaUpdated, client.MergeFrom(&run)); err != nil {
						return ctrl.Result{}, err
					}
				}
				if changedStatus {
					target := client.Object(updated)
					if changedMeta {
						var latest operatorv1alpha1.HarnessRun
						if err := r.Get(ctx, req.NamespacedName, &latest); err != nil {
							return ctrl.Result{}, err
						}
						latest.Status = updated.Status
						target = &latest
					}
					if err := r.Status().Update(ctx, target); err != nil {
						return ctrl.Result{}, err
					}
				}
				return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		if err := controllerutil.SetControllerReference(&sess, updated, r.Scheme); err == nil {
			changedMeta = true
		}
		now := metav1.NewTime(r.Clock.Now())
		setCondition(&updated.Status.Conditions, metav1.Condition{
			Type:               ConditionSession,
			Status:             metav1.ConditionTrue,
			Reason:             "SessionFound",
			Message:            "referenced session exists",
			LastTransitionTime: now,
		})
		changedStatus = true
	}

	// Delete path.
	if !updated.DeletionTimestamp.IsZero() {
		if updated.Status.PodName != "" {
			var pod corev1.Pod
			err := r.Get(ctx, client.ObjectKey{Namespace: updated.Namespace, Name: updated.Status.PodName}, &pod)
			if err == nil {
				_ = r.Delete(ctx, &pod)
				return ctrl.Result{RequeueAfter: 500 * time.Millisecond}, nil
			}
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		}
		if controllerutil.ContainsFinalizer(updated, FinalizerName) {
			controllerutil.RemoveFinalizer(updated, FinalizerName)
			changedMeta = true
		}
	}

	// Validate required spec.
	if updated.Spec.RepoURL == "" {
		now := metav1.NewTime(r.Clock.Now())
		setCondition(&updated.Status.Conditions, metav1.Condition{
			Type:               ConditionFailed,
			Status:             metav1.ConditionTrue,
			Reason:             "SpecInvalid",
			Message:            invalidSpecError("repoURL").Error(),
			LastTransitionTime: now,
		})
		updated.Status.ObservedGeneration = updated.Generation
		updated.Status.Phase = operatorv1alpha1.HarnessRunPhaseFailed
		changedStatus = true
	}
	if updated.Spec.Image == "" {
		now := metav1.NewTime(r.Clock.Now())
		setCondition(&updated.Status.Conditions, metav1.Condition{
			Type:               ConditionFailed,
			Status:             metav1.ConditionTrue,
			Reason:             "SpecInvalid",
			Message:            invalidSpecError("image").Error(),
			LastTransitionTime: now,
		})
		updated.Status.ObservedGeneration = updated.Generation
		updated.Status.Phase = operatorv1alpha1.HarnessRunPhaseFailed
		changedStatus = true
	}
	if updated.Spec.GitAuth != nil && strings.TrimSpace(updated.Spec.GitAuth.SecretName) == "" {
		now := metav1.NewTime(r.Clock.Now())
		setCondition(&updated.Status.Conditions, metav1.Condition{
			Type:               ConditionFailed,
			Status:             metav1.ConditionTrue,
			Reason:             "SpecInvalid",
			Message:            invalidSpecError("gitAuth.secretName").Error(),
			LastTransitionTime: now,
		})
		updated.Status.ObservedGeneration = updated.Generation
		updated.Status.Phase = operatorv1alpha1.HarnessRunPhaseFailed
		changedStatus = true
	}

	// Create/observe pod if not terminal.
	if updated.Status.Phase != operatorv1alpha1.HarnessRunPhaseFailed {
		if updated.Status.Phase == "" {
			updated.Status.Phase = operatorv1alpha1.HarnessRunPhasePending
			changedStatus = true
		}

		if updated.Status.PodName == "" && updated.DeletionTimestamp.IsZero() {
			pod := buildHarnessPod(updated)
			if err := controllerutil.SetControllerReference(updated, pod, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, pod); err != nil {
				if !apierrors.IsAlreadyExists(err) {
					return ctrl.Result{}, err
				}
			}
			now := metav1.NewTime(r.Clock.Now())
			setCondition(&updated.Status.Conditions, metav1.Condition{
				Type:               ConditionReady,
				Status:             metav1.ConditionTrue,
				Reason:             "PodCreated",
				Message:            "run pod created",
				LastTransitionTime: now,
			})
			updated.Status.PodName = pod.Name
			updated.Status.ObservedGeneration = updated.Generation
			updated.Status.Phase = operatorv1alpha1.HarnessRunPhaseStarting
			changedStatus = true
		}

		if updated.Status.PodName != "" {
			var pod corev1.Pod
			err := r.Get(ctx, client.ObjectKey{Namespace: updated.Namespace, Name: updated.Status.PodName}, &pod)
			if apierrors.IsNotFound(err) {
				updated.Status.PodName = ""
				updated.Status.Phase = operatorv1alpha1.HarnessRunPhasePending
				changedStatus = true
			} else if err != nil {
				return ctrl.Result{}, err
			} else {
				changed, res, deleteNow := updateStatusFromPod(updated, &pod, r.Clock.Now())
				changedStatus = changedStatus || changed
				if changedMeta {
					metaUpdated := updated.DeepCopy()
					metaUpdated.Status = run.Status
					if err := r.Patch(ctx, metaUpdated, client.MergeFrom(&run)); err != nil {
						return ctrl.Result{}, err
					}
					changedMeta = false
				}
				if changedStatus {
					var latest operatorv1alpha1.HarnessRun
					if err := r.Get(ctx, req.NamespacedName, &latest); err != nil {
						return ctrl.Result{}, err
					}
					latest.Status = updated.Status
					if err := r.Status().Update(ctx, &latest); err != nil {
						return ctrl.Result{}, err
					}
					changedStatus = false
				}
				if deleteNow {
					err := r.Delete(ctx, updated)
					if err != nil && !apierrors.IsNotFound(err) {
						return ctrl.Result{}, err
					}
					return ctrl.Result{RequeueAfter: 200 * time.Millisecond}, nil
				}
				if res.RequeueAfter > 0 {
					return res, nil
				}
			}
		}
	}

	if changedMeta {
		metaUpdated := updated.DeepCopy()
		metaUpdated.Status = run.Status
		if err := r.Patch(ctx, metaUpdated, client.MergeFrom(&run)); err != nil {
			return ctrl.Result{}, err
		}
	}
	if changedStatus {
		var latest operatorv1alpha1.HarnessRun
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

func updateStatusFromPod(run *operatorv1alpha1.HarnessRun, pod *corev1.Pod, now time.Time) (bool, ctrl.Result, bool) {
	changed := false
	setPhase := func(phase operatorv1alpha1.HarnessRunPhase) {
		if run.Status.Phase != phase {
			run.Status.Phase = phase
			changed = true
		}
	}
	if run.Status.StartTime == nil && pod.Status.StartTime != nil {
		run.Status.StartTime = &metav1.Time{Time: pod.Status.StartTime.Time}
		changed = true
	}

	nowMeta := metav1.NewTime(now)
	switch pod.Status.Phase {
	case corev1.PodRunning:
		setPhase(operatorv1alpha1.HarnessRunPhaseRunning)
		setCondition(&run.Status.Conditions, metav1.Condition{Type: ConditionRunning, Status: metav1.ConditionTrue, Reason: "PodRunning", Message: "run pod is running", LastTransitionTime: nowMeta})
		clearCondition(&run.Status.Conditions, ConditionSucceeded)
		clearCondition(&run.Status.Conditions, ConditionFailed)
		changed = true
	case corev1.PodSucceeded:
		setPhase(operatorv1alpha1.HarnessRunPhaseSucceeded)
		setCondition(&run.Status.Conditions, metav1.Condition{Type: ConditionSucceeded, Status: metav1.ConditionTrue, Reason: "PodSucceeded", Message: "run pod completed successfully", LastTransitionTime: nowMeta})
		clearCondition(&run.Status.Conditions, ConditionFailed)
		changed = true
		if run.Status.CompletionTime == nil {
			run.Status.CompletionTime = &nowMeta
			changed = true
		}
	case corev1.PodFailed:
		setPhase(operatorv1alpha1.HarnessRunPhaseFailed)
		setCondition(&run.Status.Conditions, metav1.Condition{Type: ConditionFailed, Status: metav1.ConditionTrue, Reason: "PodFailed", Message: "run pod failed", LastTransitionTime: nowMeta})
		clearCondition(&run.Status.Conditions, ConditionSucceeded)
		changed = true
		if run.Status.CompletionTime == nil {
			run.Status.CompletionTime = &nowMeta
			changed = true
		}
	default:
		if run.Status.Phase == operatorv1alpha1.HarnessRunPhaseStarting {
			// keep
		} else {
			setPhase(operatorv1alpha1.HarnessRunPhasePending)
		}
		changed = true
	}

	run.Status.ObservedGeneration = run.Generation

	if run.Status.CompletionTime != nil && run.Spec.TTLSecondsAfterFinished != nil {
		expire := run.Status.CompletionTime.Time.Add(time.Duration(*run.Spec.TTLSecondsAfterFinished) * time.Second)
		if now.After(expire) {
			return changed, ctrl.Result{}, true
		}
		return changed, ctrl.Result{RequeueAfter: expire.Sub(now)}, false
	}

	if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
		return changed, ctrl.Result{RequeueAfter: 2 * time.Second}, false
	}
	return changed, ctrl.Result{}, false
}

func (r *HarnessRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.HarnessRun{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
