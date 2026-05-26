package budget

import (
	"testing"

	finv1alpha1 "github.com/mizcausevic-dev/llm-cost-budget-operator/api/v1alpha1"
)

func specWith(limit string, warn int) finv1alpha1.LLMCostBudgetSpec {
	return finv1alpha1.LLMCostBudgetSpec{MonthlyLimitUSD: limit, WarnThresholdPercent: warn}
}

func TestEvaluate_States(t *testing.T) {
	cases := []struct {
		name    string
		limit   string
		warn    int
		spend   string
		percent int
		state   finv1alpha1.BudgetState
	}{
		{"under", "1000", 80, "500", 50, finv1alpha1.StateUnderBudget},
		{"at warn", "1000", 80, "800", 80, finv1alpha1.StateWarning},
		{"warn band", "1000", 80, "950", 95, finv1alpha1.StateWarning},
		{"at limit", "1000", 80, "1000", 100, finv1alpha1.StateOverBudget},
		{"over", "1000", 80, "1500", 150, finv1alpha1.StateOverBudget},
		{"default warn (0 -> 80)", "200", 0, "160", 80, finv1alpha1.StateWarning},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, err := Evaluate(specWith(c.limit, c.warn), c.spend)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.PercentUsed != c.percent {
				t.Fatalf("percent: want %d got %d", c.percent, r.PercentUsed)
			}
			if r.State != c.state {
				t.Fatalf("state: want %s got %s", c.state, r.State)
			}
		})
	}
}

func TestEvaluate_Errors(t *testing.T) {
	if _, err := Evaluate(specWith("abc", 80), "10"); err == nil {
		t.Fatal("expected error for non-numeric limit")
	}
	if _, err := Evaluate(specWith("0", 80), "10"); err == nil {
		t.Fatal("expected error for zero limit")
	}
	if _, err := Evaluate(specWith("100", 80), "oops"); err == nil {
		t.Fatal("expected error for non-numeric spend")
	}
	if _, err := Evaluate(specWith("100", 80), "-5"); err == nil {
		t.Fatal("expected error for negative spend")
	}
	r, err := Evaluate(specWith("bad", 80), "10")
	if err == nil || r.State != finv1alpha1.StateUnknown {
		t.Fatalf("expected Unknown state on parse error, got %s", r.State)
	}
}

func TestFormatUSD(t *testing.T) {
	if got := FormatUSD(1234.5); got != "1234.50" {
		t.Fatalf("want 1234.50, got %s", got)
	}
}
