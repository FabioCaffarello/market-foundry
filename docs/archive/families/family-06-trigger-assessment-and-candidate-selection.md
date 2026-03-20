# Family 06 — Trigger Assessment and Candidate Selection

> S191 principal deliverable. Formal assessment of whether a Family 06 candidate exists
> that satisfies the S190 gate conditions. Result: **no candidate qualifies; manual
> expansion aborted.**

## Purpose

This document evaluates whether Family 06 should proceed by:
1. Identifying all real candidates from the current codebase state.
2. Testing each candidate against the S190 gate conditions.
3. Reaching a formal proceed/abort decision.
4. Documenting the structural reasons behind the outcome.

## S190 Gate Conditions (Non-Negotiable)

| ID | Condition |
|----|-----------|
| C1 | Candidate must NOT require write-path changes |
| C2 | Reader parameters must NOT exceed 11 |
| C3 | Family 06 must measure and report ceiling metrics |
| C4 | Codegen tranche must be scoped before Family 07 trigger assessment |

## Current Analytical Coverage

Wave B delivered 6 families covering all vertical layers:

```
L1 Evidence  ──── candle           (Family 00, baseline)
L2 Signal    ──── rsi              (Family 01, Wave B)
L3 Decision  ──── rsi_oversold     (Family 02, Wave B)
L4 Strategy  ──── mean_reversion   (Family 03, Wave B)
L5 Risk      ──── position_exposure (Family 04, Wave B)
L6 Execution ──── paper_order      (Family 05, Wave B)
```

All 6 readers are type-parameterized. The `type` query parameter already allows querying any event type within each layer. Within-layer coverage is limited only by what the writer persists to ClickHouse.

## Candidate Identification

### Method

All NATS registries, writer pipeline entries, domain types, and ClickHouse tables were inventoried. Event types were classified as:
- **Covered**: writer pipeline entry exists AND reader exists.
- **Partially infrastructure-ready**: NATS registry exists, domain type exists, table exists (shared), but writer pipeline entry is missing.
- **Uncovered**: no analytical infrastructure.

### Candidates Found

| # | Candidate | Layer | NATS Registry | Table | Writer Pipeline | Reader | Classification |
|---|-----------|-------|:-------------:|:-----:|:---------------:|:------:|---------------|
| A | EMA Crossover | L2 Signal | ✅ | ✅ (shared: `signals`) | ❌ | ✅ (shared: `SignalReader`) | Partially ready |
| B | Venue Market Order | L6 Execution | ✅ | ✅ (shared: `executions`) | ❌ | ✅ (shared: `ExecutionReader`) | Partially ready |
| C | Trade Burst | L1 Evidence | ⚠️ | ❌ | ❌ | ❌ | Uncovered |
| D | Volume Metrics | L1 Evidence | ⚠️ | ❌ | ❌ | ❌ | Uncovered |
| E | Observation Trades | L0 | ✅ | ❌ | ❌ | ❌ | Not analytical |

### Structural Observation

For candidates A and B, the analytical read path already works:
- `GET /analytical/signal/history?type=ema_crossover` returns 200 with empty results today.
- `GET /analytical/execution/history?type=venue_market_order` would similarly return 200 with empty results.

The readers are generic. The handlers are generic. The only missing piece is data in ClickHouse — which requires write-path changes to produce.

## Gate Condition Testing

### Candidate A: EMA Crossover

**C1 — No write-path changes:**

Required changes to enable EMA Crossover data:
1. Add `WriterEMACrossoverSignalConsumer()` to `internal/adapters/nats/signal_registry.go`.
2. Add pipeline entry to `cmd/writer/pipeline.go` (consumer spec, inserter, mapper binding).
3. Add `"ema_crossover"` to `signal_families` in writer config.

The mapper (`mapSignalRow`) and table (`signals`) are reusable — but the consumer and pipeline entry are new write-path artifacts.

**Result: ❌ FAILS C1.**

**C2 — Reader params ≤ 11:**

The existing `SignalReader.QuerySignalHistory()` has 7 parameters: `signalType, source, symbol, timeframe, since, until, limit`. EMA Crossover requires zero additional parameters — it uses the same query with `type=ema_crossover`.

