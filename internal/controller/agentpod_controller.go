package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type AgentPodsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch,namespace=kube-system
func (r *AgentPodsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	pod := &corev1.Pod{}
	if err := r.Get(ctx, client.ObjectKey{Name: req.Name, Namespace: req.Namespace}, pod); err != nil {
		setupLog.Error(err, "Failed to get the pod")
		return reconcile.Result{}, err
	}

	if PodIsReadyForConfigInjection(pod) {
		globalAgentManager.Mutex.Lock()
		defer globalAgentManager.Mutex.Unlock()
		globalAgentManager.InjectConfig(pod)
	}

	return ctrl.Result{}, nil
}

func (r *AgentPodsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	podPredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			pod := e.Object.(*corev1.Pod)
			if pod.Labels[globalAgentManager.AppLabelKey] == globalAgentManager.AppLabelValue &&
				pod.Namespace == globalAgentManager.Namespace {
				setupLog.Info("Create event", "namespace", e.Object.GetNamespace(), "name", e.Object.GetName())
				return true
			}
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			pod := e.ObjectNew.(*corev1.Pod)
			if pod.Labels[globalAgentManager.AppLabelKey] == globalAgentManager.AppLabelValue &&
				pod.Namespace == globalAgentManager.Namespace {
				setupLog.Info("Update event", "namespace", e.ObjectNew.GetNamespace(), "name", e.ObjectNew.GetName())
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
