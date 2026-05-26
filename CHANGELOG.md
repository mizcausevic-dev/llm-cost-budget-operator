# Changelog

## v0.1.0 — 2026-05-25

- Initial release: Kubernetes operator enforcing declarative monthly USD budgets for LLM spend.
- `LLMCostBudget` CRD (group `finops.kineticgain.com/v1alpha1`) with a limit, warn threshold, and a ConfigMap cost source; status subresource + printer columns (Limit / Spend / Used% / State).
- Reconciler reads the cost ConfigMap, evaluates posture (`UnderBudget` / `Warning` / `OverBudget`), sets conditions, and emits Warning events; missing/invalid cost data resolves to `Unknown` rather than a wrong number. Watches the referenced ConfigMap so spend changes re-evaluate immediately.
- Pairs with `llm-cost-span-exporter` (its cost summary feeds the ConfigMap).
- Pure `internal/budget` evaluator + fake-client reconciler tests — full `go test ./...` runs without a cluster.
- Helm chart (CRD, read-only-ConfigMap RBAC, non-root manager), raw `config/` manifests, distroless Dockerfile.
- CI: `go vet` / `go test` / `go build` + `helm lint`. AGPL-3.0-or-later, Dependabot (gomod / actions / docker).
