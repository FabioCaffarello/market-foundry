# Family 06 — Candidate Comparison Matrix

> Formal evaluation of all potential Family 06 candidates against the S190 gate conditions.
> This matrix is the evidence base for the selection (or abort) decision in S191.

## S190 Gate Conditions (Binding)

| # | Condition | Source |
|---|-----------|--------|
| C1 | Candidate must NOT require write-path changes | S190 §Gate Decision |
| C2 | Reader parameters must NOT exceed 11 | S190 §Gate Decision |
| C3 | Family 06 must measure and report ceiling metrics | S190 §Gate Decision |
| C4 | Codegen tranche must be scoped before Family 07 trigger | S190 §Gate Decision |

## Candidate Inventory

### How Candidates Were Identified

Candidates were derived from the complete event-type inventory in the codebase:
- All NATS registries (`internal/adapters/nats/*_registry.go`)
- All writer pipeline entries (`cmd/writer/pipeline.go`)
- All ClickHouse migrations (`deploy/migrations/*.sql`)
- All reader adapters (`internal/adapters/clickhouse/*_reader.go`)
- Domain type definitions (`internal/domain/*/`)
- Configuration references (`deploy/configs/`)

### Coverage Status

| Layer | Family | Writer Pipeline | ClickHouse Table | Analytical Reader | Status |
|-------|--------|-----------------|------------------|-------------------|--------|
| L1 Evidence | candle | ✅ | ✅ (001) | ✅ CandleReader | **DELIVERED** |
| L2 Signal | rsi | ✅ | ✅ (002) | ✅ SignalReader | **DELIVERED** |
| L3 Decision | rsi_oversold | ✅ | ✅ (003) | ✅ DecisionReader | **DELIVERED** |
| L4 Strategy | mean_reversion_entry | ✅ | ✅ (004) | ✅ StrategyReader | **DELIVERED** |
| L5 Risk | position_exposure | ✅ | ✅ (005) | ✅ RiskReader | **DELIVERED** |
| L6 Execution | paper_order | ✅ | ✅ (006) | ✅ ExecutionReader | **DELIVERED** |

**All 6 vertical layers have analytical coverage. Remaining candidates are within-layer variants.**

---

## Candidate Evaluation

### Candidate A: EMA Crossover (Signal Layer, L2)

| Dimension | Assessment |
|-----------|------------|
| **Domain** | Signal (L2) — within-layer variant of existing RSI signal family |
| **Event type** | `signal.events.ema_crossover.generated` — registered in `SignalRegistry` |
| **NATS infrastructure** | ✅ Subject and stream defined, `StoreEMACrossoverSignalConsumer()` exists |
| **ClickHouse table** | ✅ Reuses `signals` table (migration 002) — schema is type-agnostic |
| **Domain type** | ✅ Uses `signal.SignalGeneratedEvent` — same as RSI |
| **Writer mapper** | ✅ `mapSignalRow()` is generic for all signal types (confirmed by test at `mappers_test.go:216`) |
| **Writer pipeline entry** | ❌ **MISSING** — no `WriterEMACrossoverSignalConsumer()` in writer registry, no pipeline entry in `pipeline.go` |
| **Reader** | ✅ `SignalReader.QuerySignalHistory()` already accepts any `signalType` — query is parameterized |
| **Handler** | ✅ `GET /analytical/signal/history?type=ema_crossover` already works — returns empty results |
| **Reader param impact** | 0 — no new parameters needed (type is already a parameter) |

**C1 (no write-path changes): ❌ FAILS** — requires new writer pipeline entry (`WriterEMACrossoverSignalConsumer` + pipeline catalog addition in `pipeline.go`)
**C2 (reader params ≤ 11): ✅ PASSES** — 0 new parameters
**C3 (ceiling metrics): N/A**
**C4 (codegen scope): N/A**

**Verdict: DISQUALIFIED — violates C1.**

**Critical insight:** The analytical read path for EMA crossover already functions. The `SignalReader` is type-parameterized — querying `type=ema_crossover` works today with zero code changes. The bottleneck is entirely on the write side: no writer pipeline entry exists to persist EMA crossover events to ClickHouse. Adding this pipeline entry is a write-path change.

---

### Candidate B: Venue Market Order (Execution Layer, L6)

| Dimension | Assessment |
|-----------|------------|
| **Domain** | Execution (L6) — within-layer variant of existing paper_order family |
| **Event type** | `execution.fill.venue_market_order.filled` — registered in `ExecutionRegistry` |
| **NATS infrastructure** | ✅ Subject and stream defined (`EXECUTION_FILL_EVENTS` — different stream from paper_order) |
| **ClickHouse table** | ✅ Could reuse `executions` table (migration 006) |
| **Domain type** | ⚠️ `execution.VenueOrderFilledEvent` — different from `PaperOrderSubmittedEvent` |
| **Writer mapper** | ❌ **MISSING** — no `mapVenueExecutionRow()`, different event struct |
| **Writer pipeline entry** | ❌ **MISSING** — no consumer spec, no pipeline entry |
| **Reader** | ✅ `ExecutionReader` already filters by type — reusable |
| **Handler** | ✅ Existing endpoint supports type filtering |
| **Reader param impact** | 0 — no new parameters needed |

