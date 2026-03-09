package controllers

const (
	FinalizerName = "kocao.withakay.github.com/finalizer"

	LabelWorkspaceSessionName = "kocao.withakay.github.com/workspace-session"
	LabelDisplayName          = "kocao.withakay.github.com/display-name"
	LabelSymphonyProjectName  = "kocao.withakay.github.com/symphony-project"
	LabelSymphonyProjectUID   = "kocao.withakay.github.com/symphony-project-uid"
	LabelSymphonyItemID       = "kocao.withakay.github.com/symphony-item-id"
	LabelGitHubRepository     = "kocao.withakay.github.com/github-repository"
	LabelGitHubIssueNumber    = "kocao.withakay.github.com/github-issue-number"
)

const (
	AnnotationAttachEnabled     = "kocao.withakay.github.com/attach-enabled"
	AnnotationEgressMode        = "kocao.withakay.github.com/egress-mode"
	AnnotationEgressHosts       = "kocao.withakay.github.com/egress-allowed-hosts"
	AnnotationSymphonyIssueURL  = "kocao.withakay.github.com/symphony-issue-url"
	AnnotationSymphonyIssueNode = "kocao.withakay.github.com/symphony-issue-node-id"

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
	ConditionSession   = "WorkspaceSessionReady"
	ConditionConfig    = "ConfigReady"
	ConditionSource    = "SourceSynced"
	ConditionLifecycle = "OrchestrationReady"
)
