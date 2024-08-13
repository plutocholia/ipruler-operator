package controller

import (
	"context"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	iprulerv1 "github.com/plutocholia/ipruler-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type NodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
func (r *NodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var node corev1.Node
	if err := r.Get(ctx, req.NamespacedName, &node); err != nil {
		// check if the resource is deleded already, Ignore returning with error to prevent reconcile dead loop due to requeuing the not nil error results
		if apierrors.IsNotFound(err) {
			r.Log.Info("resource has been deleted", "namespace", req.NamespacedName.Namespace, "name", req.NamespacedName.Name)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the resource is being deleted
	if !node.ObjectMeta.DeletionTimestamp.IsZero() {
		// The resource is being deleted
		r.Log.Info("resource is being deleted", "namespace", req.Namespace, "name", req.Name)
		if res, err := r.handleDeletion(ctx, &node); err != nil {
			return res, err
		}
		return ctrl.Result{}, nil
	}

	// Checking if the Node is Ready
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			// Your logic for handling ready nodes here
			if res, err := r.handleUpdateOrCreate(ctx, &node); err != nil {
				return res, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *NodeReconciler) handleUpdateOrCreate(ctx context.Context, node *corev1.Node) (ctrl.Result, error) {
	// get FullConfig List
	fullConfigList := &iprulerv1.FullConfigList{}
	if err := r.Client.List(ctx, fullConfigList); err != nil {
		r.Log.Error(err, "Failed to List FullConfig")
		return ctrl.Result{}, err
	}

	// find the FullConfig corresponding to the node
	var matchedFullConfig *iprulerv1.FullConfig
	for _, fullConfig := range fullConfigList.Items {
		fullConfigMatchTheNode := true
		for fullConfigLabelKey, fullConfigLabelValue := range fullConfig.Spec.NodeSelector {
			if node.GetLabels()[fullConfigLabelKey] != fullConfigLabelValue {
				fullConfigMatchTheNode = false
				break
			}
		}
		if fullConfigMatchTheNode {
			matchedFullConfig = &fullConfig
		}
	}

	// trigger the FullConfig resource for doing request stuff
	if matchedFullConfig != nil {
		if matchedFullConfig.Annotations == nil {
			matchedFullConfig.Annotations = map[string]string{}
		}
		matchedFullConfig.Annotations["lastUpdateTrigger"] = time.Now().Format(time.RFC3339)
		if err := r.Client.Update(ctx, matchedFullConfig); err != nil && apierrors.IsConflict(err) {
			r.Log.Info("Conflict in resource when updating lastUpdateTrigger annotation. The given FullConfig has been changed", "Namespace", matchedFullConfig.Namespace, "Name", matchedFullConfig.Name)
			return ctrl.Result{}, err
		} else if err != nil {
			r.Log.Error(err, "Failed to update FullConfig on lastUpdateTrigger annotation", "Namespace", matchedFullConfig.Namespace, "Name", matchedFullConfig.Name)
			return ctrl.Result{}, err
		} else {
			r.Log.Info("Updated FullConfig on lastUpdateTrigger", "Namespace", matchedFullConfig.Namespace, "Name", matchedFullConfig.Name)
		}

		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func (r *NodeReconciler) handleDeletion(ctx context.Context, node *corev1.Node) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				// TODO: implement if a node is added
				return true
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				oldNode := e.ObjectOld.(*corev1.Node)
				newNode := e.ObjectNew.(*corev1.Node)
				// if there is changes in the node labels or node status
				return !reflect.DeepEqual(oldNode.GetLabels(), newNode.GetLabels()) || !reflect.DeepEqual(oldNode.Status.Conditions, newNode.Status.Conditions)
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
