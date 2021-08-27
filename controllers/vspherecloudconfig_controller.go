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
	"fmt"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/pkg/session"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VsphereCloudConfigReconciler reconciles a VsphereCloudConfig object
type VsphereCloudConfigReconciler struct {
	client.Client
	Logger logr.Logger
	Scheme *runtime.Scheme
}

type StatusType string

const VC_CREDS_SECRET_NS = "kube-system"

// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vspherecloudconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vspherecloudconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=vspherecloudconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *VsphereCloudConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("name", req.Name, "namespace", req.Namespace)

	logger.V(4).Info("processing vSphereCloudConfig reconcile")

	config := &vdov1alpha1.VsphereCloudConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		logger.Error(err, "Error occurred when fetching vSphereCloudConfig resource", "name", req.NamespacedName)
		return ctrl.Result{}, errors.Wrapf(err, "could not fetch vSphereCLoudConfig resource %s", req.NamespacedName)
	}

	config, err := r.reconcileVCCredentials(ctx, config)
	if err != nil {
		logger.Error(err, "error occurred when reconciling vSphere credentials", "vcIp", config.Spec.VcIP)
		updateErr := r.Status().Update(ctx, config)
		if updateErr != nil {
			logger.Error(updateErr, "error occurred when updating vSphereCloudConfig resource", "vcIp", config.Spec.VcIP)
		}
		return ctrl.Result{}, err

	}

	logger.V(4).Info("updating config status", "vcIp", config.Spec.VcIP, "config", config.Status.Config)
	err = r.Status().Update(ctx, config)
	if err != nil {
		logger.Error(err, "error occurred when updating vSphereCloudConfig resource", "vcIp", config.Spec.VcIP)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VsphereCloudConfigReconciler) reconcileVCCredentials(ctx context.Context, config *vdov1alpha1.VsphereCloudConfig) (*vdov1alpha1.VsphereCloudConfig, error) {
	var vcUser, vcUserPwd, datacenter string

	if len(config.Spec.Credentials) > 0 {
		vcCredsSecret := &v1.Secret{}
		key := types.NamespacedName{
			Namespace: VC_CREDS_SECRET_NS,
			Name:      config.Spec.Credentials,
		}

		err := r.Get(ctx, key, vcCredsSecret)
		if err != nil {
			config.Status.Config = vdov1alpha1.VsphereConfigFailed
			config.Status.Message = fmt.Sprintf("could not fetch vc credentials secret %s", config.Spec.Credentials)
			return config, errors.Wrapf(err, "could not fetch vc credentials secret %s", config.Spec.Credentials)
		}

		vcUser = string(vcCredsSecret.Data["username"])
		vcUserPwd = string(vcCredsSecret.Data["password"])
	}

	if len(config.Spec.DataCenters) > 0 {
		datacenter = config.Spec.DataCenters[0]
	}

	vcIp := config.Spec.VcIP
	sess, err := session.GetOrCreate(ctx, vcIp, datacenter, vcUser, vcUserPwd, config.Spec.Thumbprint)
	if err != nil {
		config.Status.Config = vdov1alpha1.VsphereConfigFailed
		config.Status.Message = fmt.Sprintf("Error establishing session with vcenter %s for user %s", vcIp, vcUser)
		return config, errors.Wrapf(err, "Error establishing session with vcenter %s for user %s", vcIp, vcUser)
	}

	if sess != nil {
		state, err := sess.SessionManager.SessionIsActive(ctx)
		if err != nil {
			config.Status.Config = vdov1alpha1.VsphereConfigFailed
			config.Status.Message = fmt.Sprintf("unable to verify session for vc %s", vcIp)
			return config, errors.Wrapf(err, "unable to verify session for vc %s", vcIp)
		}

		r.Logger.V(4).Info("verified vc session", "isActive", state)

		config.Status.Config = vdov1alpha1.VsphereConfigVerified
		config.Status.Message = ""
	}
	return config, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VsphereCloudConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vdov1alpha1.VsphereCloudConfig{}).
		Complete(r)
}
