/*
Copyright 2021.

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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-version"
	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	dynclient "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/client"
	vdocontext "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/context"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/drivers/cpi"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/drivers/csi"
	. "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/models"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/session"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/discovery"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	CLOUD_PROVIDER_INIT_TAINT_KEY = "node.cloudprovider.kubernetes.io/uninitialized"
	TAINT_NOSCHEDULE_KEY          = "NoSchedule"
	VDO_NODE_LABEL_KEY            = "vdo.vmware.com/vdoconfig"
	VDO_NAMESPACE                 = "vmware-system-vdo"
	CPI_DEPLOYMENT_NAME           = "vsphere-cloud-controller-manager"
	DEPLOYMENT_NS                 = "kube-system"
	CPI_DAEMON_POD_KEY            = "k8s-app"
	SECRET_NAME                   = "cpi-global-secret"
	CONFIGMAP_NAME                = "cloud-config"
	CSI_DEPLOYMENT_NAME           = "vsphere-csi-node"
	CSI_DAEMON_POD_KEY            = "app"
	CSI_SECRET_NAME               = "vsphere-config-secret"
	CSI_SECRET_CONFIG_FILE        = "/tmp/csi-vsphere.conf"
	COMPAT_MATRIX_CONFIG_URL      = "MATRIX_CONFIG_URL"
	COMPAT_MATRIX_CONFIG_CONTENT  = "MATRIX_CONFIG_CONTENT"
)

// VDOConfigReconciler reconciles a VDOConfig object
type VDOConfigReconciler struct {
	client.Client
	Logger                    logr.Logger
	Scheme                    *runtime.Scheme
	ClientConfig              *restclient.Config
	CsiDeploymentYamls        []string
	CpiDeploymentYamls        []string
	CurrentCSIDeployedVersion string
	CurrentCPIDeployedVersion string
}

var (
	SessionFn = session.GetOrCreate
	GetVMFn   = session.GetVMByIP
)

const (
	CM_NAME        = "compat-matrix-config"
	CM_URL_KEY     = "versionConfigURL"
	CM_CONTENT_KEY = "versionConfigContent"
)

// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vdoconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vdoconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vdoconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vspherecloudconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vspherecloudconfigs/status,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;get;list;watch;update;delete;
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;update;patch;watch;delete;
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=*
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=*
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=*
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=*
// +kubebuilder:rbac:groups="storage.k8s.io",resources=csinodes,verbs=create;get;list;watch
// +kubebuilder:rbac:groups="storage.k8s.io",resources=csidrivers,verbs=create;update;patch;get;list;watch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=create;update;patch;get;list;watch;delete;
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete;
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;create;update;patch;delete;
// +kubebuilder:rbac:groups="apps",resources=daemonsets,verbs=get;list;create;update;patch;delete;
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;create;update;patch;delete;
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;get;list;watch;update;patch;delete;

func (r *VDOConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger.Info("Inside VDOConfig reconciler", "name", req.NamespacedName)

	vdoctx := vdocontext.VDOContext{
		Context: ctx,
		Logger:  r.Logger,
	}

	// Check if Reconcile is for ConfigMap Changes
	err := r.updateMatrixInfo(vdoctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	vdoConfig := &vdov1alpha1.VDOConfig{}

	if req.NamespacedName.Name == CM_NAME {
		vdoConfigListItems := &vdov1alpha1.VDOConfigList{}
		err = r.List(ctx, vdoConfigListItems)
		if err != nil {
			vdoctx.Logger.Error(err, "Error occurred when fetching vdoConfig resource", "name", vdoConfigListItems)
			return ctrl.Result{}, err
		}
		vdoConfigList := vdoConfigListItems.Items
		if len(vdoConfigList) <= 0 || len(vdoConfigList) > 1 {
			vdoctx.Logger.Error(err, "Skipping Reconcile for vdoConfig resource as no/multiple resources found", "name", vdoConfigListItems.ListMeta)
			return ctrl.Result{}, err
		}
		vdoConfig = &vdoConfigList[0]
	} else {
		err = r.Get(ctx, req.NamespacedName, vdoConfig)
		if err != nil {
			vdoctx.Logger.Error(err, "Error occurred when fetching vdoConfig resource", "name", req.NamespacedName)
			return ctrl.Result{}, err
		}
	}

	matrixConfigUrl := os.Getenv(COMPAT_MATRIX_CONFIG_URL)
	matrixConfigContent := os.Getenv(COMPAT_MATRIX_CONFIG_CONTENT)

	clientset, err := kubernetes.NewForConfig(r.ClientConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	matrixConfig, err := r.getMatrixConfig(matrixConfigUrl, matrixConfigContent)
	if err != nil {
		vdoctx.Logger.Error(err, "Unable to fetch deployment yamls")
		return ctrl.Result{}, err
	}

	err = r.CheckCompatAndRetrieveSpec(vdoctx, req, vdoConfig, matrixConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	result, err := r.reconcileCPIConfiguration(vdoctx, req, vdoConfig, clientset)
	if err != nil {
		return result, err
	}

	result, err = r.reconcileCSIConfiguration(vdoctx, req, vdoConfig, clientset)
	if err != nil {
		return result, err
	}

	return result, nil

}

func (r *VDOConfigReconciler) updateMatrixInfo(ctx vdocontext.VDOContext, req ctrl.Request) error {

	if req.NamespacedName.Namespace == VDO_NAMESPACE && req.NamespacedName.Name == CM_NAME {
		configMap := &v1.ConfigMap{}
		err := r.Get(ctx, req.NamespacedName, configMap)
		if err != nil {
			//Reset the env variables in case ConfigMap is not available
			r.setEnvVariables(COMPAT_MATRIX_CONFIG_URL, "")
			r.setEnvVariables(COMPAT_MATRIX_CONFIG_CONTENT, "")
			ctx.Logger.Error(err, "Error occurred when fetching ConfigMap resource", "name", req.NamespacedName)
			return err
		}

		//Get the modified Value for URL/Content and Set it as env
		if updatedURLValue, ok := configMap.Data[CM_URL_KEY]; ok {
			r.setEnvVariables(COMPAT_MATRIX_CONFIG_URL, updatedURLValue)
		}
		if updatedContentValue, ok := configMap.Data[CM_CONTENT_KEY]; ok {
			r.setEnvVariables(COMPAT_MATRIX_CONFIG_CONTENT, updatedContentValue)
		}
	}
	return nil
}

// setEnvVariables sets the env variables
func (r *VDOConfigReconciler) setEnvVariables(key string, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		r.Logger.Error(err, fmt.Sprintf("Failed to set the env variable :  %s", key))
	}
}

func (r *VDOConfigReconciler) fetchVSphereCloudConfig(ctx vdocontext.VDOContext, vSphereCloudConfigName string, vdoConfigNamespace string) (*vdov1alpha1.VsphereCloudConfig, error) {
	vsphereCloudConfigKey := types.NamespacedName{
		Namespace: vdoConfigNamespace,
		Name:      vSphereCloudConfigName,
	}

	vsphereCloudConfig := &vdov1alpha1.VsphereCloudConfig{}

	err := r.Get(ctx, vsphereCloudConfigKey, vsphereCloudConfig)
	if err != nil {
		r.Logger.Error(err, fmt.Sprintf("Error occurred when fetching vSphereCloudConfig resource %s", vsphereCloudConfigKey))
		return nil, errors.Wrapf(err, "could not fetch vSphereCLoudConfig resource %s", vsphereCloudConfigKey)
	}

	return vsphereCloudConfig, err
}
func (r *VDOConfigReconciler) verifyVsphereCloudConfig(vsphereCloudConfig *vdov1alpha1.VsphereCloudConfig) (string, error) {
	CloudConfigStatus := vsphereCloudConfig.Status.Config

	if CloudConfigStatus == "failed" {
		statusMsg := "Status of VSphereCloudConfig resource is in failed state"
		return statusMsg, errors.New("Failed to verify vSphere connection information. Status of VsphereCloudConfig resource is in failed state")
	}

	if CloudConfigStatus != "verified" {
		statusMsg := "Status of VsphereCloudConfig resource is unknown"

		return statusMsg, errors.New("Failed to verify vSphere connection information. Status of VsphereCloudConfig resource is in unknown state")
	}
	return "", nil
}

func (r *VDOConfigReconciler) fetchVsphereCloudConfigItems(vdoctx vdocontext.VDOContext, req ctrl.Request, vdoConfig *vdov1alpha1.VDOConfig, vsphereCloudConfigsList []string) ([]vdov1alpha1.VsphereCloudConfig, error) {

	var vsphereCloudConfigItems []vdov1alpha1.VsphereCloudConfig
	for i := range vsphereCloudConfigsList {
		vsphereCloudConfig, err := r.fetchVSphereCloudConfig(vdoctx, vdoConfig.Spec.CloudProvider.VsphereCloudConfigs[i], req.Namespace)
		if err != nil {
			statusMsg := "unable to fetch the vsphereCloudConfig resource"
			r.updateCPIStatusForError(vdoctx, err, vdoConfig, statusMsg)
			return vsphereCloudConfigItems, err
		}
		vsphereCloudConfigItems = append(vsphereCloudConfigItems, *vsphereCloudConfig)
	}
	return vsphereCloudConfigItems, nil
}

func (r *VDOConfigReconciler) Fetchk8sVersions(vdoctx vdocontext.VDOContext) (string, error) {
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(r.ClientConfig)
	k8sServerVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		vdoctx.Logger.Error(err, "Error occurred when k8s version")
		return "", err
	}
	k8sVersion := k8sServerVersion.Major + "." + k8sServerVersion.Minor
	return k8sVersion, nil
}

func (r *VDOConfigReconciler) FetchVsphereVersions(vdoctx vdocontext.VDOContext, req ctrl.Request, vdoConfig *vdov1alpha1.VDOConfig) (versions []string, err error) {
	var vsphereCloudConfigsList []string
	vsphereCloudConfigsList = vdoConfig.Spec.CloudProvider.VsphereCloudConfigs

	if len(vsphereCloudConfigsList) <= 0 {
		vsphereCloudConfigsList = []string{vdoConfig.Spec.StorageProvider.VsphereCloudConfig}
	}
	var vsphereCloudConfigItems []vdov1alpha1.VsphereCloudConfig
	for _, vsphereCloudConfig := range vsphereCloudConfigsList {
		vsphereCloudConfigItem, err := r.fetchVSphereCloudConfig(vdoctx, vsphereCloudConfig, req.Namespace)
		if err != nil {
			return nil, err
		}
		vsphereCloudConfigItems = append(vsphereCloudConfigItems, *vsphereCloudConfigItem)
	}

	var vsphereVersions []string

	for _, vsphereCloudConfig := range vsphereCloudConfigItems {
		sess, err := r.getVcSession(vdoctx, &vsphereCloudConfig)
		if err != nil {
			return nil, err
		}
		vsphereVersions = append(vsphereVersions, sess.VsphereVersion)
	}

	return vsphereVersions, nil
}

func (r *VDOConfigReconciler) getVcSession(vdoctx vdocontext.VDOContext, config *vdov1alpha1.VsphereCloudConfig) (*session.Session, error) {
	vcUser, vcUserPwd, err := r.fetchVcCredentials(vdoctx, *config)

	if err != nil {
		return nil, errors.Wrapf(err, "Error fetching vcenter credentials ")
	}

	vcIp := config.Spec.VcIP
	sess, err := SessionFn(vdoctx, vcIp, config.Spec.DataCenters, vcUser, vcUserPwd, config.Spec.Thumbprint)
	if err != nil {
		config.Status.Config = vdov1alpha1.VsphereConfigFailed
		config.Status.Message = fmt.Sprintf("Error establishing session with vcenter %s for user %s", vcIp, vcUser)
		return nil, errors.Wrapf(err, "Error establishing session with vcenter %s for user %s", vcIp, vcUser)
	}

	return sess, nil
}

func (r *VDOConfigReconciler) reconcileCPIConfiguration(vdoctx vdocontext.VDOContext, req ctrl.Request, vdoConfig *vdov1alpha1.VDOConfig, clientset *kubernetes.Clientset) (ctrl.Result, error) {

	vsphereCloudConfigsList := vdoConfig.Spec.CloudProvider.VsphereCloudConfigs
	if len(vsphereCloudConfigsList) <= 0 {
		vdoctx.Logger.V(4).Info("CPI is not configured for VDO")
		return ctrl.Result{}, nil
	}
	vsphereCloudConfigItems, err := r.fetchVsphereCloudConfigItems(vdoctx, req, vdoConfig, vsphereCloudConfigsList)
	if err != nil {
		return ctrl.Result{}, err
	}

	for _, vsphereCloudConfig := range vsphereCloudConfigItems {
		statusMsg, err := r.verifyVsphereCloudConfig(&vsphereCloudConfig)
		if err != nil {
			r.updateCPIStatusForError(vdoctx, err, vdoConfig, statusMsg)
			return ctrl.Result{}, err
		}
	}

	cpiSecretKey := types.NamespacedName{
		Namespace: VC_CREDS_SECRET_NS,
		Name:      SECRET_NAME,
	}

	vdoctx.Logger.V(4).Info("reconciling secret for CPI")
	vdoConfig, err = r.reconcileCPISecret(vdoctx, vdoConfig, &vsphereCloudConfigItems, cpiSecretKey)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of secret for CPI configuration")
		return ctrl.Result{}, err
	}

	vdoctx.Logger.V(4).Info("reconciling configmap for CPI")
	vdoConfig, err = r.reconcileConfigMap(vdoctx, vdoConfig, &vsphereCloudConfigItems, cpiSecretKey)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of configmap for CPI configuration")
		return ctrl.Result{}, err
	}

	vdoctx.Logger.V(4).Info("reconciling deployment for CPI")
	updateStatus, err := r.reconcileCPIDeployment(vdoctx)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of deployment of CPI spec files")
		return ctrl.Result{}, err
	}

	if updateStatus {
		err = r.updateCPIPhase(vdoctx, vdoConfig, vdov1alpha1.Deploying, "")
		if err != nil {
			vdoctx.Logger.Error(err, "Error occurred when reconciling deployment for CPI")
			return ctrl.Result{}, err
		}
	}

	vdoctx.Logger.V(4).Info("reconciling deployment status for CPI")
	err = r.reconcileCPIDeploymentStatus(vdoctx, clientset)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of deployment status for CPI")
		return ctrl.Result{}, err
	}

	if vdoConfig.Status.CPIStatus.Phase != vdov1alpha1.Deployed {
		err = r.updateCPIPhase(vdoctx, vdoConfig, vdov1alpha1.Deployed, "")
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	vdoctx.Logger.Info("reconciling node providerID")
	updReq, err := r.reconcileNodeProviderID(vdoctx, vdoConfig, clientset, &vsphereCloudConfigItems)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, err.Error())
		return ctrl.Result{}, err
	}

	if updReq {
		err = r.updateVdoConfigWithNodeStatus(vdoctx, vdoConfig, vdoConfig.Status.CPIStatus.Phase, vdoConfig.Status.CPIStatus.NodeStatus)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	vdoctx.Logger.V(4).Info("reconciling node label for CPI")
	err = r.reconcileNodeLabel(vdoctx, req, clientset, vdoConfig)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, err.Error())
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *VDOConfigReconciler) reconcileCSIConfiguration(vdoctx vdocontext.VDOContext, req ctrl.Request, vdoConfig *vdov1alpha1.VDOConfig, clientset *kubernetes.Clientset) (ctrl.Result, error) {

	vsphereCloudConfig, err := r.fetchVSphereCloudConfig(vdoctx, vdoConfig.Spec.StorageProvider.VsphereCloudConfig, req.Namespace)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, "Unable to fetch vSphereCLoudConfig resource")
		return ctrl.Result{}, err
	}

	statusMsg, err := r.verifyVsphereCloudConfig(vsphereCloudConfig)
	if err != nil {
		r.updateCPIStatusForError(vdoctx, err, vdoConfig, statusMsg)
		return ctrl.Result{}, err
	}

	vdoctx.Logger.V(4).Info("reconciling secret for CSI")
	vdoConfig, err = r.reconcileCSISecret(vdoctx, vdoConfig, vsphereCloudConfig)
	if err != nil {
		r.updateCSIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of secret for CSI configuration")
		return ctrl.Result{}, err
	}

	err = r.updateCSIPhase(vdoctx, vdoConfig, vdov1alpha1.Configuring, "")
	if err != nil {
		r.updateCSIStatusForError(vdoctx, err, vdoConfig, "Error in updating CSI phase")
		return ctrl.Result{}, err
	}

	vdoctx.Logger.V(4).Info("reconciling deployment for CSI")
	updateStatus, err := r.reconcileCSIDeployment(vdoctx)
	if err != nil {
		r.updateCSIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of deployment of CSI spec files")
		return ctrl.Result{}, err
	}

	if updateStatus {
		err = r.updateCSIPhase(vdoctx, vdoConfig, vdov1alpha1.Deploying, "")
		if err != nil {
			vdoctx.Logger.Error(err, "Error occurred when reconciling deployment for CSI")
			return ctrl.Result{}, err
		}
	}

	vdoctx.Logger.V(4).Info("reconciling deployment status for CSI")
	err = r.reconcileCSIDeploymentStatus(vdoctx, clientset)
	if err != nil {
		r.updateCSIStatusForError(vdoctx, err, vdoConfig, "Error in reconcile of deployment status for CSI")
		return ctrl.Result{}, err
	}

	if vdoConfig.Status.CSIStatus.Phase == vdov1alpha1.Deploying {
		err = r.updateCSIPhase(vdoctx, vdoConfig, vdov1alpha1.Deployed, "")
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *VDOConfigReconciler) fetchVcCredentials(ctx vdocontext.VDOContext, vsphereCloudConfig vdov1alpha1.VsphereCloudConfig) (string, string, error) {
	if len(vsphereCloudConfig.Spec.Credentials) <= 0 {
		return "", "", errors.New("error fetching credentials from vsphereCloudConfig")
	}

	vcSecret := &v1.Secret{}
	key := types.NamespacedName{
		Namespace: VC_CREDS_SECRET_NS,
		Name:      vsphereCloudConfig.Spec.Credentials,
	}

	err := r.Get(ctx, key, vcSecret)
	if err != nil {
		ctx.Logger.Error(err, "could not fetch vc credentials secret ", "name", vsphereCloudConfig.Spec.Credentials)
		return "", "", err
	}

	vcUser := string(vcSecret.Data["username"])
	vcUserPwd := string(vcSecret.Data["password"])
	return vcUser, vcUserPwd, nil
}

func (r *VDOConfigReconciler) updateCPIStatusForError(vdoctx vdocontext.VDOContext, err error, config *vdov1alpha1.VDOConfig, msg string) {
	vdoctx.Logger.Error(err, msg, "name", config.Name)
	updErr := r.updateCPIPhase(vdoctx, config, vdov1alpha1.Failed, msg)
	if updErr != nil {
		vdoctx.Logger.Error(updErr, "Error occurred when updating vdoconfig for error state")
	}
}
func (r *VDOConfigReconciler) updateCSIStatusForError(vdoctx vdocontext.VDOContext, err error, config *vdov1alpha1.VDOConfig, msg string) {
	vdoctx.Logger.Error(err, msg, "name", config.Name)
	updErr := r.updateCSIPhase(vdoctx, config, vdov1alpha1.Failed, msg)
	if updErr != nil {
		vdoctx.Logger.Error(updErr, "Error occurred when updating vdoconfig for error state")
	}
}

func (r *VDOConfigReconciler) fetchDaemonSetPodStatus(ctx vdocontext.VDOContext, clientset kubernetes.Interface, deploymentName string, deploymentNs string, daemonPodkey string) error {

	var podsNotReady bool

	daemon := &appsv1.DaemonSet{}
	daemonKey := types.NamespacedName{Name: deploymentName, Namespace: deploymentNs}

	err := r.Get(ctx, daemonKey, daemon)
	if err != nil {
		ctx.Logger.Error(err, "unable to find daemonset", "name", deploymentName)
		return err
	}

	unavailableCount := daemon.Status.NumberUnavailable

	pods, err := clientset.CoreV1().Pods(deploymentNs).List(ctx, metav1.ListOptions{LabelSelector: daemonPodkey})
	if err != nil {
		ctx.Logger.Error(err, "unable to find pods running in daemonset",
			"name", deploymentName)
		return err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != v1.PodRunning {
			podsNotReady = true
			ctx.Logger.V(4).Info("pod not in running state", "podname", pod.Name,
				"namespace", pod.Namespace)
			break
		}
	}

	if podsNotReady || unavailableCount > 0 {
		err = errors.Errorf("Some Pods are not Ready or Unavailable")
		ctx.Logger.Error(err, "Not all pods in daemonset are in running state", "daemonsetName", deploymentName)
		return err
	}

	return nil
}

func (r *VDOConfigReconciler) verifyCSINodeStatus(ctx vdocontext.VDOContext, clientset kubernetes.Interface) error {
	ctx.Logger.V(4).Info("will attempt to reconcile status of CSI Nodes")

	csinodes, err := clientset.StorageV1().CSINodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to fetch list of CSI nodes")
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to fetch list of nodes")
	}

	for _, csinode := range csinodes.Items {
		ctx.Logger.V(4).Info("CSI nodes", "name", csinode.Name)
	}

	if len(nodes.Items) == len(csinodes.Items) {
		return nil
	}

	csinodeMap := make(map[string]string)
	for _, csinode := range csinodes.Items {
		csinodeMap[csinode.Name] = ""
	}
	for _, node := range nodes.Items {
		if _, ok := csinodeMap[node.Name]; !ok {
			err = errors.Errorf("not listed as csinode %s", node.Name)
			ctx.Logger.V(4).Error(err, "csinode resource does not exist for node", "name", node.Name)
			return err
		}
	}

	return nil

}

func (r *VDOConfigReconciler) verifyCSIDriverRegisteration(ctx vdocontext.VDOContext, clientset kubernetes.Interface) error {
	ctx.Logger.V(4).Info("will verify the CSI Driver Registration")

	csidrivers, err := clientset.StorageV1().CSIDrivers().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to fetch CSI Drivers")
	}

	if len(csidrivers.Items) <= 0 {
		return errors.New("No CSI Drivers registered")
	}

	for _, csidriver := range csidrivers.Items {
		ctx.Logger.V(4).Info("CSI Drivers", "name", csidriver.Name)
	}

	return nil

}

func (r *VDOConfigReconciler) reconcileCPIDeploymentStatus(ctx vdocontext.VDOContext, clientset kubernetes.Interface) error {
	ctx.Logger.V(4).Info("will attempt to reconcile deployment status of CPI")
	err := r.fetchDaemonSetPodStatus(ctx, clientset, CPI_DEPLOYMENT_NAME, DEPLOYMENT_NS, CPI_DAEMON_POD_KEY)
	if err != nil {
		return errors.Wrapf(err, "unable to get CPI DaemonSet Pod Status")
	}

	return nil
}

func (r *VDOConfigReconciler) reconcileCSIDeploymentStatus(ctx vdocontext.VDOContext, clientset kubernetes.Interface) error {
	ctx.Logger.V(4).Info("will attempt to reconcile deployment status of CSI")

	err := r.fetchDaemonSetPodStatus(ctx, clientset, CSI_DEPLOYMENT_NAME, DEPLOYMENT_NS, CSI_DAEMON_POD_KEY)
	if err != nil {
		return errors.Wrapf(err, "unable to get CSI DaemonSet Pod Status")
	}

	err = r.verifyCSINodeStatus(ctx, clientset)
	if err != nil {
		return errors.Wrapf(err, "unable to get CSI Node Status")
	}

	err = r.verifyCSIDriverRegisteration(ctx, clientset)
	if err != nil {
		return errors.Wrapf(err, "unable to register CSI Driver")
	}

	return nil

}

func (r *VDOConfigReconciler) reconcileCPIDeployment(ctx vdocontext.VDOContext) (bool, error) {
	var updateStatus bool //maintaining the variable updateStatus for all yaml deployments

	for _, deploymentYaml := range r.CpiDeploymentYamls {
		updateStatus, err := r.applyYaml(deploymentYaml, ctx, updateStatus, dynclient.CREATE)
		if err != nil {
			return updateStatus, err
		}
	}

	return updateStatus, nil
}

func (r *VDOConfigReconciler) reconcileCSIDeployment(ctx vdocontext.VDOContext) (bool, error) {
	var updateStatus bool //maintaining the variable updateStatus for all yaml deployments

	for _, deploymentYaml := range r.CsiDeploymentYamls {
		updateStatus, err := r.applyYaml(deploymentYaml, ctx, updateStatus, dynclient.CREATE)
		if err != nil {
			return updateStatus, err
		}
	}

	return updateStatus, nil
}

// deleteCPIDeployment deletes the currently deployed CPI Drivers
func (r *VDOConfigReconciler) deleteCPIDeployment(ctx vdocontext.VDOContext) (bool, error) {
	var updateStatus bool
	for _, deploymentYaml := range r.CpiDeploymentYamls {
		updateStatus, err := r.applyYaml(deploymentYaml, ctx, updateStatus, dynclient.DELETE)
		if err != nil {
			ctx.Logger.V(4).Info("Error occured when deleting the deployment for CPI : ", deploymentYaml, updateStatus)
		}
	}
	return updateStatus, nil
}

// deleteCSIDeployment deletes the currently deployed CSI Drivers
func (r *VDOConfigReconciler) deleteCSIDeployment(ctx vdocontext.VDOContext) (bool, error) {
	var updateStatus bool

	for _, deploymentYaml := range r.CsiDeploymentYamls {
		updateStatus, err := r.applyYaml(deploymentYaml, ctx, updateStatus, dynclient.DELETE)
		if err != nil {
			ctx.Logger.V(4).Info("Error occured when deleting the deployment for CSI : ", deploymentYaml, updateStatus)
		}
	}
	return updateStatus, nil
}

func (r *VDOConfigReconciler) applyYaml(yamlPath string, ctx vdocontext.VDOContext, updateStatus bool, action dynclient.Action) (bool, error) {
	ctx.Logger.V(4).Info("will attempt to apply spec file", "yamlPath", yamlPath)

	var exists bool
	var fileBytes []byte
	var err error

	if strings.Contains(yamlPath, "file://") {
		fileBytes, err = dynclient.GenerateYamlFromFilePath(yamlPath)
		if err != nil {
			return updateStatus, err
		}
	} else {
		fileBytes, err = dynclient.GenerateYamlFromUrl(yamlPath)
		if err != nil {
			return updateStatus, err
		}
	}

	_, err = dynclient.ParseAndProcessK8sObjects(ctx, r.Client, fileBytes, "", action)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			ctx.Logger.V(4).Info("given spec file already exists", "yamlPath", yamlPath)
			exists = true
		} else {
			ctx.Logger.V(4).Error(err, "unable to apply spec file", "yamlPath", yamlPath)
			return updateStatus, err
		}
	}
	if !exists {
		updateStatus = true
	}
	return updateStatus, nil
}

func (r *VDOConfigReconciler) updateCPIPhase(ctx context.Context, vdoConfig *vdov1alpha1.VDOConfig, phase vdov1alpha1.VDOConfigPhase, msg string) error {
	vdoConfig.Status.CPIStatus.Phase = phase
	vdoConfig.Status.CPIStatus.StatusMsg = msg
	r.Logger.Info("updating vdoConfig status phase", "vdoConfig", vdoConfig.Status.CPIStatus)
	updateErr := r.Status().Update(ctx, vdoConfig)
	if updateErr != nil {
		r.Logger.Error(updateErr, "error occurred when updating vdoConfig resource")
		return updateErr
	}
	return nil
}

func (r *VDOConfigReconciler) updateCSIPhase(ctx context.Context, vdoConfig *vdov1alpha1.VDOConfig, phase vdov1alpha1.VDOConfigPhase, msg string) error {
	vdoConfig.Status.CSIStatus.Phase = phase
	vdoConfig.Status.CSIStatus.StatusMsg = msg
	r.Logger.Info("updating vdoConfig status phase", "vdoConfig", vdoConfig.Status.CSIStatus)
	updateErr := r.Status().Update(ctx, vdoConfig)
	if updateErr != nil {
		r.Logger.Error(updateErr, "error occurred when updating vdoConfig resource")
		return updateErr
	}
	return nil
}

func (r *VDOConfigReconciler) updateVdoConfigWithNodeStatus(ctx context.Context, vdoConfig *vdov1alpha1.VDOConfig,
	phase vdov1alpha1.VDOConfigPhase, nodeState map[string]vdov1alpha1.NodeStatus) error {
	vdoConfig.Status.CPIStatus.Phase = phase
	vdoConfig.Status.CPIStatus.NodeStatus = nodeState
	r.Logger.Info("updating vdoConfig status phase", "vdoConfig", vdoConfig.Status.CPIStatus)
	updateErr := r.Status().Update(ctx, vdoConfig)
	if updateErr != nil {
		r.Logger.Error(updateErr, "error occurred when updating vdoConfig resource")
		return updateErr
	}
	return nil
}

func (r *VDOConfigReconciler) reconcileNodeLabel(ctx vdocontext.VDOContext, req ctrl.Request, clientset kubernetes.Interface, vdoConfig *vdov1alpha1.VDOConfig) error {

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrapf(err, "unable to fetch list of nodes")
	}
nodeloop:
	for _, node := range nodes.Items {
		if _, ok := vdoConfig.Status.CPIStatus.NodeStatus[node.Name]; ok {

			if node.Labels == nil {
				node.Labels = make(map[string]string)
			}

			if _, ok := node.Labels[VDO_NODE_LABEL_KEY]; ok {
				continue nodeloop
			}
			r.Logger.Info("adding node label", "name", VDO_NODE_LABEL_KEY, "node", node.Name)

			node.Labels[VDO_NODE_LABEL_KEY] = req.Name
			_, err := clientset.CoreV1().Nodes().Update(ctx, &node, metav1.UpdateOptions{})

			if err != nil {
				return errors.Wrapf(err, "Unable to update label on node")
			}
		}
	}

	return nil
}

func (r *VDOConfigReconciler) reconcileNodeProviderID(ctx vdocontext.VDOContext, config *vdov1alpha1.VDOConfig, clientset kubernetes.Interface, vsphereCloudConfigs *[]vdov1alpha1.VsphereCloudConfig) (bool, error) {
	nodeStatus := make(map[string]vdov1alpha1.NodeStatus)
	var updReq bool

	cpiStatus := vdov1alpha1.Configured
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return updReq, errors.Wrapf(err, "Error reconciling the providerID for CPI, unable to fetch list of nodes")
	}

nodeLoop:
	for _, node := range nodes.Items {
		if len(node.Spec.ProviderID) > 0 {
			nodeStatus[node.Name] = vdov1alpha1.NodeStatusReady
			r.Logger.Info("Adding to Available nodes", "node", node.Name)
			continue nodeLoop
		}

		for _, taint := range node.Spec.Taints {
			if taint.Key == CLOUD_PROVIDER_INIT_TAINT_KEY {

				if _, ok := config.Status.CPIStatus.NodeStatus[node.Name]; ok {
					nodeStatus[node.Name] = vdov1alpha1.NodeStatusPending
					cpiStatus = vdov1alpha1.Configuring
					continue nodeLoop
				}

				nodeExistsInVC, err := r.checkNodeExistence(ctx, vsphereCloudConfigs, node)
				if err != nil {
					return updReq, errors.Wrapf(err, "Error reconciling the providerID for CPI")
				}

				if nodeExistsInVC {
					nodeStatus[node.Name] = vdov1alpha1.NodeStatusPending
					cpiStatus = vdov1alpha1.Configuring
					continue nodeLoop
				}

				err = errors.Errorf(" Cloud Provider is not configured to manage the node %s. Please check your cloud Provider settings. ", node.Name)
				r.updateCPIStatusForError(ctx, err, config, err.Error())
				return updReq, err
			}

		}

	}

	if config.Status.CPIStatus.Phase != cpiStatus ||
		!reflect.DeepEqual(config.Status.CPIStatus.NodeStatus, nodeStatus) {
		config.Status.CPIStatus.NodeStatus = nodeStatus
		config.Status.CPIStatus.Phase = cpiStatus
		updReq = true
	}

	return updReq, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VDOConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vdov1alpha1.VDOConfig{}).
		Watches(
			&source.Kind{Type: &v1.Node{}},
			handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
				node, ok := object.(*v1.Node)
				r.Logger.Info("received reconcile request for node",
					"providerID", node.Spec.ProviderID, "labels", node.Labels)
				if !ok {
					r.Logger.Error(nil, fmt.Sprintf("expected a Node but got a %T", object))
					return nil
				}

				if len(node.Spec.ProviderID) > 0 {
					if vdoName, ok := node.Labels[VDO_NODE_LABEL_KEY]; ok {
						return []ctrl.Request{{
							NamespacedName: types.NamespacedName{
								Namespace: VDO_NAMESPACE,
								Name:      vdoName,
							},
						}}
					}
				}

				return nil
			})).
		Watches(
			&source.Kind{Type: &v1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
				configMap, _ := object.(*v1.ConfigMap)
				if configMap.Namespace == VDO_NAMESPACE && configMap.Name == CM_NAME {
					return []ctrl.Request{{
						NamespacedName: types.NamespacedName{
							Namespace: VDO_NAMESPACE,
							Name:      configMap.Name,
						}},
					}
				}
				return nil
			})).
		Complete(r)
}

func (r *VDOConfigReconciler) reconcileCPISecret(ctx vdocontext.VDOContext, config *vdov1alpha1.VDOConfig, cloudConfigs *[]vdov1alpha1.VsphereCloudConfig, cpiSecretKey types.NamespacedName) (*vdov1alpha1.VDOConfig, error) {
	cpiSecret := v1.Secret{}

	cpiSecretDataMap := make(map[string][]byte)
	for _, cloudConfig := range *cloudConfigs {

		ctx.Logger.V(4).Info("fetching vc credentials for CPI secret", "vsphereCloudConfig", cloudConfig)
		vcUser, vcUserPwd, err := r.fetchVcCredentials(ctx, cloudConfig)
		if err != nil {
			r.updateCPIStatusForError(ctx, err, config, "Error in fetching vc credentials for CPI configuration")
			return config, err
		}
		ctx.Logger.V(4).Info("adding VC section to CPI secret", "vsphereCloudConfig", cloudConfig)

		cpi.AddVCSectionToDataMap(cloudConfig, vcUser, vcUserPwd, cpiSecretDataMap)

	}

	err := r.Get(ctx, cpiSecretKey, &cpiSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctx.Logger.V(4).Info("creating new CPI secret")
			cpiSecret = cpi.CreateSecret(cpiSecretKey, cpiSecretDataMap)

			err = r.Create(ctx, &cpiSecret)
			if err != nil {
				r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("could not create cpi secret %s", cpiSecret.Name))
				return config, errors.Wrap(err, "error creating cpi secret")
			}

			ctx.Logger.V(4).Info("created CPI secret", "name", cpiSecret.Name)
			err = r.updateCPIPhase(ctx, config, vdov1alpha1.Configuring, "")
			return config, err
		}
		r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("unable to fetch secret %s", cpiSecret.Name))
		return config, err
	}

	cpiSecretIsSame := reflect.DeepEqual(cpiSecretDataMap, cpiSecret.Data)

	if !cpiSecretIsSame {
		ctx.Logger.V(4).Info("updating cpiSecret as it doesn't match vSphereCloudConfig resource")
		cpiSecret.Data = cpiSecretDataMap
		err = r.Update(ctx, &cpiSecret)
		if err != nil {
			ctx.Logger.Error(err, "error occurred when updating cpiSecret")
			r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("could not update cpi secret %s", cpiSecret.Name))
			return config, err
		}
		err = r.updateCPIPhase(ctx, config, vdov1alpha1.Configuring, "")
		return config, err

	}

	return config, nil
}

func (r *VDOConfigReconciler) reconcileConfigMap(ctx vdocontext.VDOContext, config *vdov1alpha1.VDOConfig, vsphereCloudConfigs *[]vdov1alpha1.VsphereCloudConfig, cpiSecretKey types.NamespacedName) (*vdov1alpha1.VDOConfig, error) {
	configMapKey := types.NamespacedName{
		Namespace: VC_CREDS_SECRET_NS,
		Name:      CONFIGMAP_NAME,
	}

	configDataMap, err := cpi.CreateVsphereConfig(config, *vsphereCloudConfigs, cpiSecretKey)
	if err != nil {
		r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("could not create vsphere configDataMap %s", CONFIGMAP_NAME))
		return config, err
	}

	vsphereConfigMap := v1.ConfigMap{}

	err = r.Get(ctx, configMapKey, &vsphereConfigMap)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctx.Logger.Info(" Creating new ConfigMap", "name", CONFIGMAP_NAME)

			vsphereConfigMap, err = cpi.CreateConfigMap(configDataMap, configMapKey)
			if err != nil {
				r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("could not create vsphere configmap %s", CONFIGMAP_NAME))
				return config, err
			}

			err := r.Create(ctx, &vsphereConfigMap)
			if err != nil && !apierrors.IsAlreadyExists(err) {
				r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("could not create vsphere configmap %s", CONFIGMAP_NAME))

				return config, err
			}

			err = r.updateCPIPhase(ctx, config, vdov1alpha1.Configuring, "")
			return config, err
		}

		r.updateCPIStatusForError(ctx, err, config, fmt.Sprintf("could not fetch configmap %s", CONFIGMAP_NAME))
		return config, err
	}

	configMapIsSame := reflect.DeepEqual(configDataMap, vsphereConfigMap.Data)
	if !configMapIsSame {
		ctx.Logger.Info("updating ConfigMap as it doesn't match vSphereCloudConfig resource")
		vsphereConfigMap.Data = configDataMap
		err = r.Update(ctx, &vsphereConfigMap)
		if err != nil {
			ctx.Logger.Error(err, "error occurred when updating ConfigMap")
			return config, err
		}
		err = r.updateCPIPhase(ctx, config, vdov1alpha1.Configuring, "")
		return config, err
	}

	return config, nil
}

func (r *VDOConfigReconciler) reconcileCSISecret(ctx vdocontext.VDOContext, config *vdov1alpha1.VDOConfig, vsphereCloudConfig *vdov1alpha1.VsphereCloudConfig) (*vdov1alpha1.VDOConfig, error) {

	ctx.Logger.V(4).Info("fetching vc credentials for CSI")
	vcUser, vcUserPwd, err := r.fetchVcCredentials(ctx, *vsphereCloudConfig)
	if err != nil {
		r.updateCSIStatusForError(ctx, err, config, "Error in reconcile of fetching vc credentials for CSI configuration")
		return config, err
	}

	csiSecretKey := types.NamespacedName{
		Namespace: VC_CREDS_SECRET_NS,
		Name:      CSI_SECRET_NAME,
	}

	csiSecret := v1.Secret{}
	ctx.Logger.V(4).Info("creating CSI secret config")
	configData, err := csi.CreateCSISecretConfig(config, vsphereCloudConfig, vcUser, vcUserPwd, CSI_SECRET_CONFIG_FILE)
	if err != nil {
		r.updateCSIStatusForError(ctx, err, config, "unable to create csi config")
		return config, err
	}

	err = r.Get(ctx, csiSecretKey, &csiSecret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctx.Logger.V(4).Info("creating new CSI secret")
			csiSecret = csi.CreateCSISecret(configData, csiSecretKey)

			err = r.Create(ctx, &csiSecret)
			if err != nil {
				return config, errors.Wrap(err, fmt.Sprintf("could not create csi secret %s", csiSecret.Name))
			}

			ctx.Logger.V(4).Info("created CSI secret", "name", csiSecret.Name)
			err = r.updateCSIPhase(ctx, config, vdov1alpha1.Configuring, "")
			return config, err
		}
		ctx.Logger.Error(err, fmt.Sprintf("unable to fetch CSI secret %s", csiSecret.Name))
		return config, err
	}

	csiSecretIsSame := csi.CompareCSISecret(&csiSecret, configData)
	if !csiSecretIsSame {
		ctx.Logger.V(4).Info("updating csiSecret as it doesn't match vSphereCloudConfig resource")
		csi.UpdateCSISecret(&csiSecret, configData)
		err = r.Update(ctx, &csiSecret)
		if err != nil {
			return config, errors.Wrapf(err, fmt.Sprintf("could not update csi secret %s", csiSecret.Name))
		}
		err = r.updateCPIPhase(ctx, config, vdov1alpha1.Configuring, "")
		return config, err

	}

	return config, nil
}

// compareVersions checks if the given version lies between min and max version
func (r *VDOConfigReconciler) compareVersions(minVersion, currentVersion, maxVersion string) (bool, error) {

	minVer, err := version.NewVersion(minVersion)
	if err != nil {
		return false, err
	}

	currentVer, err := version.NewVersion(currentVersion)
	if err != nil {
		return false, err
	}

	maxVer, err := version.NewVersion(maxVersion)
	if err != nil {
		return false, err
	}

	if minVer.LessThanOrEqual(currentVer) && maxVer.GreaterThanOrEqual(currentVer) {
		return true, nil
	}

	return false, nil
}

// compareSkewVersions checks if the given version matches with the skew version
func (r *VDOConfigReconciler) compareSkewVersions(currentVersion, supportedVersion string) (bool, error) {

	currentVer, err := version.NewVersion(currentVersion)
	if err != nil {
		return false, err
	}

	supportedVer, err := version.NewVersion(supportedVersion)
	if err != nil {
		return false, err
	}

	if currentVer.Equal(supportedVer) {
		return true, nil
	}

	return false, nil

}

func (r *VDOConfigReconciler) FetchCsiDeploymentYamls(ctx vdocontext.VDOContext, matrix CompatMatrix, vSphereVersions []string, k8sVersion string) error {
	var versionList []string

	for ver := range matrix.CSISpecList {
		versionList = append(versionList, ver)
	}
	sort.Strings(versionList)
	var csiVersion string

	ctx.Logger.V(4).Info("vSphere Versions ", "version", vSphereVersions)
	ctx.Logger.V(4).Info("k8s Versions ", "version", k8sVersion)

	for _, vSphereVersion := range vSphereVersions {
		for v := len(versionList) - 1; v >= 0; v-- {
			isVsphereVersion, err := r.compareVersions(matrix.CSISpecList[versionList[v]].VSphereVersion.Min, vSphereVersion, matrix.CSISpecList[versionList[v]].VSphereVersion.Max)

			if err != nil {
				return err
			}

			isK8sVersion, err := r.compareVersions(matrix.CSISpecList[versionList[v]].K8sVersion.Min, k8sVersion, matrix.CSISpecList[versionList[v]].K8sVersion.Max)

			if err != nil {
				return err
			}

			if isVsphereVersion && isK8sVersion {
				csiVersion = versionList[v]
				break
			}
		}
	}

	// If the current evaluated versions is not equals to deployed version
	// then delete the current deployment
	if csiVersion != r.CurrentCSIDeployedVersion && r.CurrentCSIDeployedVersion != "" {
		ctx.Logger.V(4).Info("Deleting the CSI Deployment for the version : ", r.CurrentCSIDeployedVersion)

		_, err := r.deleteCSIDeployment(ctx)
		if err != nil {
			return err
		}
		// Re-initialize the Deployment Yamls
		r.CsiDeploymentYamls = []string{}
	}

	if len(csiVersion) <= 0 {
		return errors.New("could not fetch compatible CSI version for vSphere version and k8s version ")
	}

	ctx.Logger.V(4).Info("Corresponding CSI Version ", "version", csiVersion)

	r.CsiDeploymentYamls = matrix.CSISpecList[csiVersion].DeploymentPaths
	r.CurrentCSIDeployedVersion = csiVersion

	return nil
}

func (r *VDOConfigReconciler) FetchCpiDeploymentYamls(ctx vdocontext.VDOContext, matrix CompatMatrix, vSphereVersions []string, k8sVersion string) error {
	var versionList []string

	for ver := range matrix.CPISpecList {
		versionList = append(versionList, ver)
	}
	sort.Strings(versionList)

	var cpiVersion string

	ctx.Logger.V(4).Info("vSphere Versions ", "version", vSphereVersions)
	ctx.Logger.V(4).Info("k8s Versions ", "version", k8sVersion)

	for _, vSphereVersion := range vSphereVersions {
		for v := len(versionList) - 1; v >= 0; v-- {
			isVsphereVersion, err := r.compareVersions(matrix.CPISpecList[versionList[v]].VSphereVersion.Min, vSphereVersion, matrix.CPISpecList[versionList[v]].VSphereVersion.Max)
			if err != nil {
				return err
			}

			isK8sVersion, err := r.compareSkewVersions(k8sVersion, matrix.CPISpecList[versionList[v]].K8sVersion.SkewVersion)
			if err != nil {
				return err
			}

			if isVsphereVersion && isK8sVersion {
				cpiVersion = versionList[v]
				break
			}
		}
	}

	// If the current evaluated versions is not equals to deployed version
	// then delete the current deployment
	if cpiVersion != r.CurrentCPIDeployedVersion && r.CurrentCPIDeployedVersion != "" {
		ctx.Logger.V(4).Info("Deleting the CPI Deployment for the version : ", r.CurrentCPIDeployedVersion)

		_, err := r.deleteCPIDeployment(ctx)
		if err != nil {
			return err
		}
		// Re-initialize the deployment yamls
		r.CpiDeploymentYamls = []string{}
	}

	if len(cpiVersion) <= 0 {
		return errors.New("could not fetch compatible CPI version for vSphere version and k8s version ")
	}

	ctx.Logger.V(4).Info("Corresponding CPI Version ", "version", cpiVersion)

	r.CpiDeploymentYamls = matrix.CPISpecList[cpiVersion].DeploymentPaths
	r.CurrentCPIDeployedVersion = cpiVersion

	return nil
}

func (r *VDOConfigReconciler) CheckCompatAndRetrieveSpec(ctx vdocontext.VDOContext, req ctrl.Request, vdoConfig *vdov1alpha1.VDOConfig, matrixConfig string) error {

	var matrix CompatMatrix

	k8sVersion, err := r.Fetchk8sVersions(ctx)
	if err != nil {
		ctx.Logger.Error(err, "Error occurred when fetching k8sVersion")
		return err
	}

	vSphereVersions, err := r.FetchVsphereVersions(ctx, req, vdoConfig)
	if err != nil {
		ctx.Logger.Error(err, "Error occurred when fetching vSphereVersions")
		return err
	}

	if matrixConfig == os.Getenv(COMPAT_MATRIX_CONFIG_URL) {
		matrix, err = dynclient.ParseMatrixYaml(matrixConfig)
		if err != nil {
			ctx.Logger.Error(err, "Error occurred when Parsing the matrix yaml", "Path", matrixConfig)
			return err
		}
	} else {
		err = json.Unmarshal([]byte(matrixConfig), &matrix)
		if err != nil {
			ctx.Logger.Error(err, "Error occurred when Parsing the matrix yaml", "Contents", matrixConfig)
			return err
		}
	}

	err = r.FetchCpiDeploymentYamls(ctx, matrix, vSphereVersions, k8sVersion)
	if err != nil {
		ctx.Logger.Error(err, "Error occurred when fetching the CPI deployment yamls")
		return err
	}

	err = r.FetchCsiDeploymentYamls(ctx, matrix, vSphereVersions, k8sVersion)
	if err != nil {
		ctx.Logger.Error(err, "Error occurred when fetching the CSI deployment yamls")
		return err
	}

	return nil
}

func (r *VDOConfigReconciler) checkNodeExistence(ctx vdocontext.VDOContext, vsphereCloudConfigs *[]vdov1alpha1.VsphereCloudConfig, node v1.Node) (bool, error) {

	for _, cloudConfig := range *vsphereCloudConfigs {
		sess, err := r.getVcSession(ctx, &cloudConfig)
		if err != nil {
			return false, err
		}

		var nodeIP string
		if len(node.Status.Addresses) > 0 {
			for _, address := range node.Status.Addresses {
				if address.Type == v1.NodeInternalIP {
					nodeIP = address.Address
				}
			}
		}

		vm, err := GetVMFn(ctx, nodeIP, sess.Datacenters)
		if err != nil {
			return false, err
		}

		if vm != nil {
			return true, nil
		}

	}
	return false, nil
}

func (r *VDOConfigReconciler) getMatrixConfig(matrixConfigUrl, matrixConfigContent string) (string, error) {

	var err error

	if matrixConfigUrl != "" && matrixConfigContent == "" {
		return matrixConfigUrl, nil
	} else if matrixConfigUrl == "" && matrixConfigContent != "" {
		return matrixConfigContent, nil
	} else {
		err = errors.New("Matrix Config URL/Content not provided in proper format")
		return "", err
	}
}