**Result: ✅ PASSES C2.**

**Net: DISQUALIFIED by C1.**

### Candidate B: Venue Market Order

**C1 — No write-path changes:**

Required changes:
1. Add `WriterVenueMarketOrderConsumer()` to execution registry.
2. Add `mapVenueExecutionRow()` to `cmd/writer/mappers.go` (different event struct: `VenueOrderFilledEvent` vs `PaperOrderSubmittedEvent`).
3. Add pipeline entry to `cmd/writer/pipeline.go`.
4. Add config entry for the new execution family.

More invasive than Candidate A — requires a new mapper in addition to consumer/pipeline.

**Result: ❌ FAILS C1 (more severely).**

### Candidates C, D, E

All require full 9-artifact expansions including new migrations, mappers, pipeline entries, readers, and handlers. They fail C1 categorically and also represent substantially higher complexity.

**Result: ❌ FAIL C1.**

## Formal Decision

```
┌──────────────────────────────────────────────────────────────────┐
│  TRIGGER ASSESSMENT RESULT: NO VIABLE CANDIDATE                   │
│                                                                   │
│  All candidates require write-path changes.                       │
│  The S190 gate condition C1 blocks all candidates.                │
│  Family 06 manual analytical expansion is ABORTED.                │
│                                                                   │
│  The S190 alternate path activates:                               │
│    S191 → Codegen Tranche Scoping (not family expansion)          │
│                                                                   │
│  Wave B manual expansion ends at 6 families / 6 layers.           │
└──────────────────────────────────────────────────────────────────┘
```

## Why This Outcome Was Structurally Inevitable

The Wave B expansion pattern was designed for **vertical layer expansion**: each family added a new analytical reader for a layer that already had data flowing through the writer. The writer was built once with all 6 pipeline entries; the analytical expansion then added read-path coverage incrementally.

Once all 6 layers had read-path coverage, the only remaining candidates were **within-layer variants** (e.g., EMA Crossover within Signals, Venue Market Order within Executions). But within-layer variants share the same table, reader, use case, and handler — the only missing piece is the writer pipeline entry. This makes them write-path problems, not analytical read-path problems.

The S190 gate condition (no write-path changes) correctly identifies this structural boundary. It was not an arbitrary constraint — it was a condition that distinguished true analytical expansion from writer pipeline extension.

## Architectural Implications

### 1. The Analytical Read Path Is Complete (For Existing Data)

Every type-parameterized reader already supports queries for any event type within its layer. No further read-path expansion is needed until:
- A genuinely new layer is discovered (unlikely in the near term).
- Cross-family features are added (aggregations, correlations).
- The query surface evolves beyond simple time-range + type filtering.

### 2. Future Expansion Is a Write+Generate Problem

The next analytical families will be created by:
1. Adding writer pipeline entries (write-path work).
2. Generating reader/handler/test code via codegen (automated).

This is fundamentally different from the manual 9-artifact pattern and should not be governed by the same gate conditions.

### 3. The Manual Pattern Specification Is Now Complete

Six families with zero creative decisions across 5 expansions provide a complete specification for codegen:
- Schema conventions: documented across 6 migration files.
- Mapper conventions: documented across 6 mapper functions.
- Reader conventions: documented across 6 reader implementations.
- Handler conventions: documented across 6 handler methods.
- Test conventions: documented across 289 tests.

This specification is the input to the codegen tranche.

## Triggers for Family 07+

Family 07 and beyond are governed by:

| Trigger | Condition | Unlocks |
|---------|-----------|---------|
| Codegen tranche implementation | Templates built and validated against existing 6 families | Generated families (F07+) |
| Writer pipeline extension | New consumer/pipeline entry for an event type | Data flow for codegen-generated reader to consume |
| New vertical layer | A layer outside L1–L6 emerges with analytical need | Manual or codegen expansion to new layer |

## Companion Documents

| Document | Purpose |
|----------|---------|
| `family-06-candidate-comparison-matrix.md` | Detailed per-candidate evaluation against all gate conditions |
| `family-06-selection-rationale-or-abort-rationale.md` | Full abort rationale with risk assessment and deferred candidates |
| `stage-s191-...-report.md` | Stage completion report |
