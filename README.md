# llm-cost-budget-operator

A Kubernetes operator that enforces **declarative monthly USD budgets for LLM spend**. You declare a `LLMCostBudget` — a limit and where to read accumulated spend; the operator evaluates it on every change and reports `UnderBudget` / `Warning` / `OverBudget`, raising Kubernetes events when a team blows past its ceiling.

The FinOps control-plane for the GenAI cost lane. It pairs with [`llm-cost-span-exporter`](https://github.com/mizcausevic-dev/llm-cost-span-exporter): the exporter computes cost, a pipeline writes the running total to a ConfigMap, and this operator turns that number into an enforced budget with status and alerts.

## Why

Token spend is easy to measure and hard to govern: by the time finance notices, the month is over. This operator makes the budget a cluster resource. Point it at the ConfigMap your cost pipeline already writes, set a limit and a warn threshold, and budget posture becomes observable (`kubectl get budget`) and alertable (Warning/OverBudget events) — the same way you'd watch any other SLO.

## Custom resource

```yaml
apiVersion: finops.kineticgain.com/v1alpha1
kind: LLMCostBudget
metadata:
  name: team-platform-budget
spec:
  monthlyLimitUSD: "1000.00"
  warnThresholdPercent: 80          # default 80
  costSource:
    configMapName: llm-cost         # written by an llm-cost-span-exporter pipeline
    key: total_usd
```

```
$ kubectl get budget
NAME                    LIMIT     SPEND    USED%   STATE
team-platform-budget    1000.00   842.50   84      Warning
```

The controller watches both the budget and its referenced ConfigMap, so spend updates re-evaluate immediately; a periodic requeue is the safety net. Missing ConfigMap / key / non-numeric data resolve to `Unknown` with a clear `BudgetEvaluated=False` condition rather than a wrong number.

## Install

```bash
helm install budgets charts/llm-cost-budget-operator
kubectl apply -f config/samples/sample.yaml
kubectl get budget
```

The chart installs the CRD, least-privilege RBAC (read-only on ConfigMaps), a non-root ServiceAccount, and the manager Deployment. Build the image from the distroless `Dockerfile`.

## Architecture

- `api/v1alpha1` — the `LLMCostBudget` CRD types.
- `internal/budget` — **pure, cluster-free** evaluation (`Evaluate(spec, observedSpend) → state/percent`), the unit-tested core.
- `internal/controller` — the reconciler: read ConfigMap → evaluate → status + events, with a ConfigMap→budget watch mapping.
- `cmd` — the manager entrypoint.

`go test ./...` runs the full gate with no cluster: the evaluator is unit-tested and the reconciler is exercised end-to-end with controller-runtime's fake client (under/over budget, missing source, bad data, watch mapping).

## License

AGPL-3.0-or-later — see [LICENSE](LICENSE).
