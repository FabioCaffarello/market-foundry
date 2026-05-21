# Codegen Spec Reconciliation with Breadth and Behavior

**Stage:** S259
**Charter:** codegen-reentry-charter-and-scope-freeze.md (S258)
**Date:** 2026-03-21

---

## 1. Purpose

This document records the formal reconciliation of the codegen spec, templates, golden snapshots, and `integrated.yaml` against the current state of the market-foundry domain after the breadth wave (S241–S244) and behavioral wave (S249–S257).

---

## 2. Reconciliation Method

1. Read all 10 family YAML specs and compared `writer.columns` against actual INSERT SQL in `cmd/writer/pipeline.go`.
2. Compared all 20 golden snapshots against the manual code in pipeline.go and registry files.
3. Ran `codegen validate-all` — all 10 specs valid, zero cross-spec collisions.
4. Ran `codegen check-all` — all 20 golden comparisons PASS.
5. Ran `go test ./codegen/...` — all 35 tests pass.
6. Cross-referenced NATS subjects, event types, streams, and durable consumer names against registry files in `internal/adapters/nats/nats*/registry.go`.
7. Cross-referenced column lists against ClickHouse migration files in `deploy/migrations/`.

---

## 3. Reconciliation Results by Layer

### 3.1 Decision Layer (rsi_oversold, ema_crossover)

**Domain enrichment:** Migration 007 (S234) added `severity` and `rationale` columns to `decisions` table.

| Field | Spec value | Pipeline.go INSERT SQL | Match |
|---|---|---|---|
| columns | `...outcome, confidence, severity, rationale, signals, metadata, final, timestamp` | `...outcome, confidence, severity, rationale, signals, metadata, final, timestamp` | YES |

**Golden snapshots:** Pipeline entry snapshots include `severity, rationale` in INSERT SQL. Consumer spec snapshots match NATS configuration.

**Behavioral context:** Decision evaluators now produce severity (high/moderate/low) and rationale strings. The codegen spec captures these as columns in the INSERT statement but has no knowledge of how severity values are computed or what they mean. This is correct — the codegen boundary is at the writer pipeline.

**Verdict:** RECONCILED — no changes needed.

### 3.2 Strategy Layer (mean_reversion_entry, trend_following_entry)

**Domain enrichment:** Strategy resolvers now consume decision severity and apply severity-based confidence scaling and parameter adjustment via `severity_scaling.go`.

| Field | Spec value | Pipeline.go INSERT SQL | Match |
|---|---|---|---|
| columns | `...direction, confidence, decisions, parameters, metadata, final, timestamp` | `...direction, confidence, decisions, parameters, metadata, final, timestamp` | YES |

**Key observation:** The `decisions` column is a JSON array of `DecisionInput` structs that now carry `severity` and `rationale` fields. This enrichment lives inside the JSON payload, not in the column list. The codegen spec correctly captures the column name (`decisions`) without needing to know about the JSON schema inside it.

**Behavioral context:** Strategy confidence is now scaled by decision severity. This scaling logic lives in `internal/application/strategy/severity_scaling.go` and is entirely outside codegen's reach — it affects the JSON content written to the `decisions` and `parameters` columns, not the column structure.

**Verdict:** RECONCILED — no changes needed.

### 3.3 Risk Layer (position_exposure, drawdown_limit)

**Domain enrichment:** Risk evaluators now apply strategy-type-aware and severity-aware scaling via `risk_scaling.go`. Dual-risk fan-out (both position_exposure and drawdown_limit evaluate each strategy) was formalized.

| Field | Spec value | Pipeline.go INSERT SQL | Match |
|---|---|---|---|
| columns | `...disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp` | `...disposition, confidence, strategies, constraints, rationale, parameters, metadata, final, timestamp` | YES |

**Key observation:** The `strategies` column is a JSON array of `StrategyInput` structs that now carry cascaded context from decisions (severity, rationale). The `constraints` column carries `MaxPositionSize`, `MaxExposure`, `StopDistance` as JSON. Both are richer than pre-breadth, but the column structure is unchanged.

