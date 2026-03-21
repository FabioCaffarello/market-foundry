# Execution Intent Boundary and Safety Semantics

Stage: S265
Charter: PAPER-EXECUTION-WAVE-1
Date: 2026-03-21
Status: Active

---

## 1. Purpose

This document defines the boundary semantics for execution intents in paper mode: when they are produced, what safety checks apply, and how the system decides to act or not act.

## 2. Execution Intent Lifecycle in Paper Mode

```
Risk Assessment (from risk evaluator actor)
  │
  ▼
riskAssessedMessage (primitive fields, domain-isolated)
  │
  ▼
PaperOrderEvaluator.Evaluate()
  ├── disposition=rejected OR direction=flat → Side=none, Quantity=0
  └── disposition=approved/modified + direction=long/short → Side=buy/sell, Quantity=MaxPositionPct
  │
  ▼
ExecutionIntent (Status=submitted, RiskInput populated, Parameters populated)
  │
  ▼
Actor sets CorrelationID + CausationID (causal trace)
  │
  ▼
PaperFillSimulator.SimulateFill()
  ├── Side=none → pass through unchanged (no fill)
  └── Side=buy/sell → Status=filled, FilledQuantity=Quantity, Fills=[simulated fill record]
  │
  ▼
ExecutionIntent.Validate() → domain validation gate
  │
  ▼
PaperOrderSubmittedEvent (event metadata + ExecutionIntent)
  │
  ▼
ExecutionPublisherActor
  ├── ControlGate check: halted? → BLOCK (log + increment halted counter)
  └── Active → publish to EXECUTION_EVENTS stream
```

## 3. When Execution Intent Exists

An execution intent is **always** produced when a risk assessment completes. The intent may be actionable or not:

| Intent type | Side | Status after fill simulation | Action |
|-------------|------|------|--------|
| Actionable (buy) | `buy` | `filled` | Published with simulated fill record |
| Actionable (sell) | `sell` | `filled` | Published with simulated fill record |
| No-action (rejected) | `none` | `submitted` (unchanged) | Published as-is — records the decision not to act |
| No-action (flat) | `none` | `submitted` (unchanged) | Published as-is — records the decision not to act |

No-action intents are explicitly published to maintain a complete audit trail. Every risk assessment produces exactly one execution event.

## 4. When Execution is Blocked

Execution can be blocked at two points:

### 4.1 Safety Gate (pre-actor, pre-publish)

The `SafetyGate` applies two checks before publishing:

| Gate | Check | Failure mode | Recovery |
|------|-------|-------------|----------|
| Kill switch | `ControlGate.IsHalted()` | Returns `SafetyVerdict{Allowed: false, Reason: "kill_switch"}` | Manual re-activation via KV store |
| Staleness | `StalenessGuard.IsStale(intentTimestamp, now)` | Returns `SafetyVerdict{Allowed: false, Reason: "stale"}` | Automatic — next fresh intent will pass |

**Fail-open behavior**: If the kill switch KV store is unavailable (connection error), the gate **fails open** — execution continues with a warning log. This is intentional: paper mode prioritizes liveness over blocking on infrastructure failures.

### 4.2 Publisher Gate (at publish time)

The `ExecutionPublisherActor` independently checks the control gate:

| Condition | Action |
|-----------|--------|
| Control store unavailable | Warn log; publish proceeds (fail-open) |
| Gate status = `halted` | Block publish; increment `halted` counter; log warning |
| Gate status = `active` | Publish to NATS stream |
| Publish fails (transient) | Single retry after 500ms |
| Publish fails (non-retryable) | Increment `errors` counter; log error; drop |

## 5. Fields That Survive the Full Chain

The following fields are verified to survive from risk assessment through execution fill event:

| Field | Source | Survives to ExecutionIntent | Survives to Event |
|-------|--------|---------------------------|-------------------|
| Risk type | `assessment.Type` | `Risk.Type` + `Parameters["risk_type"]` | Yes |
| Risk disposition | `assessment.Disposition` | `Risk.Disposition` + `Parameters["risk_disposition"]` | Yes |
| Risk confidence | `assessment.Confidence` | `Risk.Confidence` | Yes |
| Strategy type | `assessment.Strategies[0].Type` | `Risk.StrategyType` + `Parameters["strategy_type"]` | Yes (S265) |
| Strategy direction | `assessment.Strategies[0].Direction` | `Parameters["strategy_direction"]` | Yes |
| Strategy confidence | `assessment.Strategies[0].Confidence` | `Parameters["strategy_confidence"]` | Yes |
| Decision severity | `assessment.Strategies[0].DecisionSeverity` | `Risk.DecisionSeverity` + `Parameters["decision_severity"]` | Yes (S265) |
| Constraint value | `assessment.Constraints.*` | `Quantity` + `Parameters["max_position_pct"]` | Yes |
| Correlation ID | event metadata chain | `CorrelationID` | `Metadata.CorrelationID` |
| Causation ID | risk event ID | `CausationID` | `Metadata.CausationID` |
| Timestamp | `assessment.Timestamp` | `Timestamp` | Yes |

## 6. Safety Semantics Summary

### What safety gates protect against

| Threat | Gate | Mechanism |
|--------|------|-----------|
| Stale data driving execution | `StalenessGuard` | Rejects intents older than `maxAge` |
| Emergency stop needed | `ControlGate` (kill switch) | Global halt via KV bucket; blocks all execution publishing |
| Invalid intent | `ExecutionIntent.Validate()` | Domain validation before publishing — rejects missing/invalid fields |
| Duplicate events | JetStream dedup | `DeduplicationKey()` prevents duplicate processing downstream |
| Out-of-order events | KV monotonicity | `ExecutionProjectionActor` skips events older than latest in KV |

### What safety gates do NOT protect against

| Concern | Why not addressed | Future consideration |
|---------|-------------------|---------------------|
| Position limit enforcement | Risk domain already constrains position size | Real venue wave may add position tracking |
| Concurrent conflicting intents | Paper mode is instant; no race conditions | Real venue wave needs ordering guarantees |
| Real money exposure | Paper mode only — all fills are simulated | Venue real wave has its own safety charter |

## 7. Semantic Boundaries

### Execution owns

- Side determination from disposition + direction (pure mapping, no domain logic)
- Fill simulation (instant, simulated=true, price=0, fee=0)
- Safety gate orchestration (kill switch + staleness)
- Event publishing with retry and gate checks
- KV materialization with monotonicity

### Execution does NOT own

- Position sizing (comes from risk `MaxPositionSize` / `MaxExposure`)
- Confidence values (comes from risk, already scaled by strategy type and severity)
- Whether to trade (comes from risk `Disposition`)
- Which direction to trade (comes from strategy `Direction`)
- Any behavioral scaling (all scaling happens in strategy/risk domains)

This boundary is the **anti-corruption layer** between domain intelligence and operational execution. Execution is a thin translator, not a decision-maker.
