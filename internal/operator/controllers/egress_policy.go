package controllers

import (
	"context"
	"os"
	"strings"

	operatorv1alpha1 "github.com/withakay/kocao/internal/operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const envGitHubEgressCIDRs = "CP_GITHUB_EGRESS_CIDRS"

const (
	egressModeRestricted = "restricted"
	egressModeFull       = "full"
)

func normalizeEgressMode(mode string) string {
	m := strings.ToLower(strings.TrimSpace(mode))
	switch m {
	case "", "github", "github-only", "restricted", "deny-by-default":
		return egressModeRestricted
	case "full", "full-internet", "internet":
		return egressModeFull
	default:
		// Unknown values default to restricted.
		return egressModeRestricted
	}
}

func runEgressNetworkPolicyName(runName string) string {
	base := sanitizeDNSLabel(runName)
	if base == "" {
		base = "run"
	}
	return base + "-egress"
}

func githubEgressCIDRs() []string {
	v := strings.TrimSpace(os.Getenv(envGitHubEgressCIDRs))
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// Very lightweight validation: require a slash.
		if !strings.Contains(p, "/") {
			continue
		}
		out = append(out, p)
	}
	return out
}

func desiredRunEgressNetworkPolicy(run *operatorv1alpha1.HarnessRun, mode string) *networkingv1.NetworkPolicy {
	labels := map[string]string{
		"app.kubernetes.io/managed-by":  "kocao-control-plane-operator",
		"app.kubernetes.io/name":        "kocao-harness-egress",
		"kocao.withakay.github.com/run": run.Name,
	}

	policyTypes := []networkingv1.PolicyType{networkingv1.PolicyTypeEgress}
	podSelector := metav1.LabelSelector{MatchLabels: map[string]string{"kocao.withakay.github.com/run": run.Name}}

	mode = normalizeEgressMode(mode)
	if mode == egressModeFull {
		// A single empty rule allows all egress.
		rules := []networkingv1.NetworkPolicyEgressRule{{}}
		return &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{Name: runEgressNetworkPolicyName(run.Name), Namespace: run.Namespace, Labels: labels},
			Spec:       networkingv1.NetworkPolicySpec{PodSelector: podSelector, PolicyTypes: policyTypes, Egress: rules},
		}
	}

	// Restricted baseline: default-deny + allow DNS + configured GitHub CIDRs.
	egress := make([]networkingv1.NetworkPolicyEgressRule, 0, 2)

	// DNS (UDP/TCP 53) to kube-system. Keep it broad to avoid coupling to CNI-specific labels.
	dnsNSSelector := &metav1.LabelSelector{MatchLabels: map[string]string{"kubernetes.io/metadata.name": "kube-system"}}
	dnsPorts := []networkingv1.NetworkPolicyPort{
		{Protocol: protoPtr(corev1.ProtocolUDP), Port: intstrPtr(53)},
		{Protocol: protoPtr(corev1.ProtocolTCP), Port: intstrPtr(53)},
	}
	egress = append(egress, networkingv1.NetworkPolicyEgressRule{
		To:    []networkingv1.NetworkPolicyPeer{{NamespaceSelector: dnsNSSelector}},
		Ports: dnsPorts,
	})

	// GitHub allowlist (admin-provided CIDRs).
	ports := []networkingv1.NetworkPolicyPort{
		{Protocol: protoPtr(corev1.ProtocolTCP), Port: intstrPtr(443)},
		{Protocol: protoPtr(corev1.ProtocolTCP), Port: intstrPtr(22)},
	}
	cidrs := githubEgressCIDRs()
	if len(cidrs) != 0 {
		peers := make([]networkingv1.NetworkPolicyPeer, 0, len(cidrs))
		for _, cidr := range cidrs {
			c := cidr
			peers = append(peers, networkingv1.NetworkPolicyPeer{IPBlock: &networkingv1.IPBlock{CIDR: c}})
		}
		egress = append(egress, networkingv1.NetworkPolicyEgressRule{To: peers, Ports: ports})
	}

	return &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: runEgressNetworkPolicyName(run.Name), Namespace: run.Namespace, Labels: labels},
		Spec:       networkingv1.NetworkPolicySpec{PodSelector: podSelector, PolicyTypes: policyTypes, Egress: egress},
	}
}

func ensureRunEgressNetworkPolicy(ctx context.Context, c client.Client, scheme *runtime.Scheme, run *operatorv1alpha1.HarnessRun, mode string) error {
	desired := desiredRunEgressNetworkPolicy(run, mode)
	if err := controllerutil.SetControllerReference(run, desired, scheme); err != nil {
		return err
	}

	var existing networkingv1.NetworkPolicy
	err := c.Get(ctx, client.ObjectKey{Namespace: desired.Namespace, Name: desired.Name}, &existing)
	if apierrors.IsNotFound(err) {
		return c.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Overwrite spec/labels in-place for determinism.
	updated := existing.DeepCopy()
	updated.Labels = desired.Labels
	updated.Spec = desired.Spec
	return c.Patch(ctx, updated, client.MergeFrom(&existing))
}

func protoPtr(p corev1.Protocol) *corev1.Protocol {
	pp := p
	return &pp
}

func intstrPtr(port int) *intstr.IntOrString {
	v := intstr.FromInt(port)
	return &v
}
