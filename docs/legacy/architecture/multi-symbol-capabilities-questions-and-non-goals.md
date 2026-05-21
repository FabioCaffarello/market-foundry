# Multi-Symbol Capabilities — Governing Questions and Non-Goals

> Companion to: `multi-symbol-operational-scaling-wave-charter-and-scope-freeze.md`
> Wave: Phase 29 — Multi-Symbol Operational Scaling (S300–S305)
> Date: 2026-03-21

---

## 1. Governing Questions

These questions define what the wave must answer. Each question targets a specific capability that single-symbol operation cannot prove. The wave is complete when all questions are answerable with evidence.

---

### MQ1 — Symbol Isolation

**Is each symbol's pipeline fully isolated from other symbols at every stage?**

- **What to prove**: Signal, decision, strategy, risk, and execution events for symbol A never appear in queries for symbol B.
- **Where to look**: NATS KV bucket partitioning, ClickHouse table writes/reads, composite chain lookups.
- **Evidence required**: Cross-symbol query returning zero foreign results across all 5 tables, for all 3 symbols, across all 3 families.
- **Target stage**: S301

---

### MQ2 — Composite Chain Correctness Per Symbol

**Does the composite read model return correct, complete chains when multiple symbols have concurrent data?**

- **What to prove**: `GET /analytical/composite/chain?correlation_id=X` returns the correct 5-stage chain regardless of how many other symbols have data in the same tables.
- **Where to look**: Composite reader's `WHERE correlation_id = ?` queries; application-side assembly.
- **Evidence required**: Chain completeness (5/5 stages) verified for chains from each of the 3 symbols, with concurrent data present from all symbols.
- **Target stage**: S302

---

### MQ3 — Batch Query Symbol Scoping

**Does the batch chain endpoint correctly scope results to the requested symbol?**

- **What to prove**: `GET /analytical/composite/chains?source=binancef&symbol=btcusdt&timeframe=60` returns only btcusdt chains, even when ethusdt and solusdt chains exist in the same time range.
- **Where to look**: Batch query's initial `WHERE source = ? AND symbol = ? AND timeframe = ?` filter on the executions table; subsequent enrichment.
- **Evidence required**: Batch queries for each symbol return only that symbol's chains. Cross-verification: total chains across 3 symbol-scoped queries equals total chains from unscoped enumeration.
- **Target stage**: S302

---

### MQ4 — Funnel Accuracy Per Symbol

**Are pipeline funnel stage counts accurate when computed per symbol?**

- **What to prove**: `GET /analytical/composite/funnel?type=ema&source=binancef&symbol=btcusdt&timeframe=60` returns stage counts reflecting only btcusdt data, not a mix of symbols.
- **Where to look**: Funnel queries with explicit `symbol` filter on each of the 5 tables.
- **Evidence required**: Sum of per-symbol funnel counts at each stage equals the total count across all symbols (conservation property).
- **Target stage**: S302

---

### MQ5 — Disposition Accuracy Per Symbol

**Are disposition breakdowns (approved/modified/rejected) accurate per symbol?**

- **What to prove**: Disposition counts for symbol A reflect only symbol A's risk assessments.
- **Where to look**: Disposition query's `WHERE symbol = ?` filter on risk assessments table.
- **Evidence required**: Per-symbol disposition totals sum to the all-symbol total. Percentage distributions are independent per symbol.
- **Target stage**: S302

---

### MQ6 — Ordering and Timestamp Consistency Under Concurrency

**Do concurrent multi-symbol events maintain correct temporal ordering within each symbol's chain?**

- **What to prove**: When btcusdt, ethusdt, and solusdt all produce events in the same second, each symbol's chain maintains internal causal ordering (`signal.occurred_at <= decision.occurred_at <= ... <= execution.occurred_at`).
- **Where to look**: Composite chain's stage timestamps; `causation_id` spine integrity under concurrent ingestion.
- **Evidence required**: Zero ordering violations across sampled chains from concurrent multi-symbol operation. Causal spine (`causation_id` references) remain internally consistent per chain.
- **Target stage**: S303

