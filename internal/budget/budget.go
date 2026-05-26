// Package budget holds the pure (cluster-free) budget evaluation logic.
package budget

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	finv1alpha1 "github.com/mizcausevic-dev/llm-cost-budget-operator/api/v1alpha1"
)

// DefaultWarnThresholdPercent is used when the spec leaves it unset.
const DefaultWarnThresholdPercent = 80

// Result is the outcome of evaluating a budget against observed spend.
type Result struct {
	LimitUSD    float64
	SpendUSD    float64
	PercentUsed int
	State       finv1alpha1.BudgetState
}

func parseUSD(s string) (float64, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, fmt.Errorf("not a number: %q", s)
	}
	if v < 0 {
		return 0, fmt.Errorf("must not be negative: %q", s)
	}
	return v, nil
}

// Evaluate computes the budget posture for a spec given the observed spend
// string (e.g. read from a ConfigMap). It returns an error when either value
// fails to parse or the limit is not positive.
func Evaluate(spec finv1alpha1.LLMCostBudgetSpec, observedSpend string) (Result, error) {
	limit, err := parseUSD(spec.MonthlyLimitUSD)
	if err != nil {
		return Result{State: finv1alpha1.StateUnknown}, fmt.Errorf("monthlyLimitUSD %w", err)
	}
	if limit == 0 {
		return Result{State: finv1alpha1.StateUnknown}, fmt.Errorf("monthlyLimitUSD must be greater than zero")
	}
	spend, err := parseUSD(observedSpend)
	if err != nil {
		return Result{State: finv1alpha1.StateUnknown, LimitUSD: limit}, fmt.Errorf("observed spend %w", err)
	}

	warn := spec.WarnThresholdPercent
	if warn == 0 {
		warn = DefaultWarnThresholdPercent
	}

	percent := int(math.Round((spend / limit) * 100))
	state := finv1alpha1.StateUnderBudget
	switch {
	case percent >= 100:
		state = finv1alpha1.StateOverBudget
	case percent >= warn:
		state = finv1alpha1.StateWarning
	}

	return Result{LimitUSD: limit, SpendUSD: spend, PercentUsed: percent, State: state}, nil
}

// FormatUSD renders a USD amount with two decimals for status reporting.
func FormatUSD(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}
