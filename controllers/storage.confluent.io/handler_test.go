package controllers

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	storageconfluentiov1 "github.com/druid-io/druid-operator/apis/storage.confluent.io/v1"
	"github.com/ghodss/yaml"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func setEmptyResource(obj interface{}) interface{} {
	switch obj.(type) {
	case *appsv1.Deployment:
		deployment := obj.(*appsv1.Deployment)
		for i, _ := range deployment.Spec.Template.Spec.Containers {
			deployment.Spec.Template.Spec.Containers[i].Resources = corev1.ResourceRequirements{}
		}
		return deployment
	case *appsv1.StatefulSet:
		statefulset := obj.(*appsv1.StatefulSet)
		for i, _ := range statefulset.Spec.Template.Spec.Containers {
			statefulset.Spec.Template.Spec.Containers[i].Resources = corev1.ResourceRequirements{}
		}
		return statefulset
	}
	return nil
}

func TestMakeDaemonsetForENP(t *testing.T) {
	clusterSpec := readSampleLSSpec(t)
	eksNvmeProvisionerPodSpec, eksNvmeProvisionerLabels := makeEKSNVMEProvisioner(clusterSpec)

	actual, _ := makeDaemonSet(clusterSpec, eksNvmeProvisionerPodSpec, eksNvmeProvisionerLabels)
	addHashToObject(actual)

	actual.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{}

	expected := new(appsv1.DaemonSet)
	readAndUnmarshallResource("testdata/enp-daemonset.yaml", &expected, t)

	assertEquals(expected, actual, t)

}

func TestMakeDaemonsetForLVP(t *testing.T) {
	clusterSpec := readSampleLSSpec(t)

	localVolumeProvisionerPodSpec, localVolumeProvisionerLabels := makeLocalVolumeProvisioner(clusterSpec)

	actual, _ := makeDaemonSet(clusterSpec, localVolumeProvisionerPodSpec, localVolumeProvisionerLabels)
	addHashToObject(actual)
	updatedActual := setEmptyResource(&actual.Spec.Template.Spec)

	expected := new(appsv1.DaemonSet)
	readAndUnmarshallResource("testdata/lvp-daemonset.yaml", &expected, t)
	updatedExpected := setEmptyResource(&expected.Spec.Template.Spec)

	assertEquals(updatedExpected, updatedActual, t)
}

func TestMakeDeploymentForNodeGrabber(t *testing.T) {
	clusterSpec := readSampleLSSpec(t)

	nodeGrabberPodSpec, nodeGrabberLabels := makeNodeGrabber(clusterSpec)

	actual, _ := makeDeployment(clusterSpec, nodeGrabberPodSpec, nodeGrabberLabels)
	addHashToObject(actual)
	updatedActual := setEmptyResource(&actual.Spec.Template.Spec)

	expected := new(appsv1.Deployment)
	readAndUnmarshallResource("testdata/nodegrabber-deployment.yaml", &expected, t)
	updatedExpected := setEmptyResource(&expected.Spec.Template.Spec)

	assertEquals(updatedExpected, updatedActual, t)
}

func TestMakeConfigMap(t *testing.T) {
	clusterSpec := readSampleLSSpec(t)

	configMapData, configMapLabels := makeLocalVolumeProvisionerConfigMap(clusterSpec)
	actual, _ := makeConfigMap(clusterSpec, configMapLabels, configMapData)
	addHashToObject(actual)

	expected := new(corev1.ConfigMap)
	readAndUnmarshallResource("testdata/lvp-config-map.yaml", &expected, t)
	assertEquals(expected, actual, t)
}

func readSampleLSSpec(t *testing.T) *storageconfluentiov1.LocalStorage {
	return readLSSpecFromFile(t, "testdata/localstorage-smoke-test-cluster.yaml")
}

func readLSSpecFromFile(t *testing.T, filePath string) *storageconfluentiov1.LocalStorage {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		t.Errorf("Failed to read druid cluster spec")
	}

	clusterSpec := new(storageconfluentiov1.LocalStorage)
	err = yaml.Unmarshal(bytes, &clusterSpec)
	if err != nil {
		t.Errorf("Failed to unmarshall druid cluster spec: %v", err)
	}

	return clusterSpec
}

func readAndUnmarshallResource(file string, res interface{}, t *testing.T) {
	bytes, err := ioutil.ReadFile(file)
	if err != nil {
		t.Errorf("Failed to read file[%s]", file)
	}

	err = yaml.Unmarshal(bytes, res)
	if err != nil {
		t.Errorf("Failed to unmarshall resource from file[%s]", file)
	}
}

func assertEquals(expected, actual interface{}, t *testing.T) {
	if !reflect.DeepEqual(expected, actual) {
		t.Error("Match failed!.")
	}
}

func dumpResource(res interface{}, t *testing.T) {
	bytes, err := yaml.Marshal(res)
	if err != nil {
		t.Errorf("failed to marshall: %v", err)
	}
	fmt.Println(string(bytes))
}