---

### MQ7 — Resource Scaling Behavior

**Does 3× symbol load produce acceptable resource consumption and query latency?**

- **What to prove**: Goroutine count, memory consumption, and ClickHouse query latency scale proportionally (not exponentially) with symbol count.
- **Where to look**: Runtime metrics during multi-symbol operation; composite endpoint response times.
- **Evidence required**: Quantitative comparison of single-symbol vs triple-symbol resource consumption. No goroutine leak, no unbounded queue growth, no query timeout degradation.
- **Target stage**: S304

---

## 2. Answerability Threshold

| Rating | Definition |
|--------|-----------|
| **FULL** | Question answerable with direct, quantitative evidence from the delivered surface |
| **SUBSTANTIAL** | Question answerable with evidence covering >80% of scenarios; bounded, documented gap |
| **PARTIAL** | Question answerable for some scenarios but significant blind spots remain |
| **NOT ANSWERABLE** | No evidence surface exists to address the question |

**Wave closure requires**: All 7 questions at FULL or SUBSTANTIAL, with no question below SUBSTANTIAL.

---

## 3. Question-to-Stage Mapping

| Question | Primary Stage | Validation Surface |
|----------|--------------|-------------------|
| MQ1 — Symbol Isolation | S301 | NATS KV + ClickHouse cross-symbol queries |
| MQ2 — Chain Correctness | S302 | `GET /analytical/composite/chain` |
| MQ3 — Batch Scoping | S302 | `GET /analytical/composite/chains` |
| MQ4 — Funnel Accuracy | S302 | `GET /analytical/composite/funnel` |
| MQ5 — Disposition Accuracy | S302 | `GET /analytical/composite/dispositions` |
| MQ6 — Ordering Consistency | S303 | Composite chain timestamps + causation spine |
| MQ7 — Resource Scaling | S304 | Runtime metrics + query latency measurement |

---

## 4. Relationship to Previous Governing Questions (Q1–Q7)

The Composite Execution Observability Wave (S294–S299) answered Q1–Q7 for single-symbol operation. This wave's MQ1–MQ7 are **not replacements**; they are the **multi-symbol extension**:

| Previous (Single-Symbol) | This Wave (Multi-Symbol) | Relationship |
|--------------------------|--------------------------|-------------|
| Q1 — Why was execution X submitted? | MQ2 — Chain correctness per symbol | Q1 assumed single symbol; MQ2 validates Q1 holds under multi-symbol |
| Q2 — Why rejected/modified? | MQ5 — Disposition accuracy per symbol | Q2 assumed single symbol; MQ5 validates attribution isolation |
| Q3 — Which signals contributed? | MQ1 — Symbol isolation | Q3 assumed no cross-symbol signals; MQ1 proves isolation |
| Q4 — Confidence/severity flow? | MQ6 — Ordering consistency | Q4 assumed sequential; MQ6 validates under concurrency |
| Q5 — Why did symbol stop? | MQ4 — Funnel accuracy per symbol | Q5 used funnel; MQ4 validates funnel correctness per symbol |
| Q6 — Blocked vs approved? | MQ5 — Disposition accuracy per symbol | Direct extension to multi-symbol |
| Q7 — Conversion rate per stage? | MQ4 — Funnel accuracy per symbol | Direct extension to multi-symbol |

---

## 5. Non-Goals (Explicit)

Items below are **explicitly out of scope** for this wave. Each has a rationale explaining why.

---

### NG-1: Real Venue Connectivity

**Out of scope**: No exchange API integration, no real order submission, no fill reconciliation.

**Why**: The wave validates operational scaling of the existing paper pipeline. Real venue connectivity introduces authentication, rate limiting, fill semantics, and regulatory concerns that are a separate architectural domain. Mixing venue work into a scaling wave would make both ungovernable.

---

