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
	"fmt"

	"github.com/go-logr/logr"
	iprulerv1 "github.com/plutocholia/ipruler-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// FullConfigReconciler reconciles a FullConfig object
type FullConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=fullconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=fullconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ipruler.pegah.tech,resources=fullconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FullConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.18.2/pkg/reconcile
func (r *FullConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var fullConfig iprulerv1.FullConfig
	if err := r.Get(ctx, req.NamespacedName, &fullConfig); err != nil {
		if apierrors.IsNotFound(err) {
			r.Log.Info("resource has been deleted", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the resource is being deleted
	if !fullConfig.ObjectMeta.DeletionTimestamp.IsZero() {
		r.Log.Info("resource is being deleted", "namespace", req.Namespace, "name", req.Name)
		if res, err := r.handleDeletion(ctx, &fullConfig); err != nil {
			return res, err
		}
		return ctrl.Result{}, nil
	}

	if res, err := r.handleUpdateOrCreate(ctx, &fullConfig); err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *FullConfigReconciler) handleUpdateOrCreate(ctx context.Context, fullConfig *iprulerv1.FullConfig) (ctrl.Result, error) {
	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.MatchingLabels{globalAgentManager.AppLabelKey: globalAgentManager.AppLabelValue}, client.InNamespace(globalAgentManager.Namespace)); err != nil {
		r.Log.Error(err, "Failed to get the pods list")
		return ctrl.Result{}, err
	}
	for _, pod := range podList.Items {
		if PodIsReadyForConfigInjection(&pod) {
			var node corev1.Node
			if err := r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, &node); err != nil {
				r.Log.Error(err, "message", "Failed to get Node for Pod", "Pod", pod.Name)
				return ctrl.Result{Requeue: true}, err
			}

			labelMatch := true
			nodeLabels := node.GetLabels()
			for key, value := range fullConfig.Spec.NodeSelector {
				if nodeLabels[key] != value {
					labelMatch = false
					break
				}
			}
			if labelMatch {
				fmt.Println(fullConfig.Spec.MergedConfig)
				globalAgentManager.InjectConfig(&pod, &fullConfig.Spec.MergedConfig)
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *FullConfigReconciler) handleDeletion(ctx context.Context, fullConfig *iprulerv1.FullConfig) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FullConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&iprulerv1.FullConfig{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				return true
			},
			CreateFunc: func(e event.CreateEvent) bool {
				return true
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return false
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return false
			},
		}).
		Complete(r)
}
