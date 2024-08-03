package controller

import (
	"context"
	"time"

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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

	var pod corev1.Pod
	if err := r.Get(ctx, req.NamespacedName, &pod); err != nil {
		// check if the resource has been deleded already, Ignore returning with error to prevent reconcile dead loop due to requeuing the not nil error results
		if apierrors.IsNotFound(err) {
			r.Log.Info("resource has been deleted", "namespace", req.NamespacedName.Namespace, "name", req.NamespacedName.Name)
			return reconcile.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the resource is being deleted
	if !pod.ObjectMeta.DeletionTimestamp.IsZero() {
		// The resource is being deleted
		r.Log.Info("resource is being deleted", "namespace", req.Namespace, "name", req.Name)
		if res, err := r.handleDeletion(ctx, &pod); err != nil {
			return res, err
		}
		return ctrl.Result{}, nil
	}

	if res, err := r.handleUpdateOrCreate(ctx, &pod); err != nil {
		return res, err
	}

	return ctrl.Result{}, nil
}

func (r *AgentPodsReconciler) handleDeletion(ctx context.Context, pod *corev1.Pod) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *AgentPodsReconciler) handleUpdateOrCreate(ctx context.Context, pod *corev1.Pod) (ctrl.Result, error) {

	if !PodIsReady(pod) {
		return ctrl.Result{Requeue: true}, nil
	}

	// Fetch the Node object using the NodeName from the Pod
	var node corev1.Node
	if err := r.Client.Get(ctx, client.ObjectKey{Name: pod.Spec.NodeName}, &node); err != nil {
		r.Log.Error(err, "unable to fetch Node", "NodeName", pod.Spec.NodeName)
		return ctrl.Result{}, err
	}

	// get FullConfig list
	fullConfigList := &iprulerv1.FullConfigList{}
	if err := r.Client.List(ctx, fullConfigList); err != nil {
		r.Log.Error(err, "Failed to List FullConfig")
		return ctrl.Result{}, err
	}

	// find the FullConfig corresponding to the pod
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
		if err := r.Client.Update(ctx, matchedFullConfig); err != nil {
			r.Log.Error(err, "Failed to update FullConfig to trigger reconciliation")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

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
