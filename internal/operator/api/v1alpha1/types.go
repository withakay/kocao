package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type SessionPhase string

const (
	SessionPhasePending     SessionPhase = "Pending"
	SessionPhaseActive      SessionPhase = "Active"
	SessionPhaseTerminating SessionPhase = "Terminating"
)

type HarnessRunPhase string

const (
	HarnessRunPhasePending   HarnessRunPhase = "Pending"
	HarnessRunPhaseStarting  HarnessRunPhase = "Starting"
	HarnessRunPhaseRunning   HarnessRunPhase = "Running"
	HarnessRunPhaseSucceeded HarnessRunPhase = "Succeeded"
	HarnessRunPhaseFailed    HarnessRunPhase = "Failed"
)

// Session describes a long-lived orchestration container for HarnessRuns.
//
// It is intentionally minimal for early API/UI consumers.
type Session struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SessionSpec   `json:"spec,omitempty"`
	Status SessionStatus `json:"status,omitempty"`
}

type SessionSpec struct {
	// RepoURL is an optional default repository URL for runs within the session.
	RepoURL string `json:"repoURL,omitempty"`
}

type SessionStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              SessionPhase       `json:"phase,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

type SessionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Session `json:"items"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

// GitAuthSpec configures secure Git credential injection for HTTPS clones.
// The referenced Secret MUST exist in the same namespace as the HarnessRun.
//
// This is intentionally minimal for MVP and supports token-based authentication.
type GitAuthSpec struct {
	// SecretName is the name of the Secret holding Git credentials.
	SecretName string `json:"secretName"`
	// TokenKey is the secret data key containing the token (defaults to "token").
	TokenKey string `json:"tokenKey,omitempty"`
	// UsernameKey is an optional secret data key containing the username.
	// If omitted, the harness defaults to "x-access-token".
	UsernameKey string `json:"usernameKey,omitempty"`
}

// HarnessRun describes a single run execution that is reconciled into a Pod.
type HarnessRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarnessRunSpec   `json:"spec,omitempty"`
	Status HarnessRunStatus `json:"status,omitempty"`
}

type HarnessRunSpec struct {
	// SessionName associates this run to a Session in the same namespace.
	SessionName string `json:"sessionName,omitempty"`

	// RepoURL is the repository to run against.
	RepoURL string `json:"repoURL"`

	// RepoRevision is an optional branch/tag/SHA.
	RepoRevision string `json:"repoRevision,omitempty"`

	// Image is the container image used by the run Pod.
	Image string `json:"image"`
	// Command overrides the container entrypoint.
	Command []string `json:"command,omitempty"`
	// Args are passed to the container.
	Args []string `json:"args,omitempty"`
	// WorkingDir sets container working directory.
	WorkingDir string `json:"workingDir,omitempty"`
	// Env configures container environment variables.
	Env []EnvVar `json:"env,omitempty"`

	// GitAuth references a Secret used to authenticate Git operations.
	GitAuth *GitAuthSpec `json:"gitAuth,omitempty"`

	// TTLSecondsAfterFinished controls automatic HarnessRun deletion.
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`
}

type HarnessRunStatus struct {
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Phase              HarnessRunPhase    `json:"phase,omitempty"`
	PodName            string             `json:"podName,omitempty"`
	StartTime          *metav1.Time       `json:"startTime,omitempty"`
	CompletionTime     *metav1.Time       `json:"completionTime,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}

type HarnessRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarnessRun `json:"items"`
}

func (in *Session) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(Session)
	in.DeepCopyInto(out)
	return out
}

func (in *SessionList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(SessionList)
	in.DeepCopyInto(out)
	return out
}

func (in *HarnessRun) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(HarnessRun)
	in.DeepCopyInto(out)
	return out
}

func (in *HarnessRunList) DeepCopyObject() runtime.Object {
	if in == nil {
		return nil
	}
	out := new(HarnessRunList)
	in.DeepCopyInto(out)
	return out
}
