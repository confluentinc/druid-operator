package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"

	"gopkg.in/yaml.v3"

	storageconfluentiov1 "github.com/druid-io/druid-operator/apis/storage.confluent.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("storage_operator_handler")

const (
	localStorageResourceHash = "localStorageResourceHash"
	eksNvmeProvisioner       = "eks-nvme-provisioner"
	localVolumeProvisioner   = "local-volume-provisioner"
	nodeGrabber              = "node-grabber"
	resourceCreated          = "CREATED"
	resourceUpdated          = "UPDATED"
	nodeSelectorLabel        = "beta.kubernetes.io/instance-type"
	finalizerName            = "finalizers.confluent.io"
)

func verifySpec(m *storageconfluentiov1.LocalStorage) error {
	keyValidationRegex, err := regexp.Compile("[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*")
	if err != nil {
		return err
	}

	errorMsg := ""

	if !keyValidationRegex.MatchString(m.Spec.Name) {
		errorMsg = fmt.Sprintf("%sName[%s] must match k8s resource name regex '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'", errorMsg, m.Spec.Name)
	}

	if m.Spec.InstanceType == "" {
		errorMsg = fmt.Sprintf("%sInstanceType missing from LocalStorage Spec\n", errorMsg)
	}

	if m.Spec.Replicas < 0 {
		errorMsg = fmt.Sprintf("%sReplica count less than 0\n", errorMsg)
	}

	if errorMsg == "" {
		return nil
	} else {
		return fmt.Errorf(errorMsg)
	}
}

func stringifyForLogging(obj object, ls *storageconfluentiov1.LocalStorage) string {
	if bytes, err := json.Marshal(obj); err != nil {
		logger.Error(err, err.Error(), fmt.Sprintf("Failed to serialize [%s:%s]", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName()), "name", ls.Name, "namespace", ls.Namespace)
		return fmt.Sprintf("%v", obj)
	} else {
		return string(bytes)
	}

}

func getNodeSelector(m *storageconfluentiov1.LocalStorage) map[string]string {
	return map[string]string{
		nodeSelectorLabel: m.Spec.InstanceType,
	}
}

func addCommonLabels(labels map[string]string, m *storageconfluentiov1.LocalStorage) map[string]string {
	labels["cr-name"] = m.Name
	labels["operator-version"] = "storage.confluent.io-v1"
	labels["operator-name"] = "LocalStorage"
	return labels
}

func addOperatorLabels(m *storageconfluentiov1.LocalStorage, resourceName string) map[string]string {
	labels := map[string]string{
		"name": resourceName,
	}
	labels = addCommonLabels(labels, m)
	return labels
}

func makeNodeSpecificUniqueString(m *storageconfluentiov1.LocalStorage, key string) string {
	return fmt.Sprintf("local-storage-%s-%s", key, m.Name)
}

func makeEKSNVMEProvisioner(m *storageconfluentiov1.LocalStorage) (*v1.PodSpec, map[string]string) {
	labels := addOperatorLabels(m, eksNvmeProvisioner)
	hostPathUnset := v1.HostPathUnset
	mountPropagation := v1.MountPropagationBidirectional
	_true := true
	podSpec := &v1.PodSpec{
		NodeSelector:       getNodeSelector(m),
		ServiceAccountName: m.Spec.ServiceAccountName,
		Volumes: []v1.Volume{
			{
				Name: "pv-disks",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/pv-disks",
						Type: &hostPathUnset,
					},
				},
			},
			{
				Name: "nvme",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/nvme",
						Type: &hostPathUnset,
					},
				},
			},
		},
		Containers: []v1.Container{
			{
				Image:           m.Spec.EKSImage,
				ImagePullPolicy: v1.PullAlways,
				Name:            eksNvmeProvisioner,
				VolumeMounts: []v1.VolumeMount{
					{
						MountPath:        "/pv-disks",
						MountPropagation: &mountPropagation,
						Name:             "pv-disks",
					},
					{
						MountPath:        "/nvme",
						MountPropagation: &mountPropagation,
						Name:             "nvme",
					},
				},
				Resources: v1.ResourceRequirements{},
				SecurityContext: &v1.SecurityContext{
					Privileged: &_true,
				},
			},
		},
	}
	return podSpec, labels
}

