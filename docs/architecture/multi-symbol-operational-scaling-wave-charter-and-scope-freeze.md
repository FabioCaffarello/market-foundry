# Multi-Symbol Operational Scaling Wave — Charter and Scope Freeze

> Wave: Phase 29 — Multi-Symbol Operational Scaling
> Status: **OPEN**
> Charter stage: S300
> Date: 2026-03-21
> Predecessor wave: Phase 28 — Composite Execution Observability (S294–S299, CLOSED)

---

## 1. Strategic Context

The Composite Execution Observability Wave (S294–S299) closed with 6/7 governing questions FULL, 1/7 SUBSTANTIAL, and zero regressions. The system can explain, attribute, and aggregate paper execution for a single symbol flowing through three vertical slices (EMA, Trend, Squeeze).

The next systemic pressure is **not** more families or venue readiness. It is proving that the existing pipeline — signal, decision, strategy, risk, execution, and composite observability — operates **consistently, correctly, and in isolation** when multiple symbols run simultaneously through the same infrastructure.

Phase 2 (S11–S17) established multi-symbol readiness at the infrastructure level: actor hierarchy, config-driven activation, and partition isolation via `{source}.{symbol}.{timeframe}`. What it did **not** prove is end-to-end operational correctness when N>1 symbols produce concurrent chains across all five stages with composite read model integrity.

This wave closes that gap.

---

## 2. Wave Objective

**Prove that the market-foundry paper execution pipeline operates correctly, in isolation, and with full composite observability when 3 symbols run simultaneously through existing families.**

"Correctly" means:
- No cross-symbol data contamination in any stage
- Per-symbol chain completeness matches single-symbol baselines
- Composite read model returns correct, symbol-scoped results
- Funnel and disposition aggregations are accurate per symbol
- Attribution is symbol-scoped with no cross-bleed

---

## 3. Symbol Selection

### Initial symbol set: 3 symbols

| Symbol | Source | Rationale |
|--------|--------|-----------|
| `btcusdt` | `binancef` | Baseline — already proven in single-symbol operation and all integration tests |
| `ethusdt` | `binancef` | Second symbol — already present in live analytical tests; high-volume pair validates throughput |
| `solusdt` | `binancef` | Third symbol — validates scaling beyond the familiar two; mid-cap pair with different volatility profile |

### Why 3 symbols?

- **1 symbol** is the status quo — no scaling proof.
- **2 symbols** proves isolation but not scaling behavior — pairwise might mask ordering bugs.
- **3 symbols** is the minimum to reveal systemic issues: resource contention, ordering anomalies, and fan-out bottlenecks. A triple-symbol proof generalizes to N.
- **>3 symbols** would expand scope without proportional insight gain at this stage.

---

## 4. Families and Slices In Scope

### Included (existing, proven families only):

| Family | Type | Stages Covered | Status |
|--------|------|----------------|--------|
| **EMA** | Signal → Decision → Strategy → Risk → Execution | All 5 | Proven single-symbol |
| **Trend** | Signal → Decision → Strategy → Risk → Execution | All 5 | Proven single-symbol |
| **Squeeze** | Signal → Decision → Strategy → Risk → Execution | All 5 | Proven single-symbol (S283–S293) |

### Explicitly excluded:

| Family/Slice | Reason |
|-------------|--------|
| New signal families (RSI standalone, MACD standalone) | New families are not the wave objective |
| New decision/strategy types | Depth expansion is out of scope |
| New risk constraint types | Write-side changes out of scope |
| Codegen-generated families | Codegen path frozen; not relevant to scaling proof |

### Composite observability surface (included, read-only validation):

| Endpoint | Validation Target |
|----------|------------------|
| `GET /analytical/composite/chain` | Per-symbol chain correctness |
| `GET /analytical/composite/chains` | Multi-symbol batch isolation |
| `GET /analytical/composite/funnel` | Per-symbol stage counts accuracy |
| `GET /analytical/composite/dispositions` | Per-symbol disposition accuracy |

---

## 5. Scope Freeze Rules

