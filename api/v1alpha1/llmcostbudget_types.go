package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CostSource points at the ConfigMap key holding the accumulated spend (USD)
// for the budget window — e.g. written by an llm-cost-span-exporter pipeline.
type CostSource struct {
	// +kubebuilder:validation:Required
	ConfigMapName string `json:"configMapName"`
	// Key in the ConfigMap whose value is the accumulated USD spend as a number string.
	// +kubebuilder:validation:Required
	Key string `json:"key"`
}

// LLMCostBudgetSpec declares a spend limit and where to read observed spend.
type LLMCostBudgetSpec struct {
	// MonthlyLimitUSD is the budget ceiling as a decimal string, e.g. "1500.00".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?$`
	MonthlyLimitUSD string `json:"monthlyLimitUSD"`

	// CostSource is the ConfigMap key holding observed spend.
	// +kubebuilder:validation:Required
	CostSource CostSource `json:"costSource"`

	// WarnThresholdPercent triggers the Warning state at/above this % of budget.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:default=80
	// +optional
	WarnThresholdPercent int `json:"warnThresholdPercent,omitempty"`
}

// BudgetState is the coarse budget posture.
// +kubebuilder:validation:Enum=UnderBudget;Warning;OverBudget;Unknown
type BudgetState string

const (
	StateUnderBudget BudgetState = "UnderBudget"
	StateWarning     BudgetState = "Warning"
	StateOverBudget  BudgetState = "OverBudget"
	StateUnknown     BudgetState = "Unknown"
)

// LLMCostBudgetStatus reports the observed budget posture.
type LLMCostBudgetStatus struct {
	// +optional
	ObservedSpendUSD string `json:"observedSpendUSD,omitempty"`
	// +optional
	PercentUsed int `json:"percentUsed,omitempty"`
	// +optional
	State BudgetState `json:"state,omitempty"`
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=budget
// +kubebuilder:printcolumn:name="Limit",type=string,JSONPath=`.spec.monthlyLimitUSD`
// +kubebuilder:printcolumn:name="Spend",type=string,JSONPath=`.status.observedSpendUSD`
// +kubebuilder:printcolumn:name="Used%",type=integer,JSONPath=`.status.percentUsed`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`

// LLMCostBudget is a declarative monthly USD budget for LLM spend.
type LLMCostBudget struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LLMCostBudgetSpec   `json:"spec,omitempty"`
	Status LLMCostBudgetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LLMCostBudgetList is a list of LLMCostBudget resources.
type LLMCostBudgetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LLMCostBudget `json:"items"`
}
