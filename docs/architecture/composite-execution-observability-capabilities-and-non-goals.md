# Composite Execution Observability — Capabilities and Non-Goals

**Stage:** S294
**Companion to:** composite-execution-observability-wave-charter-and-scope-freeze.md

---

## 1. Capabilities This Wave Delivers

### Capability 1: Execution Chain Reconstruction

**What:** Given any execution event, reconstruct the full causal chain that produced it.

**Shape of the answer:**
```
Execution (paper_order, buy, filled)
  ← Risk (position_exposure, approved, maxPositionSize=1000)
    ← Strategy (squeeze_breakout_entry, long, confidence=0.82)
      ← Decision (bollinger_squeeze, triggered, severity=high)
        ← Signal (bollinger, bandwidth=0.023, percentB=0.12)
          ← Evidence (candle, close=42150.50)
```

**How it works:** Follow CausationID from execution → risk_assessment → strategy → decision → signal(s). Each link is a ClickHouse lookup on `event_id = parent.causation_id`.

**Validation:** Works across all three proven slices (EMA, Trend, Squeeze) without family-specific logic.

### Capability 2: Rejection and Modification Attribution

**What:** For any rejected or modified execution, surface the specific constraint and evaluator that caused the outcome.

**Shape of the answer:**
```
Execution: REJECTED
  Rejecting evaluator: drawdown_limit
  Disposition: rejected
  Constraint: maxDrawdownPercent=0.02, currentDrawdown=0.025
  Rationale: "drawdown exceeds maximum threshold"
  Upstream strategy: trend_following_entry (long, confidence=0.71)
  Upstream decision: ema_crossover (triggered, severity=moderate)
```

**How it works:** Risk assessment events already carry `disposition`, `constraints` (JSON), and `rationale`. The composite read model extracts and surfaces these fields alongside the upstream chain.

**Validation:** Test with approved, rejected, and modified dispositions across at least two risk evaluator types (drawdown_limit, position_exposure).

### Capability 3: Pipeline Funnel Metrics

**What:** For a given family, symbol, and time range, show conversion rates at each pipeline stage.

**Shape of the answer:**
```
Family: squeeze_breakout
Symbol: btcusdt
Period: 2025-03-01 to 2025-03-15

  Signals emitted:        142
  Decisions triggered:     38  (26.8% of signals)
  Strategies resolved:     35  (92.1% of decisions)
  Risk approved:           28  (80.0% of strategies)
  Risk modified:            4  (11.4% of strategies)
  Risk rejected:            3  (8.6% of strategies)
  Executions submitted:    32  (100% of approved+modified)
  Executions filled:       30  (93.8% of submitted)
  Gate-halted:              0
```

**How it works:** COUNT queries per domain table, grouped by type, filtered by symbol/timeframe/time-range. No cross-table JOIN needed — each table has independent type/source/symbol/timeframe columns.

**Validation:** Produce funnel for each of the three proven slices; verify counts are consistent with end-to-end test expectations.

### Capability 4: Pipeline Health per Symbol

**What:** Detect when a specific symbol/timeframe partition stops flowing through one or more pipeline stages.

**Shape of the answer:**
```
Symbol: ethusdt/300

  Last signal:    2025-03-15T10:42:00Z (bollinger)     ✓ active
  Last decision:  2025-03-15T10:42:00Z (bollinger_sq)  ✓ active
  Last strategy:  2025-03-15T10:41:00Z (squeeze_brk)   ✓ active
  Last risk:      2025-03-15T10:41:00Z (position_exp)  ✓ active
  Last execution: 2025-03-14T22:15:00Z (paper_order)   ⚠ stale (12h)
```

**How it works:** MAX(occurred_at) per domain table, grouped by symbol/timeframe. Compare timestamps to detect staleness gaps between layers.

**Validation:** Verify with test data that a gap at the risk layer is detectable without false positives at the signal layer.

### Capability 5: Confidence and Severity Flow Tracing

**What:** For a given execution, show how confidence and severity values transformed across each pipeline stage.

**Shape of the answer:**
```
Execution: evt_abc123

  Signal:    value=0.023 (bandwidth)
  Decision:  severity=high, confidence=0.91
  Strategy:  confidence=0.82, direction=long
  Risk:      confidence=0.78, disposition=approved
  Execution: side=buy, quantity=500
```

**How it works:** The composite chain reconstruction (Capability 1) already retrieves all intermediate events. This capability is a projection of the confidence/severity fields from that chain.

**Validation:** Verify that confidence values are non-increasing from decision through risk (as expected by the severity scaling logic in S291).

---

## 2. Non-Goals — Items Explicitly Out of Scope

### NG-1: Monitoring and Alerting Infrastructure

**What's excluded:** Prometheus exporters, Grafana dashboards, PagerDuty integration, alerting thresholds, SLO/SLA definitions, on-call runbooks.

**Why:** The Foundry is not in production. Monitoring infrastructure is premature until there are sustained workloads to monitor. S292 publisher counters and `/statusz` are sufficient for development-phase awareness.

**Revisit when:** The Foundry operates continuously with live market data for 7+ days.

### NG-2: Distributed Tracing Infrastructure

**What's excluded:** OpenTelemetry SDK integration, Jaeger/Zipkin backends, trace context propagation headers, span hierarchies, trace sampling configuration.

