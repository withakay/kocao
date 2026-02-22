package controllers

const (
	FinalizerName = "kocao.withakay.github.com/finalizer"

	LabelSessionName = "kocao.withakay.github.com/session"
)

const (
	AnnotationAttachEnabled = "kocao.withakay.github.com/attach-enabled"
	AnnotationEgressMode    = "kocao.withakay.github.com/egress-mode"
	AnnotationEgressHosts   = "kocao.withakay.github.com/egress-allowed-hosts"
)

const (
	ConditionReady     = "Ready"
	ConditionRunning   = "Running"
	ConditionSucceeded = "Succeeded"
	ConditionFailed    = "Failed"
	ConditionSession   = "SessionReady"
)
