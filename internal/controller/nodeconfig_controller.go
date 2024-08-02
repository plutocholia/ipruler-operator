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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/go-logr/logr"
	iprulerv1 "github.com/plutocholia/ipruler-controller/api/v1"
	"github.com/plutocholia/ipruler-controller/internal/models"
)

// NodeConfigReconciler reconciles a NodeConfig object
type NodeConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=nodeconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=nodeconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=nodeconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *NodeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var nodeConfig iprulerv1.NodeConfig
	if err := r.Get(ctx, req.NamespacedName, &nodeConfig); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("resource has been deleted", "namespace", req.Namespace, "name", req.Name)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the resource is being deleted
	if !nodeConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		// The resource is being deleted
		r.Log.Info("resource is being deleted", "namespace", req.Namespace, "name", req.Name)
		if res, err := r.handleDeletion(ctx, &nodeConfig); err != nil {
			return res, err
		}
		return ctrl.Result{}, nil
	}

	// The resource is not being deleted, handle update or create
	if res, err := r.handleUpdateOrCreate(ctx, &nodeConfig); err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *NodeConfigReconciler) handleDeletion(ctx context.Context, nodeConfig *iprulerv1.NodeConfig) (ctrl.Result, error) {
	// globalAgentManager.DeleteNodeConfig(nodeConfig)
	return ctrl.Result{}, nil
}

func (r *NodeConfigReconciler) handleUpdateOrCreate(ctx context.Context, nodeConfig *iprulerv1.NodeConfig) (ctrl.Result, error) {

	// Check if the FullConfig already exists
	fullConfig := &iprulerv1.FullConfig{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: nodeConfig.Name, Namespace: nodeConfig.Namespace}, fullConfig)
	if err != nil && apierrors.IsNotFound(err) {

		// Create a new FullConfig
		newFullConfig := &iprulerv1.FullConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeConfig.Name,
				Namespace: nodeConfig.Namespace,
			},
			Spec: iprulerv1.FullConfigSpec{
				NodeSelector: nodeConfig.Spec.NodeSelector,
				NodeConfig:   nodeConfig.Spec.Config,
			},
		}

		// Set NodeConfig instance as the owner and controller
		if err := controllerutil.SetControllerReference(nodeConfig, newFullConfig, r.Scheme); err != nil {
			r.Log.Error(err, "Failed to set owner reference on new FullConfig")
			return ctrl.Result{}, err
		}

		r.Log.Info("Creating a new FullConfig", "Namespace", newFullConfig.Namespace, "Name", newFullConfig.Name)

		err = r.Client.Create(ctx, newFullConfig)
		if err != nil {
			r.Log.Error(err, "Failed to create new FullConfig", "Namespace", newFullConfig.Namespace, "Name", newFullConfig.Name)
			return ctrl.Result{}, err
		}

		// FullConfig created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		r.Log.Error(err, "Failed to get FullConfig")
		return ctrl.Result{}, err
	}

	// Check if the FullConfig needs to be updated
	if !reflect.DeepEqual(fullConfig.Spec.NodeConfig, nodeConfig.Spec.Config) {
		// update spec
		fullConfig.Spec.NodeSelector = nodeConfig.Spec.NodeSelector
		fullConfig.Spec.NodeConfig = nodeConfig.Spec.Config
		fullConfig.Spec.MergedConfig = models.MergeConfigModels(&nodeConfig.Spec.Config, &fullConfig.Spec.ClusterConfig)
		err = r.Client.Update(ctx, fullConfig)
		if err != nil {
			r.Log.Error(err, "Failed to update FullConfig", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
			return ctrl.Result{}, err
		}
		r.Log.Info("Updated FullConfig", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
		return ctrl.Result{Requeue: true}, nil
	}

	// update status
	if !fullConfig.Status.HasNodeConfig {
		fullConfig.Status.HasNodeConfig = true
		err = r.Client.Status().Update(ctx, fullConfig)
		if err != nil && apierrors.IsConflict(err) {
			r.Log.Error(err, "Conflict in resource, The given FullConfig is changed", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
			return ctrl.Result{}, nil
		} else if err != nil {
			r.Log.Error(err, "Failed to update FullConfig status", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
			return ctrl.Result{}, err
		} else {
			r.Log.Info("Updated FullConfig status", "Namespace", fullConfig.Namespace, "Name", fullConfig.Name)
		}
	}

	// FullConfig already exists and is up-to-date - don't requeue
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iprulerv1.NodeConfig{}).
		Owns(&iprulerv1.FullConfig{}).
		Complete(r)
}
