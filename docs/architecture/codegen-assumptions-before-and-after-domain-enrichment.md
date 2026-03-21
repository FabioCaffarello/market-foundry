# Codegen Assumptions: Before and After Domain Enrichment

**Stage:** S259
**Charter:** codegen-reentry-charter-and-scope-freeze.md (S258)
**Date:** 2026-03-21

---

## 1. Purpose

This document makes explicit the codegen's assumptions before and after the breadth wave (S241–S244) and behavioral wave (S249–S257), clarifying what the codegen captures, what it doesn't, and where the boundaries lie.

---

## 2. Before/After: Domain Model

### Decision Domain

| Aspect | Before (pre-S234) | After (post-S257) | Codegen impact |
|---|---|---|---|
| Table columns | outcome, confidence, signals, metadata | outcome, confidence, **severity, rationale**, signals, metadata | Spec `columns` field updated |
| Severity enum | Not present | high, moderate, low | None — codegen doesn't know enum values |
| Rationale field | Not present | Free-form string | None — codegen inserts column, doesn't populate |
| Decision types | 1 (rsi_oversold) | 2 (rsi_oversold, ema_crossover) | Second spec file already existed |
| Evaluator behavior | Simple threshold | Severity-aware with rationale | None — evaluator logic is human-authored |

### Strategy Domain

| Aspect | Before (pre-S241) | After (post-S257) | Codegen impact |
|---|---|---|---|
| Table columns | direction, confidence, decisions, parameters, metadata | Same | None — unchanged |
| Strategy types | 0 | 2 (mean_reversion_entry, trend_following_entry) | Spec files created during breadth wave |
| DecisionInput JSON | type, outcome | type, outcome, **confidence, severity, rationale, timeframe** | None — JSON schema is inside column, not in codegen |
| Severity scaling | Not present | ScaleConfidence, AdjustParam per severity | None — scaling logic is human-authored |
| Confidence factors | Not present | Per-severity multiplier maps | None — maps are in application code |

### Risk Domain

| Aspect | Before (pre-S241) | After (post-S257) | Codegen impact |
|---|---|---|---|
| Table columns | disposition, confidence, strategies, constraints, rationale, parameters, metadata | Same | None — unchanged |
| Risk types | 0 | 2 (position_exposure, drawdown_limit) | Spec files created during breadth wave |
| StrategyInput JSON | type, direction, confidence | type, direction, confidence, **decision severity, decision rationale** | None — JSON schema inside column |
| Dual fan-out | Not present | Both risk families evaluate each strategy | None — fan-out is in actor wiring |
| Strategy-type scaling | Not present | Counter-trend vs pro-trend confidence factors | None — scaling is in risk_scaling.go |
| Severity-aware tolerance | Not present | Position/drawdown tolerance multiplied by severity | None — tolerance maps in application code |

### Execution Domain

| Aspect | Before (pre-S241) | After (post-S257) | Codegen impact |
|---|---|---|---|
| Table columns | side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id | Same | None — unchanged |
| Venue family | Not present | venue_market_order (separate stream) | None — NOT in codegen scope |

### Signal Domain

| Aspect | Before | After | Codegen impact |
|---|---|---|---|
| Table columns | type, source, symbol, timeframe, value, metadata | Same | None — unchanged |
| Signal types | 2 (rsi, ema) | Same | Already integrated |

---

## 3. Before/After: Codegen Assumptions

### 3.1 What the codegen assumed before breadth/behavioral waves

| Assumption | Status |
|---|---|
| Each family has exactly 1 spec YAML file | Still valid |
| Each family maps to exactly 1 NATS consumer + 1 pipeline entry | Still valid |
| Column lists are stable per table | **Updated** — decision columns now include severity, rationale |
| Each layer has a uniform column set | Still valid — all families in a layer share the same table |
| Templates are column-agnostic (columns is a string) | Still valid — design proved resilient |
| NATS subjects follow `{layer}.events.{family}.{verb}.>` pattern | Still valid |
| Durable consumers follow `writer-{layer}-{family}` pattern | Still valid |
| Each layer has one shared Starter function | Still valid |

### 3.2 What the codegen correctly does NOT assume

| Non-assumption | Why it matters |
|---|---|
| No knowledge of severity values | Severity enum (high/moderate/low) is a domain decision |
| No knowledge of JSON payload schemas | DecisionInput, StrategyInput internals evolve without codegen involvement |
| No knowledge of confidence scaling | Scaling factors are behavioral rules, not mechanical wiring |
| No knowledge of inter-layer context flow | Decision→strategy→risk cascading is application logic |
| No knowledge of fan-out topology | Dual-risk evaluation is actor wiring, not pipeline config |
| No knowledge of rejection criteria | Confidence ≤ 0 rejection is domain logic |
| No knowledge of evaluator/resolver internals | Business logic is human-authored |

### 3.3 Assumptions that were tested and held

| Assumption | Test | Result |
|---|---|---|
| `writer.columns` as string absorbs schema changes | Added severity/rationale to decision specs | HELD — template uses `{{.Derived.InsertSQL}}` |
| `DerivedFields` naming is stable across families | Added 6 families in new layers | HELD — all derived names are correct |
| Cross-spec validation catches collisions | 10 families × 3 uniqueness checks | HELD — no false positives or missed duplicates |
| Golden comparison is normalization-safe | Templates generate expanded form, manual code uses factory | HELD — comparison normalizes whitespace/comments |