**Behavioral context:** Risk evaluators apply different confidence factors for counter-trend vs pro-trend strategies, and severity-aware position/drawdown tolerance multipliers. This logic lives in `risk_scaling.go` and is entirely outside codegen's scope.

**Verdict:** RECONCILED — no changes needed.

### 3.4 Signal Layer (rsi, ema)

**Domain enrichment:** No structural changes in signal layer during breadth/behavioral waves. Signal remains a value + metadata producer.

| Field | Spec value | Pipeline.go INSERT SQL | Match |
|---|---|---|---|
| columns | `...type, source, symbol, timeframe, value, metadata, final, timestamp` | `...type, source, symbol, timeframe, value, metadata, final, timestamp` | YES |

**Already integrated:** RSI and EMA are the only two families with codegen markers in target files (`integrated.yaml` has 4 entries).

**Verdict:** RECONCILED — no changes needed.

### 3.5 Evidence Layer (candle)

**Domain enrichment:** No structural changes.

| Field | Spec value | Pipeline.go INSERT SQL | Match |
|---|---|---|---|
| columns | `...source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final` | `...source, symbol, timeframe, open, high, low, close, volume, trade_count, open_time, close_time, final` | YES |

**Verdict:** RECONCILED — no changes needed.

### 3.6 Execution Layer (paper_order)

**Domain enrichment:** No structural changes to paper_order columns. Venue family (venue_market_order) exists in registry but is NOT in codegen scope.

| Field | Spec value | Pipeline.go INSERT SQL | Match |
|---|---|---|---|
| columns | `...side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp` | `...side, quantity, filled_quantity, status, risk, fills, parameters, metadata, exec_correlation_id, exec_causation_id, final, timestamp` | YES |

**Verdict:** RECONCILED — no changes needed.

---

## 4. Consumer Spec Style Reconciliation

### Observation

The codegen template (`consumer_spec.go.tmpl`) generates expanded struct literals:

```go
func WriterRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
    return natskit.ConsumerSpec{
        Durable: "writer-decision-rsi-oversold",
        Event: natskit.EventSpec{
            Subject: "decision.events.rsi_oversold.evaluated.>",
            Type:    "decision.events.v1.rsi_oversold_evaluated",
            Stream:  natskit.StreamSpec{Name: "DECISION_EVENTS"},
        },
        AckWait:    30 * time.Second,
        MaxDeliver: 5,
    }
}
```

The manual code in registry files uses the factory pattern:

```go
func WriterRSIOversoldDecisionConsumer() natskit.ConsumerSpec {
    return natskit.NewConsumerSpec("writer-decision-rsi-oversold",
        "decision.events.rsi_oversold.evaluated.>",
        "decision.events.v1.rsi_oversold_evaluated", "DECISION_EVENTS")
}
```

### Assessment

These are **semantically equivalent** — `natskit.NewConsumerSpec()` produces the identical struct with AckWait=30s, MaxDeliver=5. The differences are:

| Aspect | Factory form | Expanded form |
|---|---|---|
| Self-documenting | No — positional args | Yes — named fields |
| Diff-friendly | Harder to review | Easier to review |
| Factory dependency | Requires NewConsumerSpec signature stability | Independent |
| Field visibility | Hidden in factory | Explicit |

### Decision

When codegen markers are inserted in S260, the factory-style calls will be replaced with the expanded form. This is a **positive change**: the expanded form is more readable, more diffable, and doesn't depend on factory function signature. No reconciliation action needed — the style difference is intentional and expected.

---

## 5. NATS Subject and Stream Reconciliation

