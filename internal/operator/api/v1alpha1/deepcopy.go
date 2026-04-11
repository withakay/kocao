package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

func (in *Session) DeepCopyInto(out *Session) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status.ObservedGeneration = in.Status.ObservedGeneration
	out.Status.Phase = in.Status.Phase
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
}

func (in *Session) DeepCopy() *Session {
	if in == nil {
		return nil
	}
	out := new(Session)
	in.DeepCopyInto(out)
	return out
}

func (in *SessionList) DeepCopyInto(out *SessionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Session, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *SessionList) DeepCopy() *SessionList {
	if in == nil {
		return nil
	}
	out := new(SessionList)
	in.DeepCopyInto(out)
	return out
}

func (in *HarnessRun) DeepCopyInto(out *HarnessRun) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec.WorkspaceSessionName = in.Spec.WorkspaceSessionName
	out.Spec.RepoURL = in.Spec.RepoURL
	out.Spec.RepoRevision = in.Spec.RepoRevision
	out.Spec.Image = in.Spec.Image
	if in.Spec.Command != nil {
		out.Spec.Command = append([]string(nil), in.Spec.Command...)
	}
	if in.Spec.Args != nil {
		out.Spec.Args = append([]string(nil), in.Spec.Args...)
	}
	out.Spec.WorkingDir = in.Spec.WorkingDir
	if in.Spec.Env != nil {
		out.Spec.Env = make([]EnvVar, len(in.Spec.Env))
		copy(out.Spec.Env, in.Spec.Env)
	}
	if in.Spec.GitAuth != nil {
		out.Spec.GitAuth = &GitAuthSpec{
			SecretName:  in.Spec.GitAuth.SecretName,
			TokenKey:    in.Spec.GitAuth.TokenKey,
			UsernameKey: in.Spec.GitAuth.UsernameKey,
		}
	}
	if in.Spec.AgentAuth != nil {
		out.Spec.AgentAuth = &AgentAuthSpec{
			ApiKeySecretName: in.Spec.AgentAuth.ApiKeySecretName,
			OauthSecretName:  in.Spec.AgentAuth.OauthSecretName,
		}
	}
	if in.Spec.AgentSession != nil {
		out.Spec.AgentSession = &AgentSessionSpec{
			Runtime: in.Spec.AgentSession.Runtime,
			Agent:   in.Spec.AgentSession.Agent,
		}
	}
	out.Spec.EgressMode = in.Spec.EgressMode
	if in.Spec.TTLSecondsAfterFinished != nil {
		v := *in.Spec.TTLSecondsAfterFinished
		out.Spec.TTLSecondsAfterFinished = &v
	}

	out.Status.ObservedGeneration = in.Status.ObservedGeneration
	out.Status.Phase = in.Status.Phase
	out.Status.PodName = in.Status.PodName
	if in.Status.StartTime != nil {
		out.Status.StartTime = &metav1.Time{Time: in.Status.StartTime.Time}
	}
	if in.Status.CompletionTime != nil {
		out.Status.CompletionTime = &metav1.Time{Time: in.Status.CompletionTime.Time}
	}
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
	if in.Status.AgentSession != nil {
		out.Status.AgentSession = &AgentSessionStatus{
			Runtime:   in.Status.AgentSession.Runtime,
			Agent:     in.Status.AgentSession.Agent,
			SessionID: in.Status.AgentSession.SessionID,
			Phase:     in.Status.AgentSession.Phase,
		}
	}
}

func (in *HarnessRun) DeepCopy() *HarnessRun {
	if in == nil {
		return nil
	}
	out := new(HarnessRun)
	in.DeepCopyInto(out)
	return out
}

func (in *HarnessRunList) DeepCopyInto(out *HarnessRunList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]HarnessRun, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *HarnessRunList) DeepCopy() *HarnessRunList {
	if in == nil {
		return nil
	}
	out := new(HarnessRunList)
	in.DeepCopyInto(out)
	return out
}

