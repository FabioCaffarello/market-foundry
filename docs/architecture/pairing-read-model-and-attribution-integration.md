# Pairing Read Model and Attribution Integration

**Stage**: S481
**Wave**: Round-Trip Pairing (S479--S483)
**Date**: 2026-03-26
**Predecessor**: S480 (Canonical Round-Trip and Leg-Pairing Model)

---

## 1. Purpose

This document describes the read model that makes round-trip pairing accessible via HTTP query surfaces, and how paired round-trips integrate with the effectiveness/attribution pipeline to produce realized P&L classifications instead of unresolved outcomes.

---

## 2. Architecture

### 2.1 Data Flow

```
ClickHouse (existing execution tables)
    |
    v
CompositeReader.QueryChainsBatch()
    |
    v
CompositeExecutionChain[] (with fills, strategy direction)
    |
    v
IntentToLeg() per chain (infer entry/exit direction)
    |
    v
Leg[] (typed entry/exit legs with aggregated fill data)
    |
    v
MatchFIFO(legs, config) — S480 canonical matching
    |
    v
RoundTrip[] (paired + unmatched, with reason codes)
    |
    +-- Paired: ClassifyPair(entry_intent, exit_intent) -> Attribution (win/loss/breakeven)
    +-- Unmatched: state + reason code (no_exit_found, no_entry_found, etc.)
    |
    v
PairingReply (views + summary + meta)
```

### 2.2 Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Compute pairing at read time, not write time | Avoids OMS expansion, no new tables, fully additive |
| Reuse CompositeReader | No new ClickHouse queries; chains already carry all needed data |
| Integrate pairing into existing effectiveness pipeline | Reduces unresolved count without breaking existing API contracts |
| Strategy direction from chain context | Uses strategy stage direction when available, defaults to long |
| Additive to RoundTripView | Attribution is an optional field, nil for unmatched legs |

### 2.3 Components

| Component | Location | Responsibility |
|-----------|----------|----------------|
| `GetPairingUseCase` | `internal/application/analyticalclient/get_pairing.go` | Read model: fetch chains, convert to legs, match, classify |
| `PairingQuery/Reply` | `internal/application/analyticalclient/pairing_contracts.go` | Request/response contracts |
| `RoundTripView` | `pairing_contracts.go` | HTTP-facing round-trip with optional attribution |
| `CompositeWebHandler.GetPairing` | `internal/interfaces/http/handlers/composite.go` | HTTP handler for batch pairing |
| `CompositeWebHandler.GetPairingSingle` | `composite.go` | HTTP handler for single-chain pairing |

---

## 3. HTTP Endpoints

### 3.1 Batch Pairing

```
GET /analytical/composite/pairing?source=...&symbol=...&timeframe=...
```

**Parameters**:
- `source` (required): venue/segment
- `symbol` (required): instrument
- `timeframe` (required): candle interval
- `since`, `until` (optional): unix seconds time range
- `limit` (optional): default 50, max 200
- `state` (optional): filter by `paired`, `unmatched_entry`, `unmatched_exit`
- `side` (optional): filter by `buy`, `sell`

**Response**: `PairingReply` with round-trips, summary, and meta.

### 3.2 Single-Chain Pairing

```
GET /analytical/composite/pairing/chain?correlation_id=...&symbol=...
```

**Parameters**:
- `correlation_id` (required): chain identifier
- `symbol` (required): S301 isolation

**Response**: Same `PairingReply` structure for a single chain.

---

## 4. Effectiveness Integration

### 4.1 Before S481 (S476 behavior)

The batch effectiveness pipeline classified each execution chain independently using `effectiveness.Classify()`. Since single-leg fills have no paired exit, all filled orders were classified as `unresolved`.

### 4.2 After S481

The batch effectiveness pipeline now:

1. Collects all filled chains and converts them to typed legs
2. Runs `MatchFIFO()` to identify entry/exit pairs
3. For paired round-trips: uses `ClassifyPair(entry, exit)` producing `win`, `loss`, or `breakeven`
4. For unpaired legs: falls back to `Classify()` which returns `unresolved`
5. Combines both sets into the evaluation results

This change is transparent to API consumers — the response contract is unchanged. The only observable difference is that some evaluations that were previously `unresolved` now return `win`, `loss`, or `breakeven` with realized P&L.

### 4.3 Affected Use Cases

| Use Case | Change |
|----------|--------|
| `GetEffectivenessUseCase.executeBatch` | Now runs FIFO matching before classification |
| `GetEffectivenessSummaryUseCase.Execute` | Same pairing integration for cohort aggregation |
| `GetEffectivenessUseCase.executeSingle` | Unchanged (single-chain lookups have only one leg) |

---

## 5. Guard Rails Observed

| # | Guard Rail | Status |
|---|-----------|--------|
| G1 | No OMS expansion | Observed |
| G2 | No new ClickHouse tables | Observed |
| G3 | No new exchange connectivity | Observed |
| G4 | No write-path changes | Observed |
| G5 | No portfolio analytics | Observed |
| G6 | No real-time streaming | Observed |
| G7 | No domain type refactoring (additive only) | Observed |
| G8 | No UI/dashboards | Observed |
| G9 | No risk/position engine | Observed |
| G10 | Additive only (zero changes to existing behavior) | Observed |

---

## 6. Limitations

1. **Cross-session pairing**: Pairing operates within the scope of a single batch query. Chains from different time windows are not paired across queries.

2. **Single-chain lookup**: The single-chain endpoint (`/pairing/chain`) can only produce a single leg since each chain has exactly one execution intent. Pairing requires multiple chains.

3. **FIFO only**: No LIFO, HIFO, or other matching algorithms.

4. **No real-time**: Pairing is computed on demand from historical data. No streaming pairing.

5. **No cross-exchange**: Pairing enforces M2 (same source/segment). No cross-venue matching.

6. **Strategy direction inference**: When the strategy stage is absent, defaults to long convention (buy=entry, sell=exit). This may misclassify short-strategy fills.
