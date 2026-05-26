# Security Policy

`llm-cost-budget-operator` is a Kubernetes controller. It reads `LLMCostBudget`
resources and the ConfigMaps they reference, and writes status + events. It
makes no outbound network calls and serves only the manager's metrics (`:8080`)
and health (`:8081`) probes.

Operational notes:

- The bundled RBAC is least-privilege and **read-only on ConfigMaps**; the
  operator never mutates your cost data.
- The container runs as non-root, read-only root filesystem, no privilege
  escalation, all capabilities dropped (see chart `values.yaml`).
- Budget figures are only as trustworthy as the cost ConfigMap they read; treat
  them as observability, not billing of record.

## Supported versions

Only the latest tagged release is supported.

## Reporting a vulnerability

Please use GitHub Security Advisories for private disclosure:

- [Open a security advisory](https://github.com/mizcausevic-dev/llm-cost-budget-operator/security/advisories/new)

Do not file public issues for security reports.
