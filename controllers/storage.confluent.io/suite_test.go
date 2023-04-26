/*

 */

package controllers

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	storageconfluentiov1 "github.com/druid-io/druid-operator/apis/storage.confluent.io/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

type TestK8sEnvCtx struct {
	restConfig *rest.Config
	k8sManager manager.Manager
	env        *envtest.Environment
}

func setupK8Evn(t *testing.T, testK8sCtx *TestK8sEnvCtx) {
	ctrl.SetLogger(zap.New())
	testK8sCtx.env = &envtest.Environment{
		CRDInstallOptions: envtest.CRDInstallOptions{
			Paths: []string{filepath.Join("..", "..", "deploy", "crds", "storage.confluent.io.apache.org_localstorages.yaml")},
		},
	}

	config, err := testK8sCtx.env.Start()
	require.NoError(t, err)

	testK8sCtx.restConfig = config
}

func destroyK8Env(t *testing.T, testK8sCtx *TestK8sEnvCtx) {
	testK8sCtx.env.Stop()
}

func TestAPIs(t *testing.T) {
	testK8sCtx := &TestK8sEnvCtx{}

	setupK8Evn(t, testK8sCtx)
	defer destroyK8Env(t, testK8sCtx)

	err := storageconfluentiov1.AddToScheme(scheme.Scheme)
	require.NoError(t, err)

	testK8sCtx.k8sManager, err = ctrl.NewManager(testK8sCtx.restConfig, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	require.NoError(t, err)

	setupLocalStorageOperator(t, testK8sCtx)

	go func() {
		err = testK8sCtx.k8sManager.Start(ctrl.SetupSignalHandler())
		if err != nil {
			fmt.Printf("problem starting mgr [%v] \n", err)
		}
		require.NoError(t, err)
	}()

	testLocalStorageOperator(t, testK8sCtx)
}

func setupLocalStorageOperator(t *testing.T, testK8sCtx *TestK8sEnvCtx) {
	err := (&LocalStorageReconciler{
		Client:        testK8sCtx.k8sManager.GetClient(),
		Log:           ctrl.Log.WithName("controllers").WithName("LocalStorage"),
		Scheme:        testK8sCtx.k8sManager.GetScheme(),
		ReconcileWait: LookupReconcileTime(),
	}).SetupWithManager(testK8sCtx.k8sManager)

	require.NoError(t, err)
}

func testLocalStorageOperator(t *testing.T, testK8sCtx *TestK8sEnvCtx) {
	lsCR := readLSSpecFromFile(t, "testdata/localstorage-smoke-test-cluster.yaml")

	k8sClient := testK8sCtx.k8sManager.GetClient()

	err := k8sClient.Create(context.TODO(), lsCR)
	require.NoError(t, err)

	err = retry(func() error {
		ls := &storageconfluentiov1.LocalStorage{}

		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: lsCR.Name, Namespace: lsCR.Namespace}, ls)
		if err != nil {
			return err
		}

		expectedConfigMaps := []string{
			fmt.Sprintf("local-storage-%s-%s", localVolumeProvisioner, lsCR.Name),
		}
		if !areStringArraysEqual(ls.Status.ConfigMaps, expectedConfigMaps) {
			return errors.New(fmt.Sprintf("Failed to get expected ConfigMaps, got [%v]", ls.Status.ConfigMaps))
		}

		expectedDaemonSets := []string{
			fmt.Sprintf("local-storage-%s-%s", localVolumeProvisioner, lsCR.Name),
			fmt.Sprintf("local-storage-%s-%s", eksNvmeProvisioner, lsCR.Name),
		}
		if !areStringArraysEqual(ls.Status.DaemonSets, expectedDaemonSets) {
			return errors.New(fmt.Sprintf("Failed to get expected StatefulSets, got [%v]", ls.Status.DaemonSets))
		}

		expectedDeployments := []string{
			fmt.Sprintf("local-storage-%s-%s", nodeGrabber, lsCR.Name),
		}
		if !areStringArraysEqual(ls.Status.Deployments, expectedDeployments) {
			return errors.New(fmt.Sprintf("Failed to get expected Deployments, got [%v]", ls.Status.Deployments))
		}

		expectedSC := []string{
			lsCR.Spec.StorageClassName,
		}
		if !areStringArraysEqual(ls.Status.StorageClasses, expectedSC) {
			return errors.New(fmt.Sprintf("Failed to get expected SCs, got [%v]", ls.Status.StorageClasses))
		}

		return nil
	}, time.Millisecond*250, time.Second*45)

	require.NoError(t, err)
}

func areStringArraysEqual(a1, a2 []string) bool {
	if len(a1) == len(a2) {
		for i, v := range a1 {
			if v != a2[i] {
				return false
			}
		}
	} else {
		return false
	}
	return true
}

func retry(fn func() error, retryWait, timeOut time.Duration) error {
	timeOutTimestamp := time.Now().Add(timeOut)

	for {
		if err := fn(); err != nil {
			if time.Now().Before(timeOutTimestamp) {
				time.Sleep(retryWait)
			} else {
				return err
			}
		} else {
			return nil
		}
	}
}

/*
var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.New())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = storageconfluentiov1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(k8sClient).ToNot(BeNil())

	close(done)
}, 60)
*/

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
