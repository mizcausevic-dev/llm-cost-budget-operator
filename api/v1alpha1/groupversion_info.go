// Package v1alpha1 contains the LLMCostBudget API — declarative monthly USD
// budgets for LLM spend, evaluated against an observed-cost source.
// +kubebuilder:object:generate=true
// +groupName=finops.kineticgain.com
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// GroupVersion is the group/version for this API.
var GroupVersion = schema.GroupVersion{Group: "finops.kineticgain.com", Version: "v1alpha1"}

// SchemeBuilder registers the API types with a runtime scheme.
var SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

// AddToScheme adds the API types to a scheme.
var AddToScheme = SchemeBuilder.AddToScheme

func init() {
	SchemeBuilder.Register(&LLMCostBudget{}, &LLMCostBudgetList{})
}
