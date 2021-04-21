/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testsuites

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis/volumesnapshot/v1beta1"
	snapshotclientset "github.com/kubernetes-csi/external-snapshotter/v2/pkg/client/clientset/versioned"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	apps "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	clientset "k8s.io/client-go/kubernetes"
	restclientset "k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	deploymentutil "k8s.io/kubernetes/pkg/controller/deployment/util"
	"k8s.io/kubernetes/pkg/kubelet/events"
	"k8s.io/kubernetes/test/e2e/framework"
	e2eevents "k8s.io/kubernetes/test/e2e/framework/events"
	e2elog "k8s.io/kubernetes/test/e2e/framework/log"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	e2epv "k8s.io/kubernetes/test/e2e/framework/pv"
	testutil "k8s.io/kubernetes/test/utils"
	imageutils "k8s.io/kubernetes/test/utils/image"
	"sigs.k8s.io/azuredisk-csi-driver/pkg/apis/azuredisk/v1alpha1"
	v1alpha1ClientSet "sigs.k8s.io/azuredisk-csi-driver/pkg/apis/client/clientset/versioned/typed/azuredisk/v1alpha1"
	"sigs.k8s.io/azuredisk-csi-driver/pkg/azuredisk"
	controller "sigs.k8s.io/azuredisk-csi-driver/pkg/controller"
)

const (
	execTimeout = 10 * time.Second
	// Some pods can take much longer to get ready due to volume attach/detach latency.
	slowPodStartTimeout = 10 * time.Minute
	// Description that will printed during tests
	failedConditionDescription = "Error status code"

	poll            = 2 * time.Second
	pollLongTimeout = 5 * time.Minute
	pollTimeout     = 10 * time.Minute
)

type TestStorageClass struct {
	client       clientset.Interface
	storageClass *storagev1.StorageClass
	namespace    *v1.Namespace
}

func NewTestStorageClass(c clientset.Interface, ns *v1.Namespace, sc *storagev1.StorageClass) *TestStorageClass {
	return &TestStorageClass{
		client:       c,
		storageClass: sc,
		namespace:    ns,
	}
}

