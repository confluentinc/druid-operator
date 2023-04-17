package controllers

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"

	storageconfluentiov1 "github.com/druid-io/druid-operator/apis/storage.confluent.io/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func namespacedName(name, namespace string) *types.NamespacedName {
	return &types.NamespacedName{Name: name, Namespace: namespace}
}

func asOwner(m *storageconfluentiov1.LocalStorage) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

func addHashToObject(obj metav1.Object) error {
	if sha, err := getObjectHash(obj); err != nil {
		return err
	} else {
		annotations := obj.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
			obj.SetAnnotations(annotations)
		}
		annotations[localStorageResourceHash] = sha
		return nil
	}
}

func getObjectHash(obj metav1.Object) (string, error) {
	if bytes, err := json.Marshal(obj); err != nil {
		return "", err
	} else {
		sha1Bytes := sha1.Sum(bytes)
		return base64.StdEncoding.EncodeToString(sha1Bytes[:]), nil
	}
}

func getPodNames(pods []object) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.(*v1.Pod).Name)
	}
	return podNames
}

func deploymentEqualFn(obj1 object, obj2 object) bool {
	if (obj1.GetAnnotations()[nodeSelectorLabel] == obj2.GetAnnotations()[nodeSelectorLabel]) && (obj1.GetObjectKind() == obj2.GetObjectKind()) && (obj1.GetNamespace() == obj2.GetNamespace()) {
		return true
	}
	return false
}

func genericEqualFn(obj1 object, obj2 object) bool {
	return true
}

func noopUpdaterFn(prev, curr object) {

}
func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