**C1 (no write-path changes): ❌ FAILS** — requires new mapper, new consumer spec, new pipeline entry
**C2 (reader params ≤ 11): ✅ PASSES**

**Verdict: DISQUALIFIED — violates C1 (more severely than Candidate A).**

---

### Candidate C: Trade Burst (Evidence Layer, L1)

| Dimension | Assessment |
|-----------|------------|
| **Domain** | Evidence (L1) — within-layer variant of existing candle family |
| **Event type** | Referenced in config docs as potential evidence type |
| **NATS infrastructure** | ⚠️ Partial — referenced in documentation but no explicit registry entry found |
| **ClickHouse table** | ❌ **MISSING** — requires new migration (007) |
| **Domain type** | ⚠️ Not yet defined as a concrete struct |
| **Writer mapper** | ❌ **MISSING** |
| **Writer pipeline entry** | ❌ **MISSING** |
| **Reader** | ❌ Would need new reader (different schema from candles) |
| **Handler** | ❌ Would need new handler method |
| **Reader param impact** | +6–7 new reader params |

**C1 (no write-path changes): ❌ FAILS** — requires new migration, mapper, pipeline, reader, handler
**C2 (reader params ≤ 11): ⚠️ UNCLEAR** — depends on schema design

**Verdict: DISQUALIFIED — violates C1 (full 9-artifact expansion required).**

---

### Candidate D: Volume Metrics (Evidence Layer, L1)

| Dimension | Assessment |
|-----------|------------|
| **Domain** | Evidence (L1) — within-layer variant |
| **Event type** | Referenced in documentation as potential evidence type |
| **NATS infrastructure** | ⚠️ Partial |
| **ClickHouse table** | ❌ **MISSING** — requires new migration |
| **Domain type** | ⚠️ Not yet defined |
| **Writer mapper** | ❌ **MISSING** |
| **Writer pipeline entry** | ❌ **MISSING** |
| **Reader** | ❌ Would need new reader |
| **Reader param impact** | ⚠️ Potentially wider than existing families (VWAP, buy/sell volume) |

**C1 (no write-path changes): ❌ FAILS**
**C2 (reader params ≤ 11): ⚠️ RISK** — volume metrics may need additional filter dimensions

**Verdict: DISQUALIFIED — violates C1.**

---

### Candidate E: Observation Trades (L0)

| Dimension | Assessment |
|-----------|------------|
| **Domain** | Observation (L0) — market input layer |
| **Current flow** | Fed to derive binary for evidence sampling — NOT in analytical pipeline |
| **ClickHouse table** | ❌ Not planned |
| **Writer** | ❌ Not in writer service scope |

**C1 (no write-path changes): ❌ FAILS** — would require fundamental pipeline restructuring
**Verdict: DISQUALIFIED — not an analytical family; wrong layer entirely.**

---

## Comparison Matrix

| Candidate | C1 (No Write-Path) | C2 (Params ≤ 11) | New Migration | New Mapper | New Pipeline | New Reader | New Handler | Complexity |
|-----------|:-------------------:|:-----------------:|:-------------:|:----------:|:------------:|:----------:|:-----------:|:----------:|
| A: EMA Crossover | ❌ | ✅ | No | No | **Yes** | No | No | Low |
| B: Venue Market Order | ❌ | ✅ | No | **Yes** | **Yes** | No | No | Medium |
| C: Trade Burst | ❌ | ⚠️ | **Yes** | **Yes** | **Yes** | **Yes** | **Yes** | High |
| D: Volume Metrics | ❌ | ⚠️ | **Yes** | **Yes** | **Yes** | **Yes** | **Yes** | High |
| E: Observation Trades | ❌ | N/A | **Yes** | **Yes** | **Yes** | **Yes** | **Yes** | Very High |

## Structural Finding

**No candidate passes condition C1.**

The reason is architectural, not accidental:
1. All 6 vertical layers already have analytical read-path coverage.
2. The readers are type-parameterized — they already support querying any event type within their layer.
3. Every uncovered event type lacks a **writer pipeline entry** to persist data to ClickHouse.
4. Adding a writer pipeline entry is, by definition, a write-path change.

The analytical read-path is **more generic than the write-path**. This means within-layer expansion is a write-side problem, not a read-side problem. The S190 gate condition (no write-path changes) correctly identifies this constraint.

## Conclusion

**No candidate satisfies the S190 gate conditions for Family 06.**

The expansion decision must be escalated to the selection rationale document for formal disposition.