func (t *TestStorageClass) Create() storagev1.StorageClass {
	var err error

	ginkgo.By("creating a StorageClass " + t.storageClass.Name)
	t.storageClass, err = t.client.StorageV1().StorageClasses().Create(context.TODO(), t.storageClass, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return *t.storageClass
}

func (t *TestStorageClass) Cleanup() {
	e2elog.Logf("deleting StorageClass %s", t.storageClass.Name)
	err := t.client.StorageV1().StorageClasses().Delete(context.TODO(), t.storageClass.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

type TestVolumeSnapshotClass struct {
	client              restclientset.Interface
	volumeSnapshotClass *v1beta1.VolumeSnapshotClass
	namespace           *v1.Namespace
}

func NewTestVolumeSnapshotClass(c restclientset.Interface, ns *v1.Namespace, vsc *v1beta1.VolumeSnapshotClass) *TestVolumeSnapshotClass {
	return &TestVolumeSnapshotClass{
		client:              c,
		volumeSnapshotClass: vsc,
		namespace:           ns,
	}
}

func (t *TestVolumeSnapshotClass) Create() {
	ginkgo.By("creating a VolumeSnapshotClass")
	var err error
	t.volumeSnapshotClass, err = snapshotclientset.New(t.client).SnapshotV1beta1().VolumeSnapshotClasses().Create(context.TODO(), t.volumeSnapshotClass, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestVolumeSnapshotClass) CreateSnapshot(pvc *v1.PersistentVolumeClaim) *v1beta1.VolumeSnapshot {
	ginkgo.By("creating a VolumeSnapshot for " + pvc.Name)
	snapshot := &v1beta1.VolumeSnapshot{
		TypeMeta: metav1.TypeMeta{
			Kind:       VolumeSnapshotKind,
			APIVersion: SnapshotAPIVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "volume-snapshot-",
			Namespace:    t.namespace.Name,
		},
		Spec: v1beta1.VolumeSnapshotSpec{
			VolumeSnapshotClassName: &t.volumeSnapshotClass.Name,
			Source: v1beta1.VolumeSnapshotSource{
				PersistentVolumeClaimName: &pvc.Name,
			},
		},
	}
	snapshot, err := snapshotclientset.New(t.client).SnapshotV1beta1().VolumeSnapshots(t.namespace.Name).Create(context.TODO(), snapshot, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return snapshot
}

func (t *TestVolumeSnapshotClass) ReadyToUse(snapshot *v1beta1.VolumeSnapshot) {
	ginkgo.By("waiting for VolumeSnapshot to be ready to use - " + snapshot.Name)
	err := wait.Poll(15*time.Second, 5*time.Minute, func() (bool, error) {
		vs, err := snapshotclientset.New(t.client).SnapshotV1beta1().VolumeSnapshots(t.namespace.Name).Get(context.TODO(), snapshot.Name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("did not see ReadyToUse: %v", err)
		}
		return *vs.Status.ReadyToUse, nil
	})
	framework.ExpectNoError(err)
}

func (t *TestVolumeSnapshotClass) DeleteSnapshot(vs *v1beta1.VolumeSnapshot) {
	ginkgo.By("deleting a VolumeSnapshot " + vs.Name)
	err := snapshotclientset.New(t.client).SnapshotV1beta1().VolumeSnapshots(t.namespace.Name).Delete(context.TODO(), vs.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestVolumeSnapshotClass) Cleanup() {
	// skip deleting volume snapshot storage class otherwise snapshot e2e test will fail, details:
	// https://github.com/kubernetes-sigs/azuredisk-csi-driver/pull/260#issuecomment-583296932
	e2elog.Logf("skip deleting VolumeSnapshotClass %s", t.volumeSnapshotClass.Name)
	//err := snapshotclientset.New(t.client).SnapshotV1beta1().VolumeSnapshotClasses().Delete(t.volumeSnapshotClass.Name, nil)
	//framework.ExpectNoError(err)
}

type TestPreProvisionedPersistentVolume struct {
	client                    clientset.Interface
	persistentVolume          *v1.PersistentVolume
	requestedPersistentVolume *v1.PersistentVolume
}

func NewTestPreProvisionedPersistentVolume(c clientset.Interface, pv *v1.PersistentVolume) *TestPreProvisionedPersistentVolume {
	return &TestPreProvisionedPersistentVolume{
		client:                    c,
		requestedPersistentVolume: pv,
	}
}

func (pv *TestPreProvisionedPersistentVolume) Create() v1.PersistentVolume {
	var err error
	ginkgo.By("creating a PV")
	pv.persistentVolume, err = pv.client.CoreV1().PersistentVolumes().Create(context.TODO(), pv.requestedPersistentVolume, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	return *pv.persistentVolume
}

type TestPersistentVolumeClaim struct {
	client                         clientset.Interface
	claimSize                      string
	volumeMode                     v1.PersistentVolumeMode
	storageClass                   *storagev1.StorageClass
	namespace                      *v1.Namespace
	persistentVolume               *v1.PersistentVolume
	persistentVolumeClaim          *v1.PersistentVolumeClaim
	requestedPersistentVolumeClaim *v1.PersistentVolumeClaim
	dataSource                     *v1.TypedLocalObjectReference
}

func NewTestPersistentVolumeClaim(c clientset.Interface, ns *v1.Namespace, claimSize string, volumeMode VolumeMode, sc *storagev1.StorageClass) *TestPersistentVolumeClaim {
	mode := v1.PersistentVolumeFilesystem
	if volumeMode == Block {
		mode = v1.PersistentVolumeBlock
	}
	return &TestPersistentVolumeClaim{
		client:       c,
		claimSize:    claimSize,
		volumeMode:   mode,
		namespace:    ns,
		storageClass: sc,
	}
}

func NewTestPersistentVolumeClaimWithDataSource(c clientset.Interface, ns *v1.Namespace, claimSize string, volumeMode VolumeMode, sc *storagev1.StorageClass, dataSource *v1.TypedLocalObjectReference) *TestPersistentVolumeClaim {
	mode := v1.PersistentVolumeFilesystem
	if volumeMode == Block {
		mode = v1.PersistentVolumeBlock
	}
	return &TestPersistentVolumeClaim{
		client:       c,
		claimSize:    claimSize,
		volumeMode:   mode,
		namespace:    ns,
		storageClass: sc,
		dataSource:   dataSource,
	}
}

func (t *TestPersistentVolumeClaim) Create() {
	var err error

	ginkgo.By("creating a PVC")
	storageClassName := ""
	if t.storageClass != nil {
		storageClassName = t.storageClass.Name
	}
	t.requestedPersistentVolumeClaim = generatePVC(t.namespace.Name, storageClassName, t.claimSize, t.volumeMode, t.dataSource)
	t.persistentVolumeClaim, err = t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Create(context.TODO(), t.requestedPersistentVolumeClaim, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestPersistentVolumeClaim) ValidateProvisionedPersistentVolume() {
	var err error

	// Get the bound PersistentVolume
	ginkgo.By("validating provisioned PV")
	t.persistentVolume, err = t.client.CoreV1().PersistentVolumes().Get(context.TODO(), t.persistentVolumeClaim.Spec.VolumeName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	// Check sizes
	expectedCapacity := t.requestedPersistentVolumeClaim.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	claimCapacity := t.persistentVolumeClaim.Spec.Resources.Requests[v1.ResourceName(v1.ResourceStorage)]
	gomega.Expect(claimCapacity.Value()).To(gomega.Equal(expectedCapacity.Value()), "claimCapacity is not equal to requestedCapacity")

	pvCapacity := t.persistentVolume.Spec.Capacity[v1.ResourceName(v1.ResourceStorage)]
	gomega.Expect(pvCapacity.Value()).To(gomega.Equal(expectedCapacity.Value()), "pvCapacity is not equal to requestedCapacity")

	// Check PV properties
	ginkgo.By("checking the PV")
	expectedAccessModes := t.requestedPersistentVolumeClaim.Spec.AccessModes
	gomega.Expect(t.persistentVolume.Spec.AccessModes).To(gomega.Equal(expectedAccessModes))
	gomega.Expect(t.persistentVolume.Spec.ClaimRef.Name).To(gomega.Equal(t.persistentVolumeClaim.ObjectMeta.Name))
	gomega.Expect(t.persistentVolume.Spec.ClaimRef.Namespace).To(gomega.Equal(t.persistentVolumeClaim.ObjectMeta.Namespace))
	// If storageClass is nil, PV was pre-provisioned with these values already set
	if t.storageClass != nil {
		gomega.Expect(t.persistentVolume.Spec.PersistentVolumeReclaimPolicy).To(gomega.Equal(*t.storageClass.ReclaimPolicy))
		gomega.Expect(t.persistentVolume.Spec.MountOptions).To(gomega.Equal(t.storageClass.MountOptions))
		if *t.storageClass.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
			gomega.Expect(t.persistentVolume.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Values).
				To(gomega.HaveLen(1))
		}
		if len(t.storageClass.AllowedTopologies) > 0 {
			gomega.Expect(t.persistentVolume.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Key).
				To(gomega.Equal(t.storageClass.AllowedTopologies[0].MatchLabelExpressions[0].Key))
			for _, v := range t.persistentVolume.Spec.NodeAffinity.Required.NodeSelectorTerms[0].MatchExpressions[0].Values {
				gomega.Expect(t.storageClass.AllowedTopologies[0].MatchLabelExpressions[0].Values).To(gomega.ContainElement(v))
			}

		}
	}
}

func (t *TestPersistentVolumeClaim) WaitForBound() v1.PersistentVolumeClaim {
	var err error

	ginkgo.By(fmt.Sprintf("waiting for PVC to be in phase %q", v1.ClaimBound))
	err = e2epv.WaitForPersistentVolumeClaimPhase(v1.ClaimBound, t.client, t.namespace.Name, t.persistentVolumeClaim.Name, framework.Poll, framework.ClaimProvisionTimeout)
	framework.ExpectNoError(err)

	ginkgo.By("checking the PVC")
	// Get new copy of the claim
	t.persistentVolumeClaim, err = t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Get(context.TODO(), t.persistentVolumeClaim.Name, metav1.GetOptions{})
	framework.ExpectNoError(err)

	return *t.persistentVolumeClaim
}

func generatePVC(namespace, storageClassName, claimSize string, volumeMode v1.PersistentVolumeMode, dataSource *v1.TypedLocalObjectReference) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "pvc-",
			Namespace:    namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(claimSize),
				},
			},
			VolumeMode: &volumeMode,
			DataSource: dataSource,
		},
	}
}

func generateStatefulSetPVC(namespace, storageClassName, claimSize string, volumeMode v1.PersistentVolumeMode, dataSource *v1.TypedLocalObjectReference) *v1.PersistentVolumeClaim {
	return &v1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pvc",
			Namespace: namespace,
		},
		Spec: v1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClassName,
			AccessModes: []v1.PersistentVolumeAccessMode{
				v1.ReadWriteOnce,
			},
			Resources: v1.ResourceRequirements{
				Requests: v1.ResourceList{
					v1.ResourceName(v1.ResourceStorage): resource.MustParse(claimSize),
				},
			},
			VolumeMode: &volumeMode,
			DataSource: dataSource,
		},
	}
}

