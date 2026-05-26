package controller

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	finv1alpha1 "github.com/mizcausevic-dev/llm-cost-budget-operator/api/v1alpha1"
	"github.com/mizcausevic-dev/llm-cost-budget-operator/internal/budget"
)

// LLMCostBudgetReconciler evaluates LLMCostBudget objects against an observed
// cost ConfigMap and reports budget posture.
type LLMCostBudgetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=finops.kineticgain.com,resources=llmcostbudgets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=finops.kineticgain.com,resources=llmcostbudgets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

const requeueInterval = time.Hour

// Reconcile reads the cost source and updates the budget status.
func (r *LLMCostBudgetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	var b finv1alpha1.LLMCostBudget
	if err := r.Get(ctx, req.NamespacedName, &b); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var cm corev1.ConfigMap
	cmKey := types.NamespacedName{Name: b.Spec.CostSource.ConfigMapName, Namespace: b.Namespace}
	if err := r.Get(ctx, cmKey, &cm); err != nil {
		if apierrors.IsNotFound(err) {
			return r.markUnknown(ctx, &b, "CostSourceMissing",
				"cost ConfigMap "+b.Spec.CostSource.ConfigMapName+" not found")
		}
		return ctrl.Result{}, err
	}
	spend, ok := cm.Data[b.Spec.CostSource.Key]
	if !ok {
		return r.markUnknown(ctx, &b, "CostKeyMissing",
			"key "+b.Spec.CostSource.Key+" absent from ConfigMap "+cm.Name)
	}

	res, err := budget.Evaluate(b.Spec, spend)
	if err != nil {
		return r.markUnknown(ctx, &b, "InvalidCostData", err.Error())
	}

	b.Status.ObservedSpendUSD = budget.FormatUSD(res.SpendUSD)
	b.Status.PercentUsed = res.PercentUsed
	b.Status.State = res.State
	meta.SetStatusCondition(&b.Status.Conditions, metav1.Condition{
		Type:    "BudgetEvaluated",
		Status:  metav1.ConditionTrue,
		Reason:  string(res.State),
		Message: budget.FormatUSD(res.SpendUSD) + " of " + budget.FormatUSD(res.LimitUSD) + " USD used",
	})
	overStatus := metav1.ConditionFalse
	if res.State == finv1alpha1.StateOverBudget {
		overStatus = metav1.ConditionTrue
	}
	meta.SetStatusCondition(&b.Status.Conditions, metav1.Condition{
		Type:    "OverBudget",
		Status:  overStatus,
		Reason:  string(res.State),
		Message: "budget usage at " + budget.FormatUSD(float64(res.PercentUsed)) + "%",
	})

	if r.Recorder != nil {
		switch res.State {
		case finv1alpha1.StateOverBudget:
			r.Recorder.Eventf(&b, corev1.EventTypeWarning, "OverBudget",
				"LLM spend %s exceeds budget %s", b.Status.ObservedSpendUSD, b.Spec.MonthlyLimitUSD)
		case finv1alpha1.StateWarning:
			r.Recorder.Eventf(&b, corev1.EventTypeWarning, "BudgetWarning",
				"LLM spend at %d%% of budget", res.PercentUsed)
		}
	}

	log.Info("evaluated budget", "state", res.State, "percent", res.PercentUsed)
	if err := r.Status().Update(ctx, &b); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

func (r *LLMCostBudgetReconciler) markUnknown(ctx context.Context, b *finv1alpha1.LLMCostBudget, reason, msg string) (ctrl.Result, error) {
	b.Status.State = finv1alpha1.StateUnknown
	meta.SetStatusCondition(&b.Status.Conditions, metav1.Condition{
		Type:    "BudgetEvaluated",
		Status:  metav1.ConditionFalse,
		Reason:  reason,
		Message: msg,
	})
	if err := r.Status().Update(ctx, b); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// budgetsForConfigMap maps a changed ConfigMap to the budgets that reference it.
func (r *LLMCostBudgetReconciler) budgetsForConfigMap(ctx context.Context, obj client.Object) []reconcile.Request {
	var list finv1alpha1.LLMCostBudgetList
	if err := r.List(ctx, &list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		if list.Items[i].Spec.CostSource.ConfigMapName == obj.GetName() {
			reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      list.Items[i].Name,
				Namespace: list.Items[i].Namespace,
			}})
		}
	}
	return reqs
}

// SetupWithManager wires the reconciler and the ConfigMap watch.
func (r *LLMCostBudgetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&finv1alpha1.LLMCostBudget{}).
		Watches(&corev1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(r.budgetsForConfigMap)).
		Complete(r)
}
