# Analytical Writer: Correctness and Test Foundation

## Purpose

This document defines what is under proof for the writer service after S152, and what remains outside test coverage.

## Components Under Test

### 1. Mappers (`cmd/writer/mappers.go`)

The six mapper functions are the most critical correctness boundary in the writer path. Each converts a domain event into a `[]any` row matching the ClickHouse DDL column order exactly.

**What is tested:**

| Mapper | DDL columns | Tests |
|---|---|---|
| `mapCandleRow` | 16 | Column count, metadata positions, domain field values, empty decimal handling |
| `mapSignalRow` | 12 | Column count, domain fields, JSON metadata serialization, nil metadata |
| `mapDecisionRow` | 14 | Column count, domain fields, signals JSON array, outcome string cast |
| `mapStrategyRow` | 15 | Column count, domain fields, direction string cast, decisions JSON |
| `mapRiskRow` | 17 | Column count, domain fields, disposition cast, constraints JSON, rationale |
| `mapExecutionRow` | 20 | Column count, domain fields, side/status casts, risk/fills JSON, exec correlation/causation IDs |

**Critical contracts proven:**

- Column count matches DDL exactly for each table.
- Metadata fields (event_id, occurred_at, correlation_id, causation_id) occupy positions 0–3 consistently.
- `parseFloat` returns 0 for empty/invalid strings (silent degradation, acceptable for analytical layer).
- `marshalJSON` returns `"{}"` for nil, valid JSON for maps/structs/slices.
- Enum-like fields (outcome, direction, disposition, side, status) are cast to string correctly.
- Nested JSON columns (signals, decisions, strategies, constraints, risk, fills, parameters, metadata) produce valid, deserializable JSON.

### 2. Inserter Buffer Logic (`cmd/writer/inserter.go`)

**What is tested:**

- `enforceMaxPending`: evicts oldest rows (FIFO) when buffer exceeds limit.
- Eviction correctness: the newest rows survive, oldest are dropped.
- Tracker integration: `events_dropped` counter incremented on eviction.
- Nil tracker safety: no panic when tracker is nil.
- Empty buffer flush: no-op, no panic.
- `scheduleFlush` with nil engine/pid: no panic.

**What is NOT tested (requires infrastructure):**

- `flush()` with a real ClickHouse client (InsertBatch call).
- Actor message dispatch (requires Hollywood actor framework runtime).
- Timer-based flush triggering (requires actor engine message passing).
- Graceful shutdown drain behavior (requires actor lifecycle).

### 3. Helper Functions

- `parseFloat`: 7 cases including empty string, invalid input, negative, high precision.
- `marshalJSON`: nil, map, slice, empty map, empty slice, struct.

## Architecture Invariants

1. **Column order is the contract.** Each mapper must produce values in exact DDL order. The column count tests are the first line of defense against drift.
2. **No validation in mappers.** Mappers assume events are already validated upstream. They convert, not gate.
3. **Silent degradation for numerics.** `parseFloat("")` → 0.0 is intentional: the analytical layer tolerates degraded precision over crashing the pipeline.
4. **JSON serialization is the escape hatch.** Complex nested fields are serialized as JSON strings for ClickHouse String columns. This trades queryability for schema simplicity.

## What Remains Outside Coverage

| Component | Why | Risk | Mitigation |
|---|---|---|---|
| `flush()` → ClickHouse | Requires live ClickHouse connection | Medium — InsertBatch is thin wrapper | Integration test in future stage |
| Actor message dispatch | Requires Hollywood runtime | Low — standard actor pattern | End-to-end pipeline test |
| Consumer ↔ Inserter wiring | Requires NATS + actor runtime | Low — declarative in pipeline.go | Smoke tests cover this path |
| Pipeline enable/disable logic | Config-dependent | Low — settings already tested | Existing settings_test.go |
| Timer-based flush | Requires actor engine | Low — trivial AfterFunc | Manual verification via smoke tests |