func (t *TestPersistentVolumeClaim) Cleanup() {
	// Since PV is created after pod creation when the volume binding mode is WaitForFirstConsumer,
	// we need to populate fields such as PVC and PV info in TestPersistentVolumeClaim, and valid it
	if t.storageClass != nil && *t.storageClass.VolumeBindingMode == storagev1.VolumeBindingWaitForFirstConsumer {
		var err error
		t.persistentVolumeClaim, err = t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Get(context.TODO(), t.persistentVolumeClaim.Name, metav1.GetOptions{})
		framework.ExpectNoError(err)
		t.ValidateProvisionedPersistentVolume()
	}
	e2elog.Logf("deleting PVC %q/%q", t.namespace.Name, t.persistentVolumeClaim.Name)
	err := e2epv.DeletePersistentVolumeClaim(t.client, t.persistentVolumeClaim.Name, t.namespace.Name)
	framework.ExpectNoError(err)
	// Wait for the PV to get deleted if reclaim policy is Delete. (If it's
	// Retain, there's no use waiting because the PV won't be auto-deleted and
	// it's expected for the caller to do it.) Technically, the first few delete
	// attempts may fail, as the volume is still attached to a node because
	// kubelet is slowly cleaning up the previous pod, however it should succeed
	// in a couple of minutes.
	if t.persistentVolume.Spec.PersistentVolumeReclaimPolicy == v1.PersistentVolumeReclaimDelete {
		ginkgo.By(fmt.Sprintf("waiting for claim's PV %q to be deleted", t.persistentVolume.Name))
		err := e2epv.WaitForPersistentVolumeDeleted(t.client, t.persistentVolume.Name, 5*time.Second, 10*time.Minute)
		framework.ExpectNoError(err)
	}
	// Wait for the PVC to be deleted
	err = waitForPersistentVolumeClaimDeleted(t.client, t.persistentVolumeClaim.Name, t.namespace.Name, 5*time.Second, 5*time.Minute)
	framework.ExpectNoError(err)
}

func (t *TestPersistentVolumeClaim) ReclaimPolicy() v1.PersistentVolumeReclaimPolicy {
	return t.persistentVolume.Spec.PersistentVolumeReclaimPolicy
}

func (t *TestPersistentVolumeClaim) WaitForPersistentVolumePhase(phase v1.PersistentVolumePhase) {
	err := e2epv.WaitForPersistentVolumePhase(phase, t.client, t.persistentVolume.Name, 5*time.Second, 10*time.Minute)
	framework.ExpectNoError(err)
}

func (t *TestPersistentVolumeClaim) DeleteBoundPersistentVolume() {
	ginkgo.By(fmt.Sprintf("deleting PV %q", t.persistentVolume.Name))
	err := e2epv.DeletePersistentVolume(t.client, t.persistentVolume.Name)
	framework.ExpectNoError(err)
	ginkgo.By(fmt.Sprintf("waiting for claim's PV %q to be deleted", t.persistentVolume.Name))
	err = e2epv.WaitForPersistentVolumeDeleted(t.client, t.persistentVolume.Name, 5*time.Second, 10*time.Minute)
	framework.ExpectNoError(err)
}

func (t *TestPersistentVolumeClaim) DeleteBackingVolume(driver azuredisk.CSIDriver) {
	volumeID := t.persistentVolume.Spec.CSI.VolumeHandle
	ginkgo.By(fmt.Sprintf("deleting azuredisk volume %q", volumeID))
	req := &csi.DeleteVolumeRequest{
		VolumeId: volumeID,
	}
	_, err := driver.DeleteVolume(context.Background(), req)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("could not delete volume %q: %v", volumeID, err))
	}
}

type TestDeployment struct {
	client     clientset.Interface
	deployment *apps.Deployment
	namespace  *v1.Namespace
	podName    string
}