func (in *SymphonyProject) DeepCopyInto(out *SymphonyProject) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec.Paused = in.Spec.Paused
	out.Spec.Source = in.Spec.Source
	if in.Spec.Source.ActiveStates != nil {
		out.Spec.Source.ActiveStates = append([]string(nil), in.Spec.Source.ActiveStates...)
	}
	if in.Spec.Source.TerminalStates != nil {
		out.Spec.Source.TerminalStates = append([]string(nil), in.Spec.Source.TerminalStates...)
	}
	if in.Spec.Repositories != nil {
		out.Spec.Repositories = make([]SymphonyProjectRepositorySpec, len(in.Spec.Repositories))
		for i := range in.Spec.Repositories {
			out.Spec.Repositories[i] = in.Spec.Repositories[i]
			if in.Spec.Repositories[i].GitAuth != nil {
				out.Spec.Repositories[i].GitAuth = &GitAuthSpec{
					SecretName:  in.Spec.Repositories[i].GitAuth.SecretName,
					TokenKey:    in.Spec.Repositories[i].GitAuth.TokenKey,
					UsernameKey: in.Spec.Repositories[i].GitAuth.UsernameKey,
				}
			}
			if in.Spec.Repositories[i].AgentAuth != nil {
				out.Spec.Repositories[i].AgentAuth = &AgentAuthSpec{
					ApiKeySecretName: in.Spec.Repositories[i].AgentAuth.ApiKeySecretName,
					OauthSecretName:  in.Spec.Repositories[i].AgentAuth.OauthSecretName,
				}
			}
		}
	}
	out.Spec.Runtime = in.Spec.Runtime
	if in.Spec.Runtime.Command != nil {
		out.Spec.Runtime.Command = append([]string(nil), in.Spec.Runtime.Command...)
	}
	if in.Spec.Runtime.Args != nil {
		out.Spec.Runtime.Args = append([]string(nil), in.Spec.Runtime.Args...)
	}
	if in.Spec.Runtime.Env != nil {
		out.Spec.Runtime.Env = make([]EnvVar, len(in.Spec.Runtime.Env))
		copy(out.Spec.Runtime.Env, in.Spec.Runtime.Env)
	}
	if in.Spec.Runtime.TTLSecondsAfterFinished != nil {
		v := *in.Spec.Runtime.TTLSecondsAfterFinished
		out.Spec.Runtime.TTLSecondsAfterFinished = &v
	}

	out.Status = in.Status
	if in.Status.Conditions != nil {
		out.Status.Conditions = make([]metav1.Condition, len(in.Status.Conditions))
		copy(out.Status.Conditions, in.Status.Conditions)
	}
	if in.Status.LastSyncTime != nil {
		out.Status.LastSyncTime = &metav1.Time{Time: in.Status.LastSyncTime.Time}
	}
	if in.Status.LastSuccessfulSync != nil {
		out.Status.LastSuccessfulSync = &metav1.Time{Time: in.Status.LastSuccessfulSync.Time}
	}
	if in.Status.NextSyncTime != nil {
		out.Status.NextSyncTime = &metav1.Time{Time: in.Status.NextSyncTime.Time}
	}
	if in.Status.ActiveClaims != nil {
		out.Status.ActiveClaims = make([]SymphonyProjectClaimStatus, len(in.Status.ActiveClaims))
		for i := range in.Status.ActiveClaims {
			out.Status.ActiveClaims[i] = in.Status.ActiveClaims[i]
			if in.Status.ActiveClaims[i].ClaimedAt != nil {
				out.Status.ActiveClaims[i].ClaimedAt = &metav1.Time{Time: in.Status.ActiveClaims[i].ClaimedAt.Time}
			}
			if in.Status.ActiveClaims[i].LastUpdatedTime != nil {
				out.Status.ActiveClaims[i].LastUpdatedTime = &metav1.Time{Time: in.Status.ActiveClaims[i].LastUpdatedTime.Time}
			}
		}
	}
	if in.Status.RetryQueue != nil {
		out.Status.RetryQueue = make([]SymphonyProjectRetryStatus, len(in.Status.RetryQueue))
		for i := range in.Status.RetryQueue {
			out.Status.RetryQueue[i] = in.Status.RetryQueue[i]
			if in.Status.RetryQueue[i].ReadyAt != nil {
				out.Status.RetryQueue[i].ReadyAt = &metav1.Time{Time: in.Status.RetryQueue[i].ReadyAt.Time}
			}
			if in.Status.RetryQueue[i].LastErrorTime != nil {
				out.Status.RetryQueue[i].LastErrorTime = &metav1.Time{Time: in.Status.RetryQueue[i].LastErrorTime.Time}
			}
		}
	}
	if in.Status.RecentErrors != nil {
		out.Status.RecentErrors = make([]SymphonyProjectErrorStatus, len(in.Status.RecentErrors))
		for i := range in.Status.RecentErrors {
			out.Status.RecentErrors[i] = in.Status.RecentErrors[i]
			if in.Status.RecentErrors[i].LastErrorTime != nil {
				out.Status.RecentErrors[i].LastErrorTime = &metav1.Time{Time: in.Status.RecentErrors[i].LastErrorTime.Time}
			}
		}
	}
	if in.Status.RecentEvents != nil {
		out.Status.RecentEvents = make([]SymphonyProjectEventStatus, len(in.Status.RecentEvents))
		for i := range in.Status.RecentEvents {
			out.Status.RecentEvents[i] = in.Status.RecentEvents[i]
			if in.Status.RecentEvents[i].ObservedTime != nil {
				out.Status.RecentEvents[i].ObservedTime = &metav1.Time{Time: in.Status.RecentEvents[i].ObservedTime.Time}
			}
		}
	}
	if in.Status.RecentSkips != nil {
		out.Status.RecentSkips = make([]SymphonyProjectSkipStatus, len(in.Status.RecentSkips))
		for i := range in.Status.RecentSkips {
			out.Status.RecentSkips[i] = in.Status.RecentSkips[i]
			if in.Status.RecentSkips[i].ObservedTime != nil {
				out.Status.RecentSkips[i].ObservedTime = &metav1.Time{Time: in.Status.RecentSkips[i].ObservedTime.Time}
			}
		}
	}
	if in.Status.UnsupportedRepos != nil {
		out.Status.UnsupportedRepos = append([]string(nil), in.Status.UnsupportedRepos...)
	}
}

func (in *SymphonyProject) DeepCopy() *SymphonyProject {
	if in == nil {
		return nil
	}
	out := new(SymphonyProject)
	in.DeepCopyInto(out)
	return out
}

func (in *SymphonyProjectList) DeepCopyInto(out *SymphonyProjectList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]SymphonyProject, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *SymphonyProjectList) DeepCopy() *SymphonyProjectList {
	if in == nil {
		return nil
	}
	out := new(SymphonyProjectList)
	in.DeepCopyInto(out)
	return out
}