---

## 4. Codegen Knowledge Boundary

The following diagram shows exactly what the codegen knows (left) versus what it doesn't and shouldn't (right):

```
┌─────────────────────────────────────────┐  ┌──────────────────────────────────────────────┐
│         CODEGEN KNOWS (spec.yaml)       │  │         CODEGEN DOES NOT KNOW                │
│                                         │  │                                              │
│  • Family name (rsi_oversold)           │  │  • What severity values mean                 │
│  • Layer assignment (decision)          │  │  • How confidence is computed                │
│  • NATS subject pattern                 │  │  • How severity scaling works                │
│  • NATS event type string               │  │  • What's inside JSON columns                │
│  • NATS stream name                     │  │  • How evaluators make decisions             │
│  • Durable consumer name                │  │  • How resolvers compute strategy            │
│  • Target ClickHouse table              │  │  • How risk assesses position/drawdown       │
│  • Column list (as string)              │  │  • Actor lifecycle and message routing        │
│  • Mapper function name                 │  │  • Supervisor registry wiring                │
│  • Config array field name              │  │  • Inter-layer context flow                  │
│  • Domain event package/type            │  │  • Dual-risk fan-out topology                │
│  • Tier (1 or 2)                        │  │  • Degenerate case handling                  │
│                                         │  │  • Behavioral test invariants                │
│  Outputs:                               │  │  • ClickHouse schema design rationale        │
│  • Consumer spec function               │  │  • Event payload structure                   │
│  • Pipeline entry struct literal        │  │  • Venue family execution flow               │
└─────────────────────────────────────────┘  └──────────────────────────────────────────────┘
```

---

## 5. Assumptions That Could Break in Future

These assumptions are currently valid but may need re-evaluation if the domain evolves:

| Assumption | What would break it | Impact |
|---|---|---|
| All families in a layer share one table | A family needing its own table (e.g., venue fills) | Spec schema may need `writer.table` per family (already supported) |
| All families in a layer share one Starter | A family needing custom consumer logic | Would need evidence-layer-style special handling in `DerivedFields` |
| Column list is uniform within a layer | A family needing extra columns vs peers | Would need per-family column override (already supported — `columns` is per spec) |
| Two artifact types are sufficient | Need for actor wrappers, evaluator stubs | Would require new templates — escalates to new charter |
| AckWait and MaxDeliver are universal | A family needing different delivery guarantees | Would need template parameterization (blocked by OD-BW2) |

---

## 6. Domain Enrichments Not Visible to Codegen (by design)

The following enrichments happened during breadth/behavioral waves and are intentionally invisible to codegen:

### 6.1 Severity Scaling (strategy layer)

```go
// internal/application/strategy/severity_scaling.go
func ScaleConfidence(base float64, severity string, factors map[string]float64) float64
func AdjustParam(base float64, severity string, multipliers map[string]float64) float64
```

Each strategy family defines its own severity-to-multiplier maps. This is pure business logic — codegen only knows that the result goes into the `confidence` and `parameters` columns.

### 6.2 Risk Scaling (risk layer)

```go
// internal/application/risk/risk_scaling.go
func lookupFactor(strategyType string, factors map[string]float64) float64
func lookupSeverityFactor(severity string, factors map[string]float64) float64
```

Risk evaluators apply strategy-type-aware and severity-aware multipliers. Codegen only knows that the result goes into `disposition`, `confidence`, `constraints`, and `parameters` columns.

### 6.3 Cascading Context

```
Decision (severity, rationale)
  ↓ carried in DecisionInput JSON
Strategy (decisions JSON contains severity/rationale, confidence scaled by severity)
  ↓ carried in StrategyInput JSON
Risk (strategies JSON contains decision severity, confidence + tolerance adjusted)
```

This cascading context model enriches the JSON content within existing columns. The column structure is unchanged — only the data inside gets richer. Codegen has no visibility into this and should not.

### 6.4 Dual-Risk Fan-Out

Both `position_exposure` and `drawdown_limit` evaluate each strategy event independently. This fan-out is wired in the derive supervisor's actor registration — codegen's pipeline entries are per-family and don't express fan-out topology.

---

## 7. Summary

| Category | Items that changed | Codegen changes needed |
|---|---|---|
| ClickHouse columns | decisions: +severity, +rationale | YAML spec updated (already done) |
| JSON payload schemas | DecisionInput, StrategyInput enriched | None — inside column payload |
| Behavioral logic | Severity scaling, confidence mapping, rejection rules | None — application layer |
| Inter-layer flow | Cascading context, dual-risk fan-out | None — actor wiring |
| NATS configuration | No changes | None |
| Template structure | No changes needed | None |
| Spec schema (spec.go) | No changes needed | None |
| Golden snapshots | Already regenerated with correct columns | None |

**Bottom line:** The codegen's column-agnostic design (`columns` as a free-form string) made it naturally resilient to domain enrichment. The YAML specs were already updated during breadth wave stages. Zero code changes are required for reconciliation.