func NewTestDeployment(c clientset.Interface, ns *v1.Namespace, command string, pvc *v1.PersistentVolumeClaim, volumeName, mountPath string, readOnly, isWindows bool, useCMD bool, schedulerName string) *TestDeployment {
	generateName := "azuredisk-volume-tester-"
	selectorValue := fmt.Sprintf("%s%d", generateName, rand.Int())
	replicas := int32(1)
	testDeployment := &TestDeployment{
		client:    c,
		namespace: ns,
		deployment: &apps.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
			},
			Spec: apps.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": selectorValue},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": selectorValue},
					},
					Spec: v1.PodSpec{
						SchedulerName: schedulerName,
						NodeSelector:  map[string]string{"kubernetes.io/os": "linux"},
						Containers: []v1.Container{
							{
								Name:    "volume-tester",
								Image:   imageutils.GetE2EImage(imageutils.BusyBox),
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      volumeName,
										MountPath: mountPath,
										ReadOnly:  readOnly,
									},
								},
							},
						},
						RestartPolicy: v1.RestartPolicyAlways,
						Volumes: []v1.Volume{
							{
								Name: volumeName,
								VolumeSource: v1.VolumeSource{
									PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
										ClaimName: pvc.Name,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if isWindows {
		testDeployment.deployment.Spec.Template.Spec.NodeSelector = map[string]string{
			"kubernetes.io/os": "windows",
		}
		testDeployment.deployment.Spec.Template.Spec.Containers[0].Image = "e2eteam/busybox:1.29"
		if useCMD {
			testDeployment.deployment.Spec.Template.Spec.Containers[0].Command = []string{"cmd"}
			testDeployment.deployment.Spec.Template.Spec.Containers[0].Args = []string{"/c", command}
		} else {
			testDeployment.deployment.Spec.Template.Spec.Containers[0].Command = []string{"powershell.exe"}
			testDeployment.deployment.Spec.Template.Spec.Containers[0].Args = []string{"-Command", command}
		}
	}

	return testDeployment
}

func (t *TestDeployment) Create() {
	var err error
	t.deployment, err = t.client.AppsV1().Deployments(t.namespace.Name).Create(context.TODO(), t.deployment, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	err = testutil.WaitForDeploymentComplete(t.client, t.deployment, e2elog.Logf, poll, pollLongTimeout)
	framework.ExpectNoError(err)
	pods, err := getPodsForDeployment(t.client, t.deployment)
	framework.ExpectNoError(err)
	// always get first pod as there should only be one
	t.podName = pods.Items[0].Name
}

func (t *TestDeployment) WaitForPodReady() {
	pods, err := getPodsForDeployment(t.client, t.deployment)
	framework.ExpectNoError(err)
	// always get first pod as there should only be one
	pod := pods.Items[0]
	t.podName = pod.Name
	err = e2epod.WaitForPodRunningInNamespace(t.client, &pod)
	framework.ExpectNoError(err)
}

func (t *TestDeployment) Exec(command []string, expectedString string) {
	_, err := framework.LookForStringInPodExec(t.namespace.Name, t.podName, command, expectedString, execTimeout)
	framework.ExpectNoError(err)
}

func (t *TestDeployment) DeletePodAndWait() {
	e2elog.Logf("Deleting pod %q in namespace %q", t.podName, t.namespace.Name)
	err := t.client.CoreV1().Pods(t.namespace.Name).Delete(context.TODO(), t.podName, metav1.DeleteOptions{})
	if err != nil {
		if !apierrs.IsNotFound(err) {
			framework.ExpectNoError(fmt.Errorf("pod %q Delete API error: %v", t.podName, err))
		}
		return
	}
	e2elog.Logf("Waiting for pod %q in namespace %q to be fully deleted", t.podName, t.namespace.Name)
	err = e2epod.WaitForPodNoLongerRunningInNamespace(t.client, t.podName, t.namespace.Name)
	if err != nil {
		if !apierrs.IsNotFound(err) {
			framework.ExpectNoError(fmt.Errorf("pod %q error waiting for delete: %v", t.podName, err))
		}
	}
}

func (t *TestDeployment) Cleanup() {
	e2elog.Logf("deleting Deployment %q/%q", t.namespace.Name, t.deployment.Name)
	body, err := t.Logs()
	if err != nil {
		e2elog.Logf("Error getting logs for pod %s: %v", t.podName, err)
	} else {
		e2elog.Logf("Pod %s has the following logs: %s", t.podName, body)
	}
	err = t.client.AppsV1().Deployments(t.namespace.Name).Delete(context.TODO(), t.deployment.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestDeployment) Logs() ([]byte, error) {
	return podLogs(t.client, t.podName, t.namespace.Name)
}

type TestStatefulset struct {
	client      clientset.Interface
	statefulset *apps.StatefulSet
	namespace   *v1.Namespace
	podName     string
}

func NewTestStatefulset(c clientset.Interface, ns *v1.Namespace, command string, pvc *v1.PersistentVolumeClaim, volumeName, mountPath string, readOnly, isWindows, useCMD bool, schedulerName string) *TestStatefulset {
	generateName := "azuredisk-volume-tester-"
	selectorValue := fmt.Sprintf("%s%d", generateName, rand.Int())
	replicas := int32(1)
	var volumeClaimTest []v1.PersistentVolumeClaim
	volumeClaimTest = append(volumeClaimTest, *pvc)
	testStatefulset := &TestStatefulset{
		client:    c,
		namespace: ns,
		statefulset: &apps.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: generateName,
			},
			Spec: apps.StatefulSetSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": selectorValue},
				},
				Template: v1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app": selectorValue},
					},
					Spec: v1.PodSpec{
						SchedulerName: schedulerName,
						NodeSelector:  map[string]string{"kubernetes.io/os": "linux"},
						Containers: []v1.Container{
							{
								Name:    "volume-tester",
								Image:   imageutils.GetE2EImage(imageutils.BusyBox),
								Command: []string{"/bin/sh"},
								Args:    []string{"-c", command},
								VolumeMounts: []v1.VolumeMount{
									{
										Name:      volumeName,
										MountPath: mountPath,
										ReadOnly:  readOnly,
									},
								},
							},
						},
						RestartPolicy: v1.RestartPolicyAlways,
					},
				},
				VolumeClaimTemplates: volumeClaimTest,
			},
		},
	}

	if isWindows {
		testStatefulset.statefulset.Spec.Template.Spec.NodeSelector = map[string]string{
			"kubernetes.io/os": "windows",
		}
		testStatefulset.statefulset.Spec.Template.Spec.Containers[0].Image = "e2eteam/busybox:1.29"
		if useCMD {
			testStatefulset.statefulset.Spec.Template.Spec.Containers[0].Command = []string{"cmd"}
			testStatefulset.statefulset.Spec.Template.Spec.Containers[0].Args = []string{"/c", command}
		} else {
			testStatefulset.statefulset.Spec.Template.Spec.Containers[0].Command = []string{"powershell.exe"}
			testStatefulset.statefulset.Spec.Template.Spec.Containers[0].Args = []string{"-Command", command}
		}
	}

	return testStatefulset
}

func (t *TestStatefulset) Create() {
	var err error
	t.statefulset, err = t.client.AppsV1().StatefulSets(t.namespace.Name).Create(context.TODO(), t.statefulset, metav1.CreateOptions{})
	framework.ExpectNoError(err)
	err = waitForStatefulSetComplete(t.client, t.namespace, t.statefulset)
	framework.ExpectNoError(err)
	selector, err := metav1.LabelSelectorAsSelector(t.statefulset.Spec.Selector)
	framework.ExpectNoError(err)
	options := metav1.ListOptions{LabelSelector: selector.String()}
	statefulSetPods, err := t.client.CoreV1().Pods(t.namespace.Name).List(context.TODO(), options)
	framework.ExpectNoError(err)
	// always get first pod as there should only be one
	t.podName = statefulSetPods.Items[0].Name
}

