package controllers

import (
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func setCondition(conds *[]metav1.Condition, c metav1.Condition) {
	meta.SetStatusCondition(conds, c)
}

func clearCondition(conds *[]metav1.Condition, typ string) {
	meta.RemoveStatusCondition(conds, typ)
}