**Why:** The Foundry already has CorrelationID/CausationID embedded in every event. This is a domain-level causal chain, not a distributed systems trace. The composite read model delivers the same answer (why did X happen?) without the operational overhead of a tracing backend.

**Revisit when:** The Foundry runs as 5+ independent binaries in a distributed deployment where NATS message flow is insufficient for debugging.

### NG-3: Real-Time Streaming Views

**What's excluded:** WebSocket subscriptions for live execution chains, Server-Sent Events for pipeline health, Kafka Connect CDC from ClickHouse, real-time materialized views with automatic refresh.

**Why:** The analytical query surface is pull-based by design (S272/S277). Push-based views add infrastructure complexity without clear operational need in the current development phase.

**Revisit when:** An operator needs sub-second awareness of execution outcomes (e.g., live paper trading with manual oversight).

### NG-4: Dashboard or UI Delivery

**What's excluded:** Web UIs, admin panels, Grafana dashboard JSON, chart rendering, visualization libraries.

**Why:** The wave delivers JSON API endpoints. Visualization is a separate concern that depends on operator tooling preferences. Coupling API delivery with UI delivery would double scope without doubling capability.

**Revisit when:** The API surface is proven and stable; operator feedback indicates CLI/curl is insufficient.

### NG-5: Cross-Symbol Correlation Analysis

**What's excluded:** "Did a decision on BTCUSDT affect an execution on ETHUSDT?" analysis, portfolio-level aggregation, cross-symbol risk correlation, basket execution tracing.

**Why:** The current architecture partitions all processing by symbol/timeframe. Cross-symbol effects are a portfolio management concern that requires a different architectural layer (portfolio risk, cross-asset strategy). Introducing it here would require write-side changes, violating the wave's read-side-only constraint.

**Revisit when:** A portfolio risk layer is chartered.

### NG-6: Historical Replay or Backtest Support

**What's excluded:** Replay infrastructure, backtest harness, what-if simulation, parameter sweep tooling, historical signal reprocessing.

**Why:** These are feature-level capabilities that depend on the composite read model but are not part of it. The wave delivers the read surface; replay/backtest consumes it.

**Revisit when:** The composite read model is proven and an operator wants to answer "what would have happened if parameter P were different?"

### NG-7: Venue Readiness or Compliance

**What's excluded:** Exchange API integration, order routing, live order management, compliance checks, regulatory reporting, audit trails for regulatory purposes.

**Why:** Completely different domain. Paper execution is the only execution mode proven. Mixing venue concerns with observability would violate single-front discipline.

**Revisit when:** Paper execution is stable for 30+ days and a compliance charter is formally opened.

### NG-8: New Signal Families or Vertical Slices

**What's excluded:** MACD vertical slice, ATR-based strategies, VWAP-based decisions, any new signal → execution path.

**Why:** The S294 directive explicitly prohibits opening new families in this wave. The observability wave must prove its value on existing slices before new families are added.

**Revisit when:** Post-wave gate (S299) passes and the observability surface is validated.

### NG-9: Write-Side Schema Changes

**What's excluded:** New ClickHouse columns, new NATS subjects, modified event schemas, new JetStream consumers, changed message types.

**Why:** This wave is read-side only. Any write-side change indicates scope inflation and triggers a stop condition.

**Exception:** Adding ClickHouse indexes on existing columns (e.g., index on correlation_id) is permitted if required for query performance.

### NG-10: Codegen Integration

**What's excluded:** Generating composite queries from YAML specs, auto-generating HTTP handlers, template-driven read models.

**Why:** The composite read model must be designed and validated manually first. Codegen for observability artifacts is a future optimization, not a current need.

**Revisit when:** The manual composite model is stable and a second observability wave is chartered.

---

## 3. Boundary Summary

```
IN SCOPE                              OUT OF SCOPE
─────────────────────────────────────  ─────────────────────────────────────
Composite chain reconstruction        Monitoring/alerting infrastructure
Rejection/modification attribution    Distributed tracing (OTEL/Jaeger)
Pipeline funnel metrics               Real-time streaming views
Pipeline health per symbol            Dashboard/UI delivery
Confidence/severity flow tracing      Cross-symbol correlation
ClickHouse queries + HTTP endpoints   Historical replay/backtest
Deterministic integration tests       Venue readiness/compliance
Validation on 3 existing slices       New signal families
Read-side only                        Write-side schema changes
                                      Codegen integration
```

---

## 4. Dependency Map

```
Existing infrastructure (no changes needed):
  ├── ClickHouse tables: signals, decisions, strategies, risk_assessments, executions
  │     └── All have: event_id, correlation_id, causation_id, occurred_at
  ├── ClickHouse readers: SignalReader, DecisionReader, StrategyReader, RiskReader, ExecutionReader
  │     └── All support: type, source, symbol, timeframe, time-range filters
  ├── HTTP analytical handlers: /analytical/{signals,decisions,strategies,risk,executions}
  │     └── All return: domain objects with meta.query_ms
  └── Event metadata: CorrelationID, CausationID propagated through all actor messages

New capabilities this wave adds:
  ├── Composite query layer (joins individual readers along correlation/causation spine)
  ├── New HTTP endpoint(s): /analytical/execution/explain, /analytical/pipeline/funnel
  ├── Attribution extraction from risk constraints JSON
  └── Pipeline health query (MAX timestamp per domain per symbol)
```