func (t *TestStatefulset) WaitForPodReady() {
	selector, err := metav1.LabelSelectorAsSelector(t.statefulset.Spec.Selector)
	framework.ExpectNoError(err)
	options := metav1.ListOptions{LabelSelector: selector.String()}
	statefulSetPods, err := t.client.CoreV1().Pods(t.namespace.Name).List(context.TODO(), options)
	framework.ExpectNoError(err)
	// always get first pod as there should only be one
	pod := statefulSetPods.Items[0]
	t.podName = pod.Name
	err = e2epod.WaitForPodRunningInNamespace(t.client, &pod)
	framework.ExpectNoError(err)
}

func (t *TestStatefulset) Exec(command []string, expectedString string) {
	_, err := framework.LookForStringInPodExec(t.namespace.Name, t.podName, command, expectedString, execTimeout)
	framework.ExpectNoError(err)
}

func (t *TestStatefulset) DeletePodAndWait() {
	e2elog.Logf("Deleting pod %q in namespace %q", t.podName, t.namespace.Name)
	err := t.client.CoreV1().Pods(t.namespace.Name).Delete(context.TODO(), t.podName, metav1.DeleteOptions{})
	if err != nil {
		if !apierrs.IsNotFound(err) {
			framework.ExpectNoError(fmt.Errorf("pod %q Delete API error: %v", t.podName, err))
		}
		return
	}
	//sleep ensure waitForPodready will not be passed before old pod is deleted.
	time.Sleep(60 * time.Second)
}

func (t *TestStatefulset) Cleanup() {
	e2elog.Logf("deleting StatefulSet %q/%q", t.namespace.Name, t.statefulset.Name)
	body, err := t.Logs()
	if err != nil {
		e2elog.Logf("Error getting logs for pod %s: %v", t.podName, err)
	} else {
		e2elog.Logf("Pod %s has the following logs: %s", t.podName, body)
	}
	err = t.client.AppsV1().StatefulSets(t.namespace.Name).Delete(context.TODO(), t.statefulset.Name, metav1.DeleteOptions{})
	framework.ExpectNoError(err)
}

func (t *TestStatefulset) Logs() ([]byte, error) {
	return podLogs(t.client, t.podName, t.namespace.Name)
}
func waitForStatefulSetComplete(cs clientset.Interface, ns *v1.Namespace, ss *apps.StatefulSet) error {
	err := wait.PollImmediate(poll, pollTimeout, func() (bool, error) {
		var err error
		statefulSet, err := cs.AppsV1().StatefulSets(ns.Name).Get(context.TODO(), ss.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		klog.Infof("%d/%d replicas in the StatefulSet are ready", statefulSet.Status.ReadyReplicas, *statefulSet.Spec.Replicas)
		if statefulSet.Status.ReadyReplicas == *statefulSet.Spec.Replicas {
			return true, nil
		}
		return false, nil
	})

	return err
}

type TestPod struct {
	client    clientset.Interface
	pod       *v1.Pod
	namespace *v1.Namespace
}

func NewTestPod(c clientset.Interface, ns *v1.Namespace, command, schedulerName string, isWindows bool) *TestPod {
	testPod := &TestPod{
		client:    c,
		namespace: ns,
		pod: &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "azuredisk-volume-tester-",
			},
			Spec: v1.PodSpec{
				SchedulerName: schedulerName,
				NodeSelector:  map[string]string{"kubernetes.io/os": "linux"},
				Containers: []v1.Container{
					{
						Name:    "volume-tester",
						Image:   imageutils.GetE2EImage(imageutils.BusyBox),
						Command: []string{"/bin/sh"},
						Args:    []string{"-c", command},
					},
				},
				RestartPolicy: v1.RestartPolicyNever,
				Volumes:       make([]v1.Volume, 0),
			},
		},
	}
	if isWindows {
		testPod.pod.Spec.NodeSelector = map[string]string{
			"kubernetes.io/os": "windows",
		}
		testPod.pod.Spec.Containers[0].Image = "e2eteam/busybox:1.29"
		testPod.pod.Spec.Containers[0].Command = []string{"powershell.exe"}
		testPod.pod.Spec.Containers[0].Args = []string{"-Command", command}
	}

	return testPod
}

func (t *TestPod) Create() {
	var err error

	t.pod, err = t.client.CoreV1().Pods(t.namespace.Name).Create(context.TODO(), t.pod, metav1.CreateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestPod) WaitForSuccess() {
	err := e2epod.WaitForPodSuccessInNamespaceSlow(t.client, t.pod.Name, t.namespace.Name)
	framework.ExpectNoError(err)
}

func (t *TestPod) WaitForRunning() {
	err := e2epod.WaitForPodRunningInNamespace(t.client, t.pod)
	framework.ExpectNoError(err)
}

func (t *TestPod) WaitForRunningLong() {
	err := e2epod.WaitForPodRunningInNamespaceSlow(t.client, t.pod.Name, t.namespace.Name)
	framework.ExpectNoError(err)
}

func (t *TestPod) WaitForFailedMountError() {
	err := e2eevents.WaitTimeoutForEvent(
		t.client,
		t.namespace.Name,
		fields.Set{"reason": events.FailedMountVolume}.AsSelector().String(),
		"MountVolume.MountDevice failed for volume",
		pollLongTimeout)
	framework.ExpectNoError(err)
}

// Ideally this would be in "k8s.io/kubernetes/test/e2e/framework"
// Similar to framework.WaitForPodSuccessInNamespaceSlow
var podFailedCondition = func(pod *v1.Pod) (bool, error) {
	switch pod.Status.Phase {
	case v1.PodFailed:
		ginkgo.By("Saw pod failure")
		return true, nil
	case v1.PodSucceeded:
		return true, fmt.Errorf("pod %q successed with reason: %q, message: %q", pod.Name, pod.Status.Reason, pod.Status.Message)
	default:
		return false, nil
	}
}

func (t *TestPod) WaitForFailure() {
	err := e2epod.WaitForPodCondition(t.client, t.namespace.Name, t.pod.Name, failedConditionDescription, slowPodStartTimeout, podFailedCondition)
	framework.ExpectNoError(err)
}

func (t *TestPod) SetupVolume(pvc *v1.PersistentVolumeClaim, name, mountPath string, readOnly bool) {
	volumeMount := v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
	t.pod.Spec.Containers[0].VolumeMounts = append(t.pod.Spec.Containers[0].VolumeMounts, volumeMount)

	volume := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.Name,
			},
		},
	}
	t.pod.Spec.Volumes = append(t.pod.Spec.Volumes, volume)
}

