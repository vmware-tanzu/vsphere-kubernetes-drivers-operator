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
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	vdov1alpha1 "github.com/vmware-tanzu/vsphere-kubernetes-drivers-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CompatibiltyConfigReconciler reconciles a VsphereCloudConfig object
type CompatibiltyConfigReconciler struct {
	client.Client
	Logger logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=vdo.vmware.com,resources=compatibilityconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=compatibilityconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=vdo.vmware.com,resources=compatibilityconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;get;list;watch;update;patch;
// +kubebuilder:rbac:groups=*,resources=namespaces,verbs=get;list;watch;update;patch;

func (r *CompatibiltyConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Logger.WithValues("name", req.Name, "namespace", req.Namespace)

	logger.V(4).Info("processing CompatibilityConfig reconcile")
	if req.NamespacedName.Name != "" || req.Namespace != "" {
		return ctrl.Result{}, nil
	}
	compatibilityConfig := &vdov1alpha1.CompatibilityConfig{}
	if err := r.Get(ctx, req.NamespacedName, compatibilityConfig); err != nil {
		logger.Error(err, "Error occurred when fetching CompatibilityConfig resource", "name", req.NamespacedName)
		return ctrl.Result{}, errors.Wrapf(err, "could not fetch CompatibilityConfig resource %s", req.NamespacedName)
	}
	//Update the ConfigMap

	if compatibilityConfig.Spec.MatrixURL != "" {
		//Get ConfigMap
		configMap := &v1.ConfigMap{}
		err := r.Get(ctx, req.NamespacedName, configMap)
		if err != nil {
			logger.Error(err, "Error while getting the CompatibilityConfigMap")
			return ctrl.Result{}, err
		}
		// Update ConfigMap
		configMap.Data[CM_URL_KEY] = compatibilityConfig.Spec.MatrixURL
		err = r.Update(ctx, configMap)
		if err != nil {
			logger.Error(err, "Error while updating the CompatibilityConfigMap")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CompatibiltyConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vdov1alpha1.CompatibilityConfig{}).
		Watches(&source.Kind{Type: &vdov1alpha1.CompatibilityConfig{}},
			handler.EnqueueRequestsFromMapFunc(func(object client.Object) []reconcile.Request {
				compatibilityConfig, _ := object.(*vdov1alpha1.CompatibilityConfig)
				if compatibilityConfig.Namespace == VDO_NAMESPACE {
					return []ctrl.Request{{
						NamespacedName: types.NamespacedName{
							Namespace: VDO_NAMESPACE,
							Name:      compatibilityConfig.Name,
						}},
					}
				}
				return nil
			}),
		).
		Complete(r)
}
