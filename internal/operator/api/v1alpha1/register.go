package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	Group   = "kocao.withakay.github.com"
	Version = "v1alpha1"
)

var (
	GroupVersion = schema.GroupVersion{Group: Group, Version: Version}
)

func AddToScheme(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion, &Session{}, &SessionList{}, &HarnessRun{}, &HarnessRunList{})
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}