func (t *TestPod) SetupInlineVolume(name, mountPath, diskURI string, readOnly bool) {
	volumeMount := v1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	}
	t.pod.Spec.Containers[0].VolumeMounts = append(t.pod.Spec.Containers[0].VolumeMounts, volumeMount)

	kind := v1.AzureDataDiskKind("Managed")
	diskName, _ := azuredisk.GetDiskName(diskURI)
	volume := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			AzureDisk: &v1.AzureDiskVolumeSource{
				DiskName:    diskName,
				DataDiskURI: diskURI,
				ReadOnly:    &readOnly,
				Kind:        &kind,
			},
		},
	}
	t.pod.Spec.Volumes = append(t.pod.Spec.Volumes, volume)
}

func (t *TestPod) SetupRawBlockVolume(pvc *v1.PersistentVolumeClaim, name, devicePath string) {
	volumeDevice := v1.VolumeDevice{
		Name:       name,
		DevicePath: devicePath,
	}
	t.pod.Spec.Containers[0].VolumeDevices = make([]v1.VolumeDevice, 0)
	t.pod.Spec.Containers[0].VolumeDevices = append(t.pod.Spec.Containers[0].VolumeDevices, volumeDevice)

	volume := v1.Volume{
		Name: name,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvc.Name,
			},
		},
	}
	t.pod.Spec.Volumes = append(t.pod.Spec.Volumes, volume)
}

func (t *TestPod) SetNodeSelector(nodeSelector map[string]string) {
	t.pod.Spec.NodeSelector = nodeSelector
}

func (t *TestPod) SetNodeUnschedulable(nodeName string, unschedulable bool) {
	var err error
	var node *v1.Node
	node, err = t.client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	framework.ExpectNoError(err)
	node.Spec.Unschedulable = unschedulable
	_, err = t.client.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
	framework.ExpectNoError(err)
}

func (t *TestPod) Cleanup() {
	cleanupPodOrFail(t.client, t.pod.Name, t.namespace.Name)
}

func (t *TestPod) GetZoneForVolume(index int) string {
	pvcSource := t.pod.Spec.Volumes[index].VolumeSource.PersistentVolumeClaim
	if pvcSource == nil {
		return ""
	}

	pvc, err := t.client.CoreV1().PersistentVolumeClaims(t.namespace.Name).Get(context.TODO(), pvcSource.ClaimName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	pv, err := t.client.CoreV1().PersistentVolumes().Get(context.TODO(), pvc.Spec.VolumeName, metav1.GetOptions{})
	framework.ExpectNoError(err)

	zone := ""
	for _, term := range pv.Spec.NodeAffinity.Required.NodeSelectorTerms {
		for _, ex := range term.MatchExpressions {
			if ex.Key == "topology.disk.csi.azure.com/zone" && ex.Operator == v1.NodeSelectorOpIn {
				zone = ex.Values[0]
			}
		}
	}

	return zone
}

func (t *TestPod) Logs() ([]byte, error) {
	return podLogs(t.client, t.pod.Name, t.namespace.Name)
}

func ListNodeNames(c clientset.Interface) []string {
	var nodeNames []string
	nodes, err := c.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	framework.ExpectNoError(err)
	for _, item := range nodes.Items {
		nodeNames = append(nodeNames, item.ObjectMeta.Name)
	}
	return nodeNames
}

func cleanupPodOrFail(client clientset.Interface, name, namespace string) {
	e2elog.Logf("deleting Pod %q/%q", namespace, name)
	body, err := podLogs(client, name, namespace)
	if err != nil {
		e2elog.Logf("Error getting logs for pod %s: %v", name, err)
	} else {
		e2elog.Logf("Pod %s has the following logs: %s", name, body)
	}
	e2epod.DeletePodOrFail(client, namespace, name)
}

func podLogs(client clientset.Interface, name, namespace string) ([]byte, error) {
	return client.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{}).Do(context.TODO()).Raw()
}

func getPodsForDeployment(client clientset.Interface, deployment *apps.Deployment) (*v1.PodList, error) {
	replicaSet, err := deploymentutil.GetNewReplicaSet(deployment, client.AppsV1())
	if err != nil {
		return nil, fmt.Errorf("Failed to get new replica set for deployment %q: %v", deployment.Name, err)
	}
	if replicaSet == nil {
		return nil, fmt.Errorf("expected a new replica set for deployment %q, found none", deployment.Name)
	}
	podListFunc := func(namespace string, options metav1.ListOptions) (*v1.PodList, error) {
		return client.CoreV1().Pods(namespace).List(context.TODO(), options)
	}
	rsList := []*apps.ReplicaSet{replicaSet}
	podList, err := deploymentutil.ListPods(deployment, rsList, podListFunc)
	if err != nil {
		return nil, fmt.Errorf("Failed to list Pods of Deployment %q: %v", deployment.Name, err)
	}
	return podList, nil
}

// waitForPersistentVolumeClaimDeleted waits for a PersistentVolumeClaim to be removed from the system until timeout occurs, whichever comes first.
func waitForPersistentVolumeClaimDeleted(c clientset.Interface, ns string, pvcName string, Poll, timeout time.Duration) error {
	framework.Logf("Waiting up to %v for PersistentVolumeClaim %s to be removed", timeout, pvcName)
	for start := time.Now(); time.Since(start) < timeout; time.Sleep(Poll) {
		_, err := c.CoreV1().PersistentVolumeClaims(ns).Get(context.TODO(), pvcName, metav1.GetOptions{})
		if err != nil {
			if apierrs.IsNotFound(err) {
				framework.Logf("Claim %q in namespace %q doesn't exist in the system", pvcName, ns)
				return nil
			}
			framework.Logf("Failed to get claim %q in namespace %q, retrying in %v. Error: %v", pvcName, ns, Poll, err)
		}
	}
	return fmt.Errorf("PersistentVolumeClaim %s is not removed from the system within %v", pvcName, timeout)
}

