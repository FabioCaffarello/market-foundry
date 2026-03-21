# Composite Execution Read Model Over Five ClickHouse Tables

**Stage:** S296
**Status:** Delivered
**Predecessor:** S295 (Correlation/Causation Spine Validation)

---

## 1. Purpose

This document defines the composite execution read model that unifies the five ClickHouse domain tables into a single, coherent view of an execution's causal chain. The model enables operators to answer "why did execution X happen?" without manual ID correlation.

## 2. The Five Tables

| # | Table | Domain Layer | Key Columns | Role in Chain |
|---|-------|-------------|-------------|---------------|
| 1 | `signals` | Signal | type, value, metadata | Chain root — the market observation that initiated the flow |
| 2 | `decisions` | Decision | outcome, severity, confidence, rationale, signals | First evaluation — did the signal cross a threshold? |
| 3 | `strategies` | Strategy | direction, confidence, decisions, parameters | Resolution — what trade direction and confidence? |
| 4 | `risk_assessments` | Risk | disposition, confidence, constraints, rationale | Gate — approved, modified, or rejected? |
| 5 | `executions` | Execution | side, quantity, status, risk, fills | Terminal — the paper order submitted (or not) |

All five tables share the same causal metadata columns:
- `event_id` — unique identifier for the event
- `correlation_id` — immutable thread identifier, propagated end-to-end
- `causation_id` — parent event's `event_id`, forming a DAG
- `occurred_at` — event envelope timestamp

## 3. Composition Strategy

### Application-Side Composition (No JOINs)

The composite reader uses **five independent queries** by `correlation_id`, assembled in the Go application layer. This approach was chosen over ClickHouse JOINs because:

1. **Simplicity** — each query hits one table, one filter, one row. No complex JOIN plans.
2. **Resilience** — a failed query for one stage does not prevent partial chain assembly.
3. **Alignment** — ClickHouse MergeTree tables are optimized for point lookups on indexed columns; `correlation_id` is stored in every row.
4. **Maintainability** — no materialized views, no CDC, no schema coupling between tables.

### Query Pattern Per Stage

```sql
SELECT event_id, occurred_at, correlation_id, causation_id, <domain_columns>
FROM <table>
WHERE correlation_id = ?
ORDER BY timestamp DESC LIMIT 1
```

Each query returns at most one row (the most recent event for that correlation_id in that table).

### Batch Lookup

For batch mode (by symbol/timeframe/time-range):

1. Query `executions` table for distinct `correlation_id`s matching filters
2. For each correlation_id, reconstruct the full chain via 5 independent queries
3. Return chains ordered by execution timestamp DESC

```sql
SELECT correlation_id
FROM executions
WHERE source = ? AND symbol = ? AND timeframe = ?
  [AND timestamp >= ?] [AND timestamp <= ?]
GROUP BY correlation_id
ORDER BY max(timestamp) DESC
LIMIT ?
```

## 4. Canonical Read Model Shape

```
CompositeExecutionChain
├── correlation_id: string
├── signal: SignalWithTrace (optional)
│   ├── event_id, correlation_id, causation_id, occurred_at
│   └── Signal domain fields (type, value, metadata, ...)
├── decision: DecisionWithTrace (optional)
│   ├── event_id, correlation_id, causation_id, occurred_at
│   └── Decision domain fields (outcome, severity, confidence, rationale, ...)
├── strategy: StrategyWithTrace (optional)
│   ├── event_id, correlation_id, causation_id, occurred_at
│   └── Strategy domain fields (direction, confidence, decisions, parameters, ...)
├── risk: RiskWithTrace (optional)
│   ├── event_id, correlation_id, causation_id, occurred_at
│   └── RiskAssessment domain fields (disposition, confidence, constraints, ...)
├── execution: ExecutionWithTrace (optional)
│   ├── event_id, event_correlation_id, event_causation_id, occurred_at
│   └── ExecutionIntent domain fields (side, quantity, status, risk, fills, ...)
├── stage_count: int (0-5)
├── chain_complete: bool
└── missing_stages: []string
```

Each `*WithTrace` type extends the domain struct with the causal metadata that was previously only available via raw SQL (S295 gap G1).

## 5. Access Modes

| Mode | Input | Starting Table | Use Case |
|------|-------|---------------|----------|
| Single chain | `correlation_id` | All 5 in parallel | "Explain execution X" |
| Batch | `source`, `symbol`, `timeframe`, time range | `executions` first | "Show recent chains for btcusdt" |

## 6. Implementation Files

| File | Layer | Purpose |
|------|-------|---------|
| `internal/adapters/clickhouse/composite_reader.go` | Adapter | 5-table composition + chain assembly |
| `internal/application/analyticalclient/composite_contracts.go` | Contract | `CompositeExecutionChain`, `*WithTrace` types, query/reply |
| `internal/application/analyticalclient/get_composite_chain.go` | Use Case | Validation, mode dispatch, timing |
| `internal/adapters/clickhouse/composite_reader_test.go` | Test | Chain completeness unit tests |
| `internal/application/analyticalclient/get_composite_chain_test.go` | Test | Use case unit tests (9 cases) |
| `internal/adapters/clickhouse/composite_reader_integration_test.go` | Test | Live ClickHouse integration (6 criteria) |

## 7. Relation to Governing Questions

| Question | Answerable? | How |
|----------|-------------|-----|
| Q1: Why was execution X submitted? | Yes | Single chain by correlation_id shows full signal→execution flow |
| Q2: Why was execution X rejected? | Partially | Risk stage shows disposition + constraints + rationale |
| Q3: Which signals contributed to decision D? | Yes | Decision stage includes signals array + causal metadata |
| Q4: Confidence/severity flow? | Yes | Each stage carries confidence; decision carries severity |
| Q5: Why did symbol stop receiving executions? | Deferred to S297/S298 | Requires batch analysis + gap detection |
| Q6: Blocked vs approved count in period T? | Deferred to S298 | Requires aggregation queries |
| Q7: Conversion rate per stage? | Deferred to S298 | Requires funnel metrics |