func makeLocalVolumeProvisioner(m *storageconfluentiov1.LocalStorage) (*v1.PodSpec, map[string]string) {
	labels := addOperatorLabels(m, localVolumeProvisioner)
	hostPathUnset := v1.HostPathUnset
	configMapVolumeMode := int32(0420)
	mountPropagation := v1.MountPropagationHostToContainer
	_true := true
	podSpec := &v1.PodSpec{
		NodeSelector:       getNodeSelector(m),
		ServiceAccountName: m.Spec.ServiceAccountName,
		Volumes: []v1.Volume{
			{
				Name: "local-storage",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/pv-disks",
						Type: &hostPathUnset,
					},
				},
			},
			{
				Name: "provisioner-dev",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/dev",
						Type: &hostPathUnset,
					},
				},
			},
			{
				Name: "provisioner-config",
				VolumeSource: v1.VolumeSource{
					ConfigMap: &v1.ConfigMapVolumeSource{
						DefaultMode: &configMapVolumeMode,
						LocalObjectReference: v1.LocalObjectReference{
							Name: localVolumeProvisioner,
						},
					},
				},
			},
		},
		Containers: []v1.Container{
			{
				Image:           m.Spec.LocalVolumeImage,
				ImagePullPolicy: v1.PullAlways,
				Name:            localVolumeProvisioner,
				Env: []v1.EnvVar{
					{
						Name: "MY_NODE_NAME",
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "spec.nodeName",
							},
						},
					},
					{
						Name: "MY_NAMESPACE",
						ValueFrom: &v1.EnvVarSource{
							FieldRef: &v1.ObjectFieldSelector{
								APIVersion: "v1",
								FieldPath:  "metadata.namespace",
							},
						},
					},
				},
				VolumeMounts: []v1.VolumeMount{
					{
						MountPath: "/etc/provisioner/config",
						Name:      "provisioner-config",
						ReadOnly:  true,
					},
					{
						MountPath: "/dev",
						Name:      "provisioner-dev",
					},
					{
						MountPath:        "/pv-disks",
						MountPropagation: &mountPropagation,
						Name:             "local-storage",
					},
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
						v1.ResourceMemory: *resource.NewQuantity(100*1024*1024, resource.BinarySI),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
						v1.ResourceMemory: *resource.NewQuantity(200*1024*1024, resource.BinarySI),
					},
				},
				SecurityContext: &v1.SecurityContext{
					Privileged: &_true,
				},
			},
		},
	}
	return podSpec, labels
}

func makeNodeGrabber(m *storageconfluentiov1.LocalStorage) (*v1.PodSpec, map[string]string) {
	labels := addOperatorLabels(m, nodeGrabber)
	podSpec := &v1.PodSpec{
		NodeSelector: getNodeSelector(m),
		Containers: []v1.Container{{
			Image:           m.Spec.NodeGrabberImage,
			Command:         []string{"tail", "-f", "/dev/null"},
			ImagePullPolicy: v1.PullAlways,
			Name:            nodeGrabber,
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
					v1.ResourceMemory: *resource.NewQuantity(50*1024*1024, resource.BinarySI),
				},
			},
			Ports: []v1.ContainerPort{{
				ContainerPort: 28140,
				Name:          "hello",
				HostPort:      28140,
				Protocol:      "TCP",
			}},
		}},
	}
	return podSpec, labels
}

func makeLocalVolumeProvisionerConfigMap(m *storageconfluentiov1.LocalStorage) (map[string]string, map[string]string) {
	labels := addOperatorLabels(m, localVolumeProvisioner)
	storageClassMap := fmt.Sprintf(`{"%s": {"hostDir": "/pv-disks", "mountDir": "/pv-disks"}}`, m.Spec.StorageClassName)
	var _data interface{}
	_ = yaml.Unmarshal([]byte(storageClassMap), &_data)
	yamlBytes, _ := yaml.Marshal(_data)
	jsonStr, _ := json.Marshal(labels)
	data := map[string]string{
		"setPVOwnerRef":   "true",
		"storageClassMap": string(yamlBytes),
		"useNodeNameOnly": "true",
		"labelsForPV":     string(jsonStr),
	}
	return data, labels
}