### NG-2: Order Management System (OMS)

**Out of scope**: No order lifecycle management, no position tracking, no fill aggregation.

**Why**: OMS is a stateful write-side system. This wave is about proving existing read+write correctness at multi-symbol scale, not adding new stateful subsystems.

---

### NG-3: Portfolio-Level Aggregation

**Out of scope**: No cross-symbol portfolio views, no combined PnL, no correlation analysis between symbols.

**Why**: Portfolio aggregation requires combining data across symbols — the opposite of the isolation proof this wave establishes. Portfolio views are a logical successor wave, not a concurrent concern.

---

### NG-4: New Signal, Decision, Strategy, or Risk Families

**Out of scope**: No new families introduced. The wave uses EMA, Trend, and Squeeze exclusively.

**Why**: Adding families simultaneously with symbols would make it impossible to attribute failures: is it the new family or the new symbol? Scaling proof requires holding families constant.

---

### NG-5: Broad Operational Dashboards

**Out of scope**: No Grafana dashboards, no Prometheus metrics export, no alerting rules.

**Why**: Dashboards are a presentation layer over metrics. The wave produces evidence through tests and endpoint validation, not through dashboard construction. Dashboard work belongs to an operational maturity wave.

---

### NG-6: Write-Side Schema Changes

**Out of scope**: No new ClickHouse tables, no ALTER TABLE, no new domain fields, no migration files.

**Why**: The wave validates that existing schemas already support multi-symbol operation. If schema changes are needed, that reveals a design gap — the appropriate response is to document it and scope a fix, not to change schemas mid-wave.

---

### NG-7: Codegen Path Extension

**Out of scope**: No codegen template changes, no new generated families, no codegen tooling work.

**Why**: The codegen path is frozen since S263. Multi-symbol scaling is orthogonal to code generation.

---

### NG-8: Performance Optimization

**Out of scope**: No query optimization, no caching layers, no connection pooling changes, no ClickHouse tuning.

**Why**: S304 **measures** resource behavior; it does not **optimize** it. If measurements reveal unacceptable degradation, the wave documents the finding and scopes remediation as a successor stage. Optimization without measurement evidence is premature.

---

### NG-9: New HTTP Endpoints

**Out of scope**: No new analytical endpoints, no multi-symbol aggregation endpoints, no admin endpoints.

**Why**: The existing composite surface (4 endpoints) is validated, not extended. New endpoints belong to capability expansion waves, not scaling proof.

---

### NG-10: Residual Gap Closure from S299

**Out of scope**: GAP-Q2-A (per-constraint trigger identification) and GAP-Q5-A (pre-execution stopped chain discovery) are not addressed in this wave.

**Why**: These gaps are single-symbol observability depth enhancements. They are independent of multi-symbol scaling and would dilute the wave's focus. They remain documented as future enhancement opportunities.

---

## 6. Scope Inflation Detection

The following patterns indicate scope creep and should trigger a pause:

| Pattern | Action |
|---------|--------|
| "We need a new table for multi-symbol tracking" | Stop — existing schema should suffice; if not, document as finding |
| "Let's add a 4th symbol while we're at it" | Stop — 3 symbols is the chartered scope; 4th adds work without proportional insight |
| "We should optimize X while we're measuring" | Stop — measure first, optimize in successor stage |
| "Let's also add portfolio-level views" | Stop — NG-3 explicitly excludes this |
| "This would be easier with a new endpoint" | Stop — NG-9 explicitly excludes this |
| "We should close GAP-Q2-A now" | Stop — NG-10 explicitly excludes this |

---

## 7. Wave Exit Criteria

The wave closes when:

1. All 7 governing questions (MQ1–MQ7) are at FULL or SUBSTANTIAL.
2. Zero regressions against the S299 baseline (Q1–Q7 still answerable).
3. Non-goals remain untouched — no scope inflation occurred.
4. Evidence is documented in architecture documents and stage reports.
5. The closure gate (S305) formally records the verdict.
