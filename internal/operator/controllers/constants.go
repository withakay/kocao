package controllers

const (
	FinalizerName = "kocao.withakay.github.com/finalizer"

	LabelSessionName = "kocao.withakay.github.com/session"
)

const (
	AnnotationAttachEnabled = "kocao.withakay.github.com/attach-enabled"
	AnnotationEgressMode    = "kocao.withakay.github.com/egress-mode"
	AnnotationEgressHosts   = "kocao.withakay.github.com/egress-allowed-hosts"

	// GitHub outcome metadata is reported by the harness (or external automation)
	// and surfaced through the control-plane API for UI visibility.
	AnnotationGitHubBranch      = "kocao.withakay.github.com/github-branch"
	AnnotationPullRequestURL    = "kocao.withakay.github.com/pull-request-url"
	AnnotationPullRequestStatus = "kocao.withakay.github.com/pull-request-status"
)

const (
	ConditionReady     = "Ready"
	ConditionRunning   = "Running"
	ConditionSucceeded = "Succeeded"
	ConditionFailed    = "Failed"
	ConditionSession   = "SessionReady"
)
