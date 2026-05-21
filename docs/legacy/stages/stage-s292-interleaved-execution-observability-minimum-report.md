# Stage S292: Interleaved Execution Observability Minimum — Report

## Objective

Add minimum useful observability to the squeeze breakout vertical slice, providing operational visibility across signal, decision, strategy, risk, and execution paper layers without opening a parallel observability platform.

## Executive Summary

S292 delivers domain-aware counters at every publisher actor in the derive pipeline. The counters follow a `{layer}:{type}:{outcome}` naming convention, are surfaced through existing `/statusz` and `/diagz` HTTP endpoints, and require zero new dependencies. The squeeze breakout slice is now operationally observable end-to-end.

## Observability Added

### Instrumentation Approach

- **Location**: Publisher actor level (SignalPublisher, DecisionPublisher, StrategyPublisher, RiskPublisher, ExecutionPublisher)
- **Mechanism**: `healthz.Tracker.Counter()` — existing thread-safe atomic counters
- **Exposure**: `/statusz` and `/diagz` JSON endpoints on the derive health server
- **Dependencies added**: None

### Counter Inventory

| Layer | Counter Pattern | Variants |
|-------|----------------|----------|
| Signal | `signal:{type}` | bollinger, rsi, ema_crossover |
| Decision | `decision:{type}:{outcome}` | triggered, not_triggered |
| Strategy | `strategy:{type}:{direction}` | long, short, flat |
| Risk | `risk:{type}:{disposition}` | approved, modified, rejected |
| Execution | `execution:{type}:{side}` | buy, sell, none |
| Execution | `execution:{type}:{status}` | submitted, filled |
| Execution | `execution:gate_halted` | (flat counter) |

Total: 6 counter patterns covering the full squeeze path from signal to fill.

## Files Changed

| File | Change |
|------|--------|
| `internal/actors/scopes/derive/signal_publisher_actor.go` | Added `signal:{type}` counter on publish |
| `internal/actors/scopes/derive/decision_publisher_actor.go` | Added `decision:{type}:{outcome}` counter on publish |
| `internal/actors/scopes/derive/strategy_publisher_actor.go` | Added `strategy:{type}:{direction}` counter on publish |
| `internal/actors/scopes/derive/risk_publisher_actor.go` | Added `risk:{type}:{disposition}` counter on publish |
| `internal/actors/scopes/derive/execution_publisher_actor.go` | Added `execution:{type}:{side}`, `execution:{type}:{status}`, `execution:gate_halted` counters |
| `internal/shared/healthz/observability_counters_test.go` | New test: counter naming semantics + /statusz visibility |
| `docs/architecture/interleaved-execution-observability-minimum.md` | Architecture doc: design, scope, limits |
| `docs/architecture/squeeze-slice-metrics-semantics-and-usage.md` | Reference doc: counter semantics, operational usage, jq recipes |

## Validation

- All 8 binaries build clean (`make build`)
- 16/16 healthz tests pass including 2 new observability tests
- `TestTracker_DomainCounterSemantics` — validates counter naming convention and value accuracy
- `TestTracker_CountersVisibleInStatusz` — validates counters appear in /statusz JSON response

## Metrics and Signals Delivered

### Squeeze Path Counters (Primary Deliverable)
- `signal:bollinger` — bollinger signals generated
- `decision:bollinger_squeeze:triggered` — squeeze conditions detected
- `decision:bollinger_squeeze:not_triggered` — no squeeze
- `strategy:squeeze_breakout_entry:long` — long entry resolutions
- `strategy:squeeze_breakout_entry:flat` — flat/no-trade resolutions
- `risk:position_exposure:{approved,modified,rejected}` — position exposure dispositions
- `risk:drawdown_limit:{approved,modified,rejected}` — drawdown limit dispositions
- `execution:paper_order:{buy,sell,none}` — paper order sides
- `execution:paper_order:{submitted,filled}` — paper order statuses
- `execution:gate_halted` — control gate blocks

### Cross-Slice Counters (Bonus: Applies to All Families)
The counter pattern applies to all publisher actors, not just squeeze. RSI, EMA crossover, mean reversion, and trend following paths also gain observability counters through the same mechanism.

## Explicit Limits

1. **No latency tracking** — counters are event counts only, not timing histograms
2. **No per-symbol breakdown** — type-level granularity only (except existing `published:<symbol>`)
3. **No time-series persistence** — counters reset on process restart
4. **No cross-binary correlation** — derive counters only; writer/store not instrumented
5. **No alerting** — no thresholds, no paging, no dashboard integration
6. **No Prometheus** — stays with custom healthz tracker, no new dependencies

## Acceptance Criteria Verification

| Criterion | Status |
|-----------|--------|
| Slice has minimum useful observability | Met — all 5 layers instrumented |
| Improves operational reading without new wave | Met — zero new infrastructure |
| Metrics have clear semantics | Met — documented in metrics reference |
| Reinforces robustness without scope inflation | Met — 5 single-line changes in publisher actors |

## Preparation for S293

Potential next directions after S292:

1. **Smoke script for observability** — extend `scripts/smoke-os-process-operational.sh` to query `/statusz` and validate counter presence after a live run
2. **Writer pipeline counters** — add similar domain-aware counters to the writer inserter for ClickHouse materialization visibility
3. **Cross-binary correlation** — explore NATS stream consumer lag as a proxy for derive→writer pipeline health
4. **Counter reset endpoint** — optional `/resetcounters` for operational testing cycles
5. **Signal evolution families (MACD, VWAP, ATR)** — counters will automatically apply when these families are wired through publisher actors
