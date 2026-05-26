package controller

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	finv1alpha1 "github.com/mizcausevic-dev/llm-cost-budget-operator/api/v1alpha1"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := finv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func budgetCR() *finv1alpha1.LLMCostBudget {
	return &finv1alpha1.LLMCostBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "team-budget", Namespace: "ns1"},
		Spec: finv1alpha1.LLMCostBudgetSpec{
			MonthlyLimitUSD:      "1000",
			WarnThresholdPercent: 80,
			CostSource:           finv1alpha1.CostSource{ConfigMapName: "llm-cost", Key: "total_usd"},
		},
	}
}

func costCM(value string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "llm-cost", Namespace: "ns1"},
		Data:       map[string]string{"total_usd": value},
	}
}

func reconcilerFor(t *testing.T, objs ...client.Object) (*LLMCostBudgetReconciler, client.Client) {
	t.Helper()
	s := newScheme(t)
	statusObjs := []client.Object{}
	for _, o := range objs {
		if _, ok := o.(*finv1alpha1.LLMCostBudget); ok {
			statusObjs = append(statusObjs, o)
		}
	}
	cl := fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).WithStatusSubresource(statusObjs...).Build()
	return &LLMCostBudgetReconciler{Client: cl, Scheme: s, Recorder: record.NewFakeRecorder(10)}, cl
}

func req() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Name: "team-budget", Namespace: "ns1"}}
}

func get(t *testing.T, cl client.Client) finv1alpha1.LLMCostBudget {
	t.Helper()
	var b finv1alpha1.LLMCostBudget
	if err := cl.Get(context.Background(), req().NamespacedName, &b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestReconcile_UnderBudget(t *testing.T) {
	r, cl := reconcilerFor(t, budgetCR(), costCM("500"))
	if _, err := r.Reconcile(context.Background(), req()); err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	b := get(t, cl)
	if b.Status.State != finv1alpha1.StateUnderBudget || b.Status.PercentUsed != 50 {
		t.Fatalf("want UnderBudget/50, got %s/%d", b.Status.State, b.Status.PercentUsed)
	}
	if b.Status.ObservedSpendUSD != "500.00" {
		t.Fatalf("want spend 500.00, got %s", b.Status.ObservedSpendUSD)
	}
}

func TestReconcile_OverBudget(t *testing.T) {
	r, cl := reconcilerFor(t, budgetCR(), costCM("1200"))
	if _, err := r.Reconcile(context.Background(), req()); err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	b := get(t, cl)
	if b.Status.State != finv1alpha1.StateOverBudget {
		t.Fatalf("want OverBudget, got %s", b.Status.State)
	}
	if c := meta.FindStatusCondition(b.Status.Conditions, "OverBudget"); c == nil || c.Status != metav1.ConditionTrue {
		t.Fatalf("expected OverBudget=True, got %v", b.Status.Conditions)
	}
}

func TestReconcile_MissingConfigMap(t *testing.T) {
	r, cl := reconcilerFor(t, budgetCR()) // no ConfigMap
	if _, err := r.Reconcile(context.Background(), req()); err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	b := get(t, cl)
	if b.Status.State != finv1alpha1.StateUnknown {
		t.Fatalf("want Unknown, got %s", b.Status.State)
	}
	if c := meta.FindStatusCondition(b.Status.Conditions, "BudgetEvaluated"); c == nil || c.Reason != "CostSourceMissing" {
		t.Fatalf("expected CostSourceMissing reason, got %v", b.Status.Conditions)
	}
}

func TestReconcile_MissingKey(t *testing.T) {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "llm-cost", Namespace: "ns1"}, Data: map[string]string{"other": "1"}}
	r, cl := reconcilerFor(t, budgetCR(), cm)
	if _, err := r.Reconcile(context.Background(), req()); err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	if get(t, cl).Status.State != finv1alpha1.StateUnknown {
		t.Fatal("want Unknown for missing key")
	}
}

func TestReconcile_BadCostData(t *testing.T) {
	r, cl := reconcilerFor(t, budgetCR(), costCM("not-a-number"))
	if _, err := r.Reconcile(context.Background(), req()); err != nil {
		t.Fatalf("reconcile error: %v", err)
	}
	b := get(t, cl)
	if c := meta.FindStatusCondition(b.Status.Conditions, "BudgetEvaluated"); c == nil || c.Reason != "InvalidCostData" {
		t.Fatalf("expected InvalidCostData, got %v", b.Status.Conditions)
	}
}

func TestBudgetsForConfigMap(t *testing.T) {
	r, _ := reconcilerFor(t, budgetCR(), costCM("100"))
	reqs := r.budgetsForConfigMap(context.Background(), costCM("100"))
	if len(reqs) != 1 || reqs[0].Name != "team-budget" {
		t.Fatalf("expected mapping to team-budget, got %v", reqs)
	}
	// A ConfigMap nobody references maps to nothing.
	other := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "unrelated", Namespace: "ns1"}}
	if reqs := r.budgetsForConfigMap(context.Background(), other); len(reqs) != 0 {
		t.Fatalf("expected no mapping, got %v", reqs)
	}
}

func TestReconcile_NotFound(t *testing.T) {
	s := newScheme(t)
	cl := fake.NewClientBuilder().WithScheme(s).Build()
	r := &LLMCostBudgetReconciler{Client: cl, Scheme: s, Recorder: record.NewFakeRecorder(1)}
	if _, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "ghost", Namespace: "ns1"}}); err != nil {
		t.Fatalf("missing object should be a no-op, got %v", err)
	}
}
