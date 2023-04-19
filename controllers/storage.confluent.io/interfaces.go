package controllers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type object interface {
	metav1.Object
	runtime.Object
}

type objectList interface {
	metav1.ListInterface
	runtime.Object
}