func deleteUnusedResources(sdk client.Client, drd *storageconfluentiov1.LocalStorage,
	names map[string]bool, selectorLabels map[string]string, emptyListObjFn func() objectList, itemsExtractorFn func(obj runtime.Object) []object) []string {

	listOpts := []client.ListOption{
		client.InNamespace(drd.Namespace),
		client.MatchingLabels(selectorLabels),
	}

	survivorNames := make([]string, 0, len(names))

	listObj := emptyListObjFn()

	if err := sdk.List(context.TODO(), listObj, listOpts...); err != nil {
		e := fmt.Errorf("failed to list [%s] due to [%s]", listObj.GetObjectKind().GroupVersionKind().Kind, err.Error())
		logger.Error(e, e.Error(), "name", drd.Name, "namespace", drd.Namespace)
	} else {
		for _, s := range itemsExtractorFn(listObj) {
			if !names[s.GetName()] {
				if err := sdk.Delete(context.TODO(), s, &client.DeleteOptions{}); err != nil {
					survivorNames = append(survivorNames, s.GetName())
				}
			} else {
				survivorNames = append(survivorNames, s.GetName())
			}
		}
	}

	return survivorNames
}

func statusPatcher(sdk client.Client, updatedStatus storageconfluentiov1.LocalStorageStatus, m *storageconfluentiov1.LocalStorage) error {

	if !reflect.DeepEqual(updatedStatus, m.Status) {
		patchBytes, err := json.Marshal(map[string]storageconfluentiov1.LocalStorageStatus{"status": updatedStatus})
		if err != nil {
			return fmt.Errorf("failed to serialize status patch to bytes: %v", err)
		}
		if err := sdk.Status().Patch(context.TODO(), m, client.RawPatch(types.MergePatchType, patchBytes)); err != nil {
			return err
		}
	}
	return nil
}

func updaterCondition(m *storageconfluentiov1.LocalStorage, prevObj object, obj object, isEqualFn func(prev, curr object) bool) bool {
	// update the resources if ForceDeploy is set to true
	if m.Spec.ForceDeploy {
		return true
	}
	if obj.GetAnnotations()[localStorageResourceHash] != prevObj.GetAnnotations()[localStorageResourceHash] && !isEqualFn(prevObj, obj) {
		return true
	}
	return false
}

func sdkCreateOrUpdateAsNeeded(
	sdk client.Client,
	objFn func() (object, error),
	emptyObjFn func() object,
	isEqualFn func(prev, curr object) bool,
	updaterFn func(prev, curr object),
	ls *storageconfluentiov1.LocalStorage,
	names map[string]bool) (string, error) {
	if obj, err := objFn(); err != nil {
		return "", err
	} else {
		names[obj.GetName()] = true

		addOwnerRefToObject(obj, asOwner(ls))
		addHashToObject(obj)

		prevObj := emptyObjFn()
		if err := sdk.Get(context.TODO(), *namespacedName(obj.GetName(), obj.GetNamespace()), prevObj); err != nil {
			if apierrors.IsNotFound(err) {
				// resource does not exist, create it.
				err := sdk.Create(context.TODO(), obj)
				if err != nil {
					return "", err
				} else {
					return resourceCreated, nil
				}
			} else {
				e := fmt.Errorf("failed to get [%s:%s] due to [%s]", obj.GetObjectKind().GroupVersionKind().Kind, obj.GetName(), err.Error())
				logger.Error(e, e.Error(), "Prev object", stringifyForLogging(prevObj, ls), "name", ls.Name, "namespace", ls.Namespace)
				return "", e
			}
		} else {
			// resource already exists, updated it if needed
			if updaterCondition(ls, prevObj, obj, isEqualFn) {
				obj.SetResourceVersion(prevObj.GetResourceVersion())
				updaterFn(prevObj, obj)
				if err := sdk.Update(context.TODO(), obj); err != nil {
					return "", err
				} else {
					return resourceUpdated, err
				}
			} else {
				return "", nil
			}
		}
	}
}

func listObjects(ctx context.Context, sdk client.Client, drd *storageconfluentiov1.LocalStorage, selectorLabels map[string]string, emptyListObjFn func() objectList, ListObjFn func(obj runtime.Object) []object) ([]object, error) {
	listOpts := []client.ListOption{
		client.InNamespace(drd.Namespace),
		client.MatchingLabels(selectorLabels),
	}
	listObj := emptyListObjFn()

	if err := sdk.List(ctx, listObj, listOpts...); err != nil {
		return nil, err
	}

	return ListObjFn(listObj), nil
}

func listObjectsWithoutNamespace(ctx context.Context, sdk client.Client, drd *storageconfluentiov1.LocalStorage, selectorLabels map[string]string, emptyListObjFn func() objectList, ListObjFn func(obj runtime.Object) []object) ([]object, error) {

	listOpts := []client.ListOption{
		client.MatchingLabels(selectorLabels),
	}
	listObj := emptyListObjFn()

	if err := sdk.List(ctx, listObj, listOpts...); err != nil {
		return nil, err
	}

	return ListObjFn(listObj), nil
}

