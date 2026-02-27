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