| Family | Spec subject | Registry subject | Match |
|---|---|---|---|
| rsi | signal.events.rsi.generated.> | signal.events.rsi.generated.> | YES |
| ema | signal.events.ema.generated.> | signal.events.ema.generated.> | YES |
| rsi_oversold | decision.events.rsi_oversold.evaluated.> | decision.events.rsi_oversold.evaluated | WILDCARD DIFF |
| ema_crossover | decision.events.ema_crossover.evaluated.> | decision.events.ema_crossover.evaluated | WILDCARD DIFF |
| mean_reversion_entry | strategy.events.mean_reversion_entry.resolved.> | strategy.events.mean_reversion_entry.resolved | WILDCARD DIFF |
| trend_following_entry | strategy.events.trend_following_entry.resolved.> | strategy.events.trend_following_entry.resolved | WILDCARD DIFF |
| position_exposure | risk.events.position_exposure.assessed.> | risk.events.position_exposure.assessed | WILDCARD DIFF |
| drawdown_limit | risk.events.drawdown_limit.assessed.> | risk.events.drawdown_limit.assessed | WILDCARD DIFF |
| paper_order | execution.events.paper_order.submitted.> | execution.events.paper_order.submitted | WILDCARD DIFF |
| candle | evidence.events.candle.sampled.> | evidence.events.candle.sampled | WILDCARD DIFF |

### Wildcard Suffix Explanation

The spec subjects include `>` (NATS multi-level wildcard) for writer consumer subscription — this allows the consumer to receive events regardless of trailing subject tokens (e.g., symbol, timeframe). The registry `EventSpec.Subject` does NOT include `>` because it defines the publish subject (without wildcard).

This is correct behavior: the **writer consumer** needs the wildcard to catch all events, while the **publisher** uses the exact subject. The codegen spec captures the consumer-facing subject (with `>`), which is what the `ConsumerSpec.Event.Subject` field needs. The writer consumer specs in the actual code (e.g., `WriterRSIOversoldDecisionConsumer`) also use the `>` suffix.

**Verdict:** CONSISTENT — no reconciliation needed.

---

## 6. Cross-Spec Validation Results

```
$ codegen validate-all
VALID    candle (layer=evidence, tier=1)
VALID    drawdown_limit (layer=risk, tier=1)
VALID    ema (layer=signal, tier=1)
VALID    ema_crossover (layer=decision, tier=1)
VALID    mean_reversion_entry (layer=strategy, tier=1)
VALID    paper_order (layer=execution, tier=1)
VALID    position_exposure (layer=risk, tier=1)
VALID    rsi (layer=signal, tier=1)
VALID    rsi_oversold (layer=decision, tier=1)
VALID    trend_following_entry (layer=strategy, tier=1)

Cross-spec uniqueness: OK (10 families, no collisions)
```

No duplicate family names, NATS subjects, or durable consumers.

---

## 7. Golden Snapshot Verification

```
$ codegen check-all
PASS  candle/consumer_spec          PASS  candle/pipeline_entry
PASS  drawdown_limit/consumer_spec  PASS  drawdown_limit/pipeline_entry
PASS  ema/consumer_spec             PASS  ema/pipeline_entry
PASS  ema_crossover/consumer_spec   PASS  ema_crossover/pipeline_entry
PASS  mean_reversion_entry/consumer_spec  PASS  mean_reversion_entry/pipeline_entry
PASS  paper_order/consumer_spec     PASS  paper_order/pipeline_entry
PASS  position_exposure/consumer_spec  PASS  position_exposure/pipeline_entry
PASS  rsi/consumer_spec             PASS  rsi/pipeline_entry
PASS  rsi_oversold/consumer_spec    PASS  rsi_oversold/pipeline_entry
PASS  trend_following_entry/consumer_spec  PASS  trend_following_entry/pipeline_entry

20 passed, 0 failed
```

All templates produce output that matches golden snapshots exactly.

---

## 8. Test Suite Verification

```
$ go test ./codegen/... -count=1
ok  codegen  0.218s  (35 tests passed)
```

All codegen tests pass, including cross-family golden comparison (`TestCheckAllFamilies`).

---

## 9. Overall Reconciliation Verdict

**RECONCILED — zero code changes required.**

The codegen spec was designed with `writer.columns` as a free-form string field, which made it naturally resilient to the breadth wave column additions (severity, rationale). The YAML spec files were updated with correct column lists during the breadth wave stages. Golden snapshots were regenerated to reflect those column lists. Templates are column-agnostic and did not need modification.

The domain's richer behavioral semantics (severity scaling, confidence mapping, cascading context, dual-risk fan-out) are expressed through JSON payloads inside existing columns and through application-layer logic — both of which are outside codegen's boundary and should remain so.