1. **No new families** may be introduced during this wave. All work uses the existing EMA, Trend, and Squeeze slices.
2. **No write-side schema changes**. All ClickHouse tables, domain types, and event schemas remain unchanged.
3. **No new HTTP endpoints**. The existing composite surface is validated, not extended.
4. **No venue readiness work**. All execution remains paper-mode.
5. **No portfolio-level aggregation**. Each symbol is validated independently; cross-symbol portfolio views are a separate wave.
6. **No OMS or order routing**. Execution remains intent-only; no external exchange integration.
7. **Configuration changes only** for symbol activation — the system is already symbol-agnostic by design.

---

## 6. Wave Entry Conditions (all met)

| Condition | Evidence |
|-----------|----------|
| Composite observability wave closed | S299 — WAVE CLOSED, 6/7 FULL |
| Three vertical slices proven single-symbol | EMA (Phase 10), Trend (Phase 11), Squeeze (S283–S293) |
| Composite read model operational | S296 — 5-table composition via correlation_id spine |
| HTTP explainability surface operational | S297–S298 — 4 endpoints with 36+ tests |
| Infrastructure supports multi-symbol | Phase 2 (S11–S17) — partition isolation, config-driven activation |
| Zero regressions in predecessor wave | S299 — verified across all dimensions |

---

## 7. Recommended Stage Sequence

| Stage | Title | Objective | Dependencies |
|-------|-------|-----------|--------------|
| **S300** | Multi-Symbol Operational Scaling Charter (this document) | Open wave, freeze scope | S299 |
| **S301** | Multi-Symbol Config Activation and Isolation Proof | Activate 3 symbols; prove partition isolation across NATS KV, ClickHouse writes, and actor hierarchy | S300 |
| **S302** | Cross-Symbol Composite Read Model Validation | Prove composite chain, batch, funnel, and disposition queries return correct symbol-scoped results with no cross-contamination | S301 |
| **S303** | Multi-Symbol Concurrent Pipeline Consistency | Validate ordering, timestamp consistency, and chain completeness when 3 symbols produce concurrent events | S302 |
| **S304** | Operational Scaling Boundary and Resource Proof | Validate resource consumption, goroutine scaling, and ClickHouse query performance with 3× symbol load | S303 |
| **S305** | Multi-Symbol Operational Gate and Wave Closure | Evidence gate: all governing questions answered; regression check; wave closure verdict | S304 |

**Estimated wave size**: 6 stages (S300–S305), consistent with predecessor waves.

---

## 8. Success Criteria

The wave is complete when:

1. Three symbols produce independent, complete chains through all three families.
2. No cross-symbol data contamination is detected in any stage or table.
3. Composite read model returns correct results when queried per symbol and across the batch surface.
4. Funnel and disposition counts are accurate per symbol, not aggregated across symbols.
5. Attribution remains symbol-scoped with no cross-bleed.
6. All governing questions (defined in companion document) are answerable.
7. Zero regressions against the S299 baseline.

---

## 9. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Cross-symbol contamination in NATS KV buckets | Low | High | Partition key `{source}.{symbol}.{timeframe}` already isolates; S301 validates |
| ClickHouse query fan-out degrades performance | Medium | Medium | ORDER BY prefix `(source, symbol, ...)` ensures efficient filtering; S304 measures |
| Concurrent chain writes cause ordering anomalies | Medium | Medium | `occurred_at` is event-sourced from domain, not ingestion time; S303 validates |
| Goroutine/actor proliferation with 3× symbols | Low | Low | Actor hierarchy already designed for N symbols; S304 measures resource bounds |
| Scope creep into new families or venue work | Medium | High | Charter freeze rules (section 5) explicitly prevent; gate reviews enforce |

---

## 10. Relationship to Existing Architecture

This wave **does not alter** any architectural pattern. It validates that existing patterns scale:

- **Partition isolation** (`{source}.{symbol}.{timeframe}`) — proven in infrastructure, now proven in operation
- **Actor hierarchy** — per-symbol actor trees already exist; now exercised concurrently
- **Composite read model** — application-side composition already symbol-filtered; now validated multi-symbol
- **Causal spine** — `correlation_id` / `causation_id` already scoped to individual chains; no cross-symbol spine expected

The wave produces **evidence**, not new architecture.
