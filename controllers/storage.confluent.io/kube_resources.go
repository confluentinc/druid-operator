package controllers

import (
	storageconfluentiov1 "github.com/druid-io/druid-operator/apis/storage.confluent.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func makeLocalStorageEmptyObj() *storageconfluentiov1.LocalStorage {
	return &storageconfluentiov1.LocalStorage{
		TypeMeta: metav1.TypeMeta{
			Kind:       "LocalStorage",
			APIVersion: "storage.confluent.io/v1",
		},
	}
}

func makeConfigMap(name string, namespace string, labels map[string]string, data map[string]string) (*v1.ConfigMap, error) {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}, nil
}
func makeDaemonSet(m *storageconfluentiov1.LocalStorage, p *v1.PodSpec, ls map[string]string) (*appsv1.DaemonSet, error) {
	daemonSetSpec := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Daemonset",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeNodeSpecificUniqueString(m, ls["name"]),
			Namespace: m.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: *p,
			},
		},
	}
	return daemonSetSpec, nil
}

func makeStorageClass(m *storageconfluentiov1.LocalStorage) (*storagev1.StorageClass, error) {
	volumeBindMode := storagev1.VolumeBindingWaitForFirstConsumer
	reclaimPolicy := v1.PersistentVolumeReclaimDelete
	storageClassSpec := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: m.Spec.StorageClassName,
		},
		Provisioner:       "kubernetes.io/no-provisioner",
		VolumeBindingMode: &volumeBindMode,
		ReclaimPolicy:     &reclaimPolicy,
	}
	return storageClassSpec, nil
}

func makeStorageClassEmptyObj() *storagev1.StorageClass {
	return &storagev1.StorageClass{}
}

func makeStorageClassListEmptyObj() *storagev1.StorageClassList {
	return &storagev1.StorageClassList{}
}

func makeDeployment(m *storageconfluentiov1.LocalStorage, p *v1.PodSpec, ls map[string]string) (*appsv1.Deployment, error) {
	deploymentSpec := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeNodeSpecificUniqueString(m, ls["name"]),
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: *p,
			},
		},
	}
	return deploymentSpec, nil
}

func makeDeploymentEmptyObj() *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
	}
}

func makeDaemonSetEmptyObj() *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
	}
}

func makeDaemonSetListEmptyObj() *appsv1.DaemonSetList {
	return &appsv1.DaemonSetList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
	}
}

func makeDeploymentListEmptyObj() *appsv1.DeploymentList {
	return &appsv1.DeploymentList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
}

func makePVListEmptyObj() *v1.PersistentVolumeList {
	return &v1.PersistentVolumeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
	}
}

func makePVEmptyObj() *v1.PersistentVolume {
	return &v1.PersistentVolume{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PersistentVolume",
			APIVersion: "v1",
		},
	}
}

func makeConfigMapEmptyObj() *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
	}
}

func makePodList() *v1.PodList {
	return &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}