type TestAzVolumeAttachment struct {
	azclient             v1alpha1ClientSet.DiskV1alpha1Interface
	namespace            string
	underlyingVolume     string
	primaryNodeName      string
	maxMountReplicaCount int
}

func NewTestAzDriverNode(azDriverNode v1alpha1ClientSet.AzDriverNodeInterface, nodeName string) *v1alpha1.AzDriverNode {
	// Delete the leftover azDriverNode from previous runs
	if _, err := azDriverNode.Get(context.Background(), nodeName, metav1.GetOptions{}); err == nil {
		err := azDriverNode.Delete(context.Background(), nodeName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	}

	newAzDriverNode, err := azDriverNode.Create(context.Background(), &v1alpha1.AzDriverNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
		},
		Spec: v1alpha1.AzDriverNodeSpec{
			NodeName: nodeName,
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	return newAzDriverNode
}

func DeleteTestAzDriverNode(azDriverNode v1alpha1ClientSet.AzDriverNodeInterface, nodeName string) {
	_ = azDriverNode.Delete(context.Background(), nodeName, metav1.DeleteOptions{})
}

func NewTestAzVolumeAttachment(azVolumeAttachment v1alpha1ClientSet.AzVolumeAttachmentInterface, volumeAttachmentName, nodeName, volumeName string) *v1alpha1.AzVolumeAttachment {
	// Delete leftover azVolumeAttachments from previous runs
	if _, err := azVolumeAttachment.Get(context.Background(), volumeAttachmentName, metav1.GetOptions{}); err == nil {
		err := azVolumeAttachment.Delete(context.Background(), volumeAttachmentName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	}

	newAzVolumeAttachment, err := azVolumeAttachment.Create(context.Background(), &v1alpha1.AzVolumeAttachment{
		ObjectMeta: metav1.ObjectMeta{
			Name: volumeAttachmentName,
		},
		Spec: v1alpha1.AzVolumeAttachmentSpec{
			UnderlyingVolume: volumeName,
			NodeName:         nodeName,
			RequestedRole:    v1alpha1.PrimaryRole,
		},
		Status: &v1alpha1.AzVolumeAttachmentStatus{
			Role: v1alpha1.PrimaryRole,
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	return newAzVolumeAttachment
}

func DeleteTestAzVolumeAttachment(azVolumeAttachment v1alpha1ClientSet.AzVolumeAttachmentInterface, volumeAttachmentName string) {
	_ = azVolumeAttachment.Delete(context.Background(), volumeAttachmentName, metav1.DeleteOptions{})
}

func NewTestAzVolume(azVolume v1alpha1ClientSet.AzVolumeInterface, underlyingVolumeName string, maxMountReplicaCount int) *v1alpha1.AzVolume {
	// Delete leftover azVolumes from previous runs
	if _, err := azVolume.Get(context.Background(), underlyingVolumeName, metav1.GetOptions{}); err == nil {
		err := azVolume.Delete(context.Background(), underlyingVolumeName, metav1.DeleteOptions{})
		framework.ExpectNoError(err)
	}
	newAzVolume, err := azVolume.Create(context.Background(), &v1alpha1.AzVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name: underlyingVolumeName,
		},
		Spec: v1alpha1.AzVolumeSpec{
			UnderlyingVolume:     underlyingVolumeName,
			MaxMountReplicaCount: maxMountReplicaCount,
		},
	}, metav1.CreateOptions{})
	framework.ExpectNoError(err)

	return newAzVolume
}

func SetupTestAzVolumeAttachment(azclient v1alpha1ClientSet.DiskV1alpha1Interface, namespace, underlyingVolume, primaryNodeName string, maxMountReplicaCount int) *TestAzVolumeAttachment {
	return &TestAzVolumeAttachment{
		azclient:             azclient,
		namespace:            namespace,
		underlyingVolume:     underlyingVolume,
		primaryNodeName:      primaryNodeName,
		maxMountReplicaCount: maxMountReplicaCount,
	}
}

func (t *TestAzVolumeAttachment) Create() *v1alpha1.AzVolumeAttachment {
	// create test az volume
	azVol := t.azclient.AzVolumes(t.namespace)
	_ = NewTestAzVolume(azVol, t.underlyingVolume, t.maxMountReplicaCount)

	// create test az volume attachment
	azAtt := t.azclient.AzVolumeAttachments(t.namespace)
	attName := GetAzVolumeAttachmentName(t.underlyingVolume, t.primaryNodeName)
	att := NewTestAzVolumeAttachment(azAtt, attName, t.primaryNodeName, t.underlyingVolume)

	return att
}

func (t *TestAzVolumeAttachment) Cleanup() {
	klog.Info("cleaning up")
	err := t.azclient.AzVolumes(t.namespace).Delete(context.Background(), t.underlyingVolume, metav1.DeleteOptions{})
	if !apierrs.IsNotFound(err) {
		framework.ExpectNoError(err)
	}

	// Delete All AzVolumeAttachments for t.underlyingVolume
	err = t.azclient.AzVolumeAttachments(t.namespace).Delete(context.Background(), GetAzVolumeAttachmentName(t.underlyingVolume, t.primaryNodeName), metav1.DeleteOptions{})
	if !apierrs.IsNotFound(err) {
		framework.ExpectNoError(err)
	}

	nodes, err := t.azclient.AzDriverNodes(t.namespace).List(context.Background(), metav1.ListOptions{})
	if !apierrs.IsNotFound(err) {
		framework.ExpectNoError(err)
	}
	for _, node := range nodes.Items {
		err = t.azclient.AzVolumeAttachments(t.namespace).Delete(context.Background(), GetAzVolumeAttachmentName(t.underlyingVolume, node.Name), metav1.DeleteOptions{})
		if !apierrs.IsNotFound(err) {
			framework.ExpectNoError(err)
		}
	}
}

func GetAzVolumeAttachmentName(underlyingVolume, nodeName string) string {
	return fmt.Sprintf("%s-%s-attachment", underlyingVolume, nodeName)
}

// Wait for the azVolumeAttachment object update
func (t *TestAzVolumeAttachment) WaitForAttach(timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		att, err := t.azclient.AzVolumeAttachments(t.namespace).Get(context.TODO(), GetAzVolumeAttachmentName(t.underlyingVolume, t.primaryNodeName), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if att.Status != nil {
			klog.Infof("volume (%s) attached to node (%s)", att.Spec.UnderlyingVolume, att.Spec.NodeName)
			return true, nil
		}
		return false, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

// Wait for the azVolumeAttachment object update
func (t *TestAzVolumeAttachment) WaitForFinalizer(timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		att, err := t.azclient.AzVolumeAttachments(t.namespace).Get(context.TODO(), GetAzVolumeAttachmentName(t.underlyingVolume, t.primaryNodeName), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if att.ObjectMeta.Finalizers == nil {
			return false, nil
		}
		for _, finalizer := range att.ObjectMeta.Finalizers {
			if finalizer == controller.AzVolumeAttachmentFinalizer {
				klog.Infof("finalizer (%s) found on AzVolumeAttachment object (%s)", controller.AzVolumeAttachmentFinalizer, att.Name)
				return true, nil
			}
		}
		return false, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

// Wait for the azVolumeAttachment object update
func (t *TestAzVolumeAttachment) WaitForLabels(timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		att, err := t.azclient.AzVolumeAttachments(t.namespace).Get(context.TODO(), GetAzVolumeAttachmentName(t.underlyingVolume, t.primaryNodeName), metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if att.Labels == nil {
			return false, nil
		}
		if _, ok := att.Labels["node-name"]; !ok {
			return false, nil
		}
		if _, ok := att.Labels["volume-name"]; !ok {
			return false, nil
		}
		return true, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

// Wait for the azVolumeAttachment object update
func (t *TestAzVolumeAttachment) WaitForDelete(nodeName string, timeout time.Duration) error {
	attName := GetAzVolumeAttachmentName(t.underlyingVolume, nodeName)
	conditionFunc := func() (bool, error) {
		_, err := t.azclient.AzVolumeAttachments(t.namespace).Get(context.TODO(), attName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			klog.Infof("azVolumeAttachment %s deleted.", attName)
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

// Wait for the azVolumeAttachment object update
func (t *TestAzVolumeAttachment) WaitForPrimary(timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		attachments, err := t.azclient.AzVolumeAttachments(t.namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		for _, attachment := range attachments.Items {
			if attachment.Status == nil {
				continue
			}
			if attachment.Spec.UnderlyingVolume == t.underlyingVolume && attachment.Spec.RequestedRole == v1alpha1.PrimaryRole && attachment.Status.Role == v1alpha1.PrimaryRole {
				return true, nil
			}
		}
		return false, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

// Wait for the azVolumeAttachment object update
func (t *TestAzVolumeAttachment) WaitForReplicas(numReplica int, timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		attachments, err := t.azclient.AzVolumeAttachments(t.namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return false, err
		}
		counter := 0
		for _, attachment := range attachments.Items {
			if attachment.Status == nil {
				continue
			}
			if attachment.Spec.UnderlyingVolume == t.underlyingVolume && attachment.Status.Role == v1alpha1.ReplicaRole {
				counter++
			}
		}
		klog.Infof("%d replica found for volume %s", counter, t.underlyingVolume)
		return counter == numReplica, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

type TestAzVolume struct {
	azclient             v1alpha1ClientSet.DiskV1alpha1Interface
	namespace            string
	underlyingVolume     string
	maxMountReplicaCount int
}

func SetupTestAzVolume(azclient v1alpha1ClientSet.DiskV1alpha1Interface, namespace string, underlyingVolume string, maxMountReplicaCount int) *TestAzVolume {
	return &TestAzVolume{
		azclient:             azclient,
		namespace:            namespace,
		underlyingVolume:     underlyingVolume,
		maxMountReplicaCount: maxMountReplicaCount,
	}
}

func (t *TestAzVolume) Create() *v1alpha1.AzVolume {
	// create test az volume
	azVolClient := t.azclient.AzVolumes(t.namespace)
	azVolume := NewTestAzVolume(azVolClient, t.underlyingVolume, t.maxMountReplicaCount)

	return azVolume
}

//Cleanup after TestAzVolume was created
func (t *TestAzVolume) Cleanup() {
	klog.Info("cleaning up TestAzVolume")
	err := t.azclient.AzVolumes(t.namespace).Delete(context.Background(), t.underlyingVolume, metav1.DeleteOptions{})
	if !apierrs.IsNotFound(err) {
		framework.ExpectNoError(err)
	}
	time.Sleep(time.Duration(1) * time.Minute)

}

// Wait for the azVolume object update
func (t *TestAzVolume) WaitForFinalizer(timeout time.Duration) error {
	conditionFunc := func() (bool, error) {
		azVolume, err := t.azclient.AzVolumes(t.namespace).Get(context.TODO(), t.underlyingVolume, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if azVolume.ObjectMeta.Finalizers == nil {
			return false, nil
		}
		for _, finalizer := range azVolume.ObjectMeta.Finalizers {
			if finalizer == controller.AzVolumeFinalizer {
				klog.Infof("finalizer (%s) found on AzVolume object (%s)", controller.AzVolumeFinalizer, azVolume.Name)
				return true, nil
			}
		}
		return false, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}

// Wait for the azVolume object update
func (t *TestAzVolume) WaitForDelete(timeout time.Duration) error {
	klog.Infof("Waiting for delete azVolume object")
	conditionFunc := func() (bool, error) {
		_, err := t.azclient.AzVolumes(t.namespace).Get(context.TODO(), t.underlyingVolume, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			klog.Infof("azVolume %s deleted.", t.underlyingVolume)
			return true, nil
		} else if err != nil {
			return false, err
		}
		return false, nil
	}
	return wait.PollImmediate(time.Duration(15)*time.Second, timeout, conditionFunc)
}
