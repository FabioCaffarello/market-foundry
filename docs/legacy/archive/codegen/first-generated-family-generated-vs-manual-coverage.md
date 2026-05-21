# First Generated Family — Generated vs Manual Coverage

> Stage: S202 — First Codegen-First Family Definition
> Family: EMA (signal layer)
> Date: 2026-03-20

## Coverage Matrix

| Artifact | ID | Generated? | Owner | Rationale |
|----------|-----|-----------|-------|-----------|
| Consumer spec function | A1 | **Yes** | Machine | Fully derivable from spec; template-driven; zero creative content |
| Pipeline entry struct | A2 | **Yes** | Machine | Fully derivable from spec; template-driven; references only spec fields and derived names |
| Row mapper function | A3 | No — **Reused** | Human | `mapSignalRow` already exists and handles all `SignalGeneratedEvent` instances; no new mapper needed |
| Mapper unit tests | A4 | No — **Reused** | Human | Existing `mapSignalRow` tests cover the shared mapper |
| Config entry | A5 | No — Manual | Human | Operational decision: add `"ema"` to `signal_families` array in `writer.jsonc` |
| Smoke test coverage | A6 | No — Manual | Human | Assertion logic requires domain knowledge about expected EMA behavior |

## Generated Artifacts — Detail

### A1: Consumer Spec (Generated)

- **Source:** `codegen/families/ema.yaml` → `codegen/templates/consumer_spec.go.tmpl`
- **Output:** `codegen/golden-snapshots/ema/consumer_spec.go.golden`
- **Target:** `internal/adapters/nats/signal_registry.go`
- **Governance:** `codegen:begin consumer_spec family=ema` / `codegen:end`
- **Content:** Function `WriterEMASignalConsumer()` returning durable consumer configuration
- **Why generated:** Every field is a direct copy from spec YAML; zero interpretation needed

### A2: Pipeline Entry (Generated)

- **Source:** `codegen/families/ema.yaml` → `codegen/templates/pipeline_entry.go.tmpl`
- **Output:** `codegen/golden-snapshots/ema/pipeline_entry.go.golden`
- **Target:** `cmd/writer/pipeline.go`
- **Governance:** `codegen:begin pipeline_entry family=ema` / `codegen:end`
- **Content:** Struct literal wiring consumer spec → mapper → inserter actor
- **Why generated:** All references are deterministic: derived names, spec fields, existing factory functions

## Manual Artifacts — Detail

### A3: Mapper (Reused — No New Code)

- **Existing function:** `mapSignalRow(signal.SignalGeneratedEvent) []any`
- **Location:** `cmd/writer/mappers.go`
- **Why reused:** All signal families emit `SignalGeneratedEvent`; the mapper extracts the same 12 columns regardless of which signal family produced the event
- **Why not generated:** Mapper contains domain knowledge (column ordering, type conversions, JSON marshaling); this is a human decision

### A4: Mapper Tests (Reused — No New Code)

- **Existing tests:** `cmd/writer/mappers_test.go`
- **Why reused:** Tests for `mapSignalRow` already cover the shared mapping logic
- **Note:** If EMA events have different field distributions (e.g., different metadata keys), the existing tests still cover the mapper's structural correctness. Field-level validation is a domain concern, not a codegen concern.

### A5: Config Entry (Manual — New)

- **File:** `deploy/configs/writer.jsonc`
- **Change:** Add `"ema"` to the `signal_families` array
- **Why manual:** Config determines which families are active at runtime; this is an operational decision, not a structural one
- **Effort:** Single line addition

### A6: Smoke Test (Manual — New)

- **Scope:** Verify EMA pipeline activates, consumes events, and writes to `signals` table
- **Location:** `scripts/smoke-analytical-e2e.sh` (extend existing)
- **Why manual:** Smoke assertions encode expected runtime behavior; codegen has no knowledge of what constitutes a valid EMA signal event
- **Effort:** Small — adapts existing RSI smoke pattern for EMA subject/durable

## Infrastructure Reuse Summary

The EMA family requires **zero new infrastructure**:

| Infrastructure | Status | Shared With |
|---------------|--------|-------------|
| `SIGNAL_EVENTS` NATS stream | Exists | RSI |
| `signals` ClickHouse table | Exists | RSI |
| `NewSignalConsumer` factory | Exists | RSI |
| `reg.signal` registry | Exists | RSI |
| `mapSignalRow` mapper | Exists | RSI |
| `IsSignalFamilyEnabled` config method | Exists | RSI |
| Signal reader/handler/route | Exists | RSI |

## Coverage Boundary Diagram

```
┌─────────────────────────────────────────────────────────┐
│  EMA Signal Family — Artifact Ownership                 │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌───────────────────────────────────┐                  │
│  │  GENERATED (Machine-Owned)        │                  │
│  │                                   │                  │
│  │  A1: WriterEMASignalConsumer()    │                  │
│  │  A2: pipeline entry { ema ... }   │                  │
│  │                                   │                  │
│  │  Source: ema.yaml → templates     │                  │
│  │  CI: golden + marker governance   │                  │
│  └───────────────────────────────────┘                  │
│                                                         │
│  ┌───────────────────────────────────┐                  │
│  │  REUSED (Existing Human-Owned)    │                  │
│  │                                   │                  │
│  │  A3: mapSignalRow (shared)        │                  │
│  │  A4: mapper tests (shared)        │                  │
│  │  Infrastructure: stream, table,   │                  │
│  │    consumer factory, registry,    │                  │
│  │    reader, handler, route         │                  │
│  └───────────────────────────────────┘                  │
│                                                         │
│  ┌───────────────────────────────────┐                  │
│  │  NEW MANUAL (Human-Authored)      │                  │
│  │                                   │                  │
│  │  A5: config entry ("ema")         │                  │
│  │  A6: smoke test extension         │                  │
│  └───────────────────────────────────┘                  │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

## What Is Explicitly NOT Generated

| Artifact | Reason |
|----------|--------|
| Domain event type (`SignalGeneratedEvent`) | Shared type; already exists; contains domain semantics |
| NATS stream definition (`SIGNAL_EVENTS`) | Infrastructure concern; already exists |
| ClickHouse DDL (`signals` table) | Schema concern; already exists |
| Consumer factory (`NewSignalConsumer`) | Contains NATS wiring logic; shared across signal families |
| Event registry (`reg.signal`) | Runtime infrastructure; shared |
| Reader adapter (`signal_reader.go`) | Read-path; Tier 2 only |
| HTTP handler / route | Gateway concern; out of writer codegen scope |
| Mapper function | Domain knowledge; shared with RSI |

## Delta From RSI Governed Slice

Since RSI (S200) is already governed in the same layer, the EMA family proves additive scaling:

| Dimension | RSI (S200) | EMA (S203) |
|-----------|-----------|-----------|
| Spec YAML | New | New |
| Golden snapshots | New | New |
| Consumer spec function | New (generated) | New (generated) |
| Pipeline entry | New (generated) | New (generated) |
| Manifest entries | 2 new | 2 new |
| Mapper | Existing manual | Reused (zero new code) |
| Config | Existing | 1 line addition |
| Smoke | Existing | Small extension |
| DDL | Existing | Reused |
| Infrastructure | Existing | Reused |