func deleteMarkedResource(sdk client.Client, drd *storageconfluentiov1.LocalStorage, dsList, depList, pvList []object) error {

	for i := range dsList {
		if err := sdk.Delete(context.TODO(), dsList[i], &client.DeleteAllOfOptions{}); err != nil {
			return err
		}
	}

	for i := range depList {
		if err := sdk.Delete(context.TODO(), depList[i], &client.DeleteAllOfOptions{}); err != nil {
			return err
		}
	}

	for i := range pvList {
		if err := sdk.Delete(context.TODO(), pvList[i], &client.DeleteAllOfOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func executeFinalizers(sdk client.Client, m *storageconfluentiov1.LocalStorage) error {

	if ContainsString(m.ObjectMeta.Finalizers, finalizerName) {
		storageLabels := addCommonLabels(map[string]string{}, m)
		dsList, err := listObjects(context.TODO(), sdk, m, storageLabels, func() objectList { return makeDaemonSetListEmptyObj() }, func(listObj runtime.Object) []object {
			items := listObj.(*appsv1.DaemonSetList).Items
			result := make([]object, len(items))
			for i := 0; i < len(items); i++ {
				result[i] = &items[i]
			}
			return result
		})
		if err != nil {
			return err
		}

		depList, err := listObjects(context.TODO(), sdk, m, storageLabels, func() objectList { return makeDeploymentListEmptyObj() }, func(listObj runtime.Object) []object {
			items := listObj.(*appsv1.DeploymentList).Items
			result := make([]object, len(items))
			for i := 0; i < len(items); i++ {
				result[i] = &items[i]
			}
			return result
		})
		if err != nil {
			return err
		}

		pvList, err := listObjectsWithoutNamespace(context.TODO(), sdk, m, storageLabels, func() objectList { return makePVListEmptyObj() }, func(listObj runtime.Object) []object {
			items := listObj.(*v1.PersistentVolumeList).Items
			result := make([]object, len(items))
			for i := 0; i < len(items); i++ {
				result[i] = &items[i]
			}
			return result
		})
		if err != nil {
			return err
		}

		msg := fmt.Sprintf("Trigerring finalizer for CR [%s] in namespace [%s]", m.Name, m.Namespace)
		logger.Info(msg)
		if err := deleteMarkedResource(sdk, m, dsList, depList, pvList); err != nil {
			return err
		} else {
			msg := fmt.Sprintf("Finalizer success for CR [%s] in namespace [%s]", m.Name, m.Namespace)
			logger.Info(msg)
		}

		// remove our finalizer from the list and update it.
		m.ObjectMeta.Finalizers = RemoveString(m.ObjectMeta.Finalizers, finalizerName)

		if err := sdk.Update(context.TODO(), m); err != nil {
			return err
		}

	}
	return nil

}

func checkIfCRExists(sdk client.Client, m *storageconfluentiov1.LocalStorage) bool {
	if err := sdk.Get(context.TODO(), *namespacedName(m.Name, m.Namespace), makeLocalStorageEmptyObj()); err != nil {
		return false
	} else {
		return true
	}
}

func getNamesFromMap(obj map[string]bool) []string {
	var names []string
	for p, _ := range obj {
		names = append(names, p)
	}
	return names
}

func deployCluster(sdk client.Client, m *storageconfluentiov1.LocalStorage) error {
	err := verifySpec(m)
	if err != nil {
		e := fmt.Errorf("invalid Spec[%s:%s] due to [%s]", m.Kind, m.Name, err.Error())
		logger.Error(err, e.Error())
		return nil
	}
	daemonsetNames := make(map[string]bool)
	deploymentNames := make(map[string]bool)
	configMapNames := make(map[string]bool)
	storageClassNames := make(map[string]bool)
	pvNames := make(map[string]bool)

	md := m.GetDeletionTimestamp() != nil
	if md {
		return executeFinalizers(sdk, m)
	}
	cr := checkIfCRExists(sdk, m)
	if cr {
		if !ContainsString(m.ObjectMeta.Finalizers, finalizerName) {
			m.SetFinalizers(append(m.GetFinalizers(), finalizerName))
			if err := sdk.Update(context.TODO(), m); err != nil {
				return err
			}
		}
	}

	configMapData, configMapLabels := makeLocalVolumeProvisionerConfigMap(m)

	if _, err := sdkCreateOrUpdateAsNeeded(sdk,
		func() (object, error) {
			return makeConfigMap(m, configMapLabels, configMapData)
		},
		func() object { return makeConfigMapEmptyObj() },
		genericEqualFn, noopUpdaterFn, m, configMapNames); err != nil {
		return err
	}

	if _, err := sdkCreateOrUpdateAsNeeded(sdk,
		func() (object, error) {
			return makeStorageClass(m)
		},
		func() object { return makeStorageClassEmptyObj() },
		genericEqualFn, noopUpdaterFn, m, storageClassNames); err != nil {
		return err
	}

	eksNvmeProvisionerPodSpec, eksNvmeProvisionerLabels := makeEKSNVMEProvisioner(m)
	localVolumeProvisionerPodSpec, localVolumeProvisionerLabels := makeLocalVolumeProvisioner(m)
	nodeGrabberPodSpec, nodeGrabberLabels := makeNodeGrabber(m)

	if _, err := sdkCreateOrUpdateAsNeeded(sdk,
		func() (object, error) {
			return makeDaemonSet(m, eksNvmeProvisionerPodSpec, eksNvmeProvisionerLabels)
		},
		func() object { return makeDaemonSetEmptyObj() },
		genericEqualFn, noopUpdaterFn, m, daemonsetNames); err != nil {
		return err
	}

	if _, err := sdkCreateOrUpdateAsNeeded(sdk,
		func() (object, error) {
			return makeDaemonSet(m, localVolumeProvisionerPodSpec, localVolumeProvisionerLabels)
		},
		func() object { return makeDaemonSetEmptyObj() },
		genericEqualFn, noopUpdaterFn, m, daemonsetNames); err != nil {
		return err
	}

	if _, err := sdkCreateOrUpdateAsNeeded(sdk,
		func() (object, error) {
			return makeDeployment(m, nodeGrabberPodSpec, nodeGrabberLabels)
		},
		func() object { return makeDeploymentEmptyObj() },
		deploymentEqualFn, noopUpdaterFn, m, deploymentNames); err != nil {
		return err
	}
	commonLabels := addCommonLabels(map[string]string{}, m)
	pvList, err := listObjects(context.TODO(), sdk, m, commonLabels, func() objectList { return makePVListEmptyObj() }, func(listObj runtime.Object) []object {
		items := listObj.(*v1.PersistentVolumeList).Items
		result := make([]object, len(items))
		for i := 0; i < len(items); i++ {
			result[i] = &items[i]
		}
		return result
	})
	if err != nil {
		return err
	}
	for i := range pvList {
		if _, err := sdkCreateOrUpdateAsNeeded(sdk,
			func() (object, error) { return pvList[i], nil },
			func() object { return makePVEmptyObj() },
			genericEqualFn, noopUpdaterFn, m, pvNames); err != nil {
			return err
		}
	}

	updatedStatus := storageconfluentiov1.LocalStorageStatus{}

	listOpts := []client.ListOption{
		client.InNamespace(m.Namespace),
		client.MatchingLabels(addCommonLabels(map[string]string{}, m)),
	}
	emptyPodList := makePodList()
	if err := sdk.List(context.TODO(), emptyPodList, listOpts...); err != nil {
		return err
	}
	podItems := emptyPodList.Items
	podList := make([]object, len(podItems))
	for i := 0; i < len(podItems); i++ {
		podList[i] = &podItems[i]
	}

	updatedStatus.Pods = getPodNames(podList)
	sort.Strings(updatedStatus.Pods)

	configMapNamesList := getNamesFromMap(configMapNames)
	daemonSetNamesList := getNamesFromMap(daemonsetNames)
	deploymentNamesList := getNamesFromMap(deploymentNames)
	pvNamesList := getNamesFromMap(pvNames)
	storageClassNamesList := getNamesFromMap(storageClassNames)

	updatedStatus.ConfigMaps = configMapNamesList
	sort.Strings(updatedStatus.ConfigMaps)

	updatedStatus.DaemonSets = daemonSetNamesList
	sort.Strings(updatedStatus.DaemonSets)

	updatedStatus.Deployments = deploymentNamesList
	sort.Strings(updatedStatus.Deployments)

	updatedStatus.PersistentVolumes = pvNamesList
	sort.Strings(updatedStatus.PersistentVolumes)

	updatedStatus.StorageClasses = storageClassNamesList
	sort.Strings(updatedStatus.StorageClasses)

	err = statusPatcher(sdk, updatedStatus, m)
	if err != nil {
		return err
	}
	return nil
}
