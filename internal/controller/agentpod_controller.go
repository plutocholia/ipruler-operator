package controller

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type AgentPodsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch,namespace=kube-system
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
func (r *AgentPodsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// pod := &corev1.Pod{}
	// if err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, pod); err != nil {
	// 	r.Log.Error(err, "Failed to get the pod")
	// 	return reconcile.Result{}, err
	// }

	// if PodIsReadyForConfigInjection(pod) {
	// 	var node corev1.Node
	// 	if err := r.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, &node); err != nil {
	// 		r.Log.Error(err, "message", "Failed to get Node for Pod", "Pod", pod.Name)
	// 		return reconcile.Result{}, err
	// 	}
	// 	nodeLabels := node.GetLabels()
	// 	nodeConfig := globalAgentManager.FindNodeConfigByLabelList(nodeLabels)
	// 	if nodeConfig != nil {
	// 		globalAgentManager.InjectConfig(pod, nodeConfig)
	// 	} else {
	// 		r.Log.Info("No NodeConfig is available for ", "pod", pod.Name)
	// 	}
	// }

	return ctrl.Result{}, nil
}

func (r *AgentPodsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	podPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			pod := e.Object.(*corev1.Pod)
			if pod.Labels[globalAgentManager.AppLabelKey] == globalAgentManager.AppLabelValue &&
				pod.Namespace == globalAgentManager.Namespace {
				r.Log.Info("Create event", "namespace", e.Object.GetNamespace(), "name", e.Object.GetName())
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			pod := e.ObjectNew.(*corev1.Pod)
			if pod.Labels[globalAgentManager.AppLabelKey] == globalAgentManager.AppLabelValue &&
				pod.Namespace == globalAgentManager.Namespace {
				r.Log.Info("Update event", "namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName())
				return true
			}
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		WithEventFilter(podPredicate).
		Complete(r)
}
