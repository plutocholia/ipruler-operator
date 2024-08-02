/*
Copyright 2024.

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

package controller

import (
	"context"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	iprulerv1 "github.com/plutocholia/ipruler-controller/api/v1"
	"github.com/plutocholia/ipruler-controller/internal/models"
)

// ClusterConfigReconciler reconciles a ClusterConfig object
type ClusterConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=clusterconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=clusterconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=clusterconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *ClusterConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var clusterConfig iprulerv1.ClusterConfig
	if err := r.Get(ctx, req.NamespacedName, &clusterConfig); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("resource has been deleted", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the resource is being deleted
	if !clusterConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		r.Log.Info("resource is being deleted", "namespace", req.Namespace, "name", req.Name)
		if res, err := r.handleDeletion(ctx, &clusterConfig); err != nil {
			return res, err
		}
		return ctrl.Result{}, nil
	}

	if res, err := r.handleUpdateOrCreate(ctx, &clusterConfig); err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *ClusterConfigReconciler) handleUpdateOrCreate(ctx context.Context, clusterConfig *iprulerv1.ClusterConfig) (ctrl.Result, error) {

	sharedFullConfig.ClusterConfigName = clusterConfig.Name
	sharedFullConfig.ClusterConfigNamespace = clusterConfig.Namespace

	fullConfigList := &iprulerv1.FullConfigList{}
	err := r.Client.List(ctx, fullConfigList)
	if err != nil {
		r.Log.Error(err, "Failed to List FullConfig")
		return ctrl.Result{}, err
	}

	if len(fullConfigList.Items) == 0 {
		return ctrl.Result{Requeue: true}, nil
	}

	// update ClusterConfig and MergedConfig Part
	for _, fullConfig := range fullConfigList.Items {
		if !reflect.DeepEqual(fullConfig.Spec.ClusterConfig, clusterConfig.Spec.Config) {
			fullConfig.Spec.ClusterConfig = clusterConfig.Spec.Config
			fullConfig.Spec.MergedConfig = models.MergeConfigModels(&clusterConfig.Spec.Config, &fullConfig.Spec.NodeConfig)
			err = r.Client.Update(ctx, &fullConfig)
			if err != nil {
				r.Log.Error(err, "Failed to update FullConfig", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
				return ctrl.Result{}, err
			}
			r.Log.Info("Updated FullConfig", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	// update status
	for _, fullConfig := range fullConfigList.Items {
		fullConfig.Status.HasClusterConfig = true
		err = r.Client.Status().Update(ctx, &fullConfig)
		if err != nil {
			r.Log.Error(err, "Failed to update FullConfig status", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
			return ctrl.Result{Requeue: true}, err
		} else {
			r.Log.Info("Updated FullConfig status", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
		}
	}

	return ctrl.Result{}, nil
}

func (r *ClusterConfigReconciler) handleDeletion(ctx context.Context, clusterConfig *iprulerv1.ClusterConfig) (ctrl.Result, error) {

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.

func (r *ClusterConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iprulerv1.ClusterConfig{}).
		Watches(
			&iprulerv1.FullConfig{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForFullConfig),
		).
		Complete(r)
}

func (r *ClusterConfigReconciler) findObjectsForFullConfig(ctx context.Context, fullConfig client.Object) []ctrl.Request {

	requests := []ctrl.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      sharedFullConfig.ClusterConfigName,
				Namespace: sharedFullConfig.ClusterConfigNamespace,
			},
		},
	}

	return requests
}
