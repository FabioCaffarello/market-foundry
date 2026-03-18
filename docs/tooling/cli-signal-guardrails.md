# CLI Signal Domain Guardrails

## Purpose

This document defines the architectural guardrails that `raccoon-cli` enforces for the **signal** domain. Signal is the third processing layer (`observation → evidence → signal`) and must be governed with the same rigor as observation and evidence.

## Guardrails Overview

### 1. Signal Stream Ownership

**Rule:** SIGNAL_EVENTS must be declared in source and must be a canonical stream.

- Validated by: `runtime-bindings` (stream-ownership check) and `drift-detect` (stream-registry-drift)
- Severity: Error
- Rationale: Without SIGNAL_EVENTS in the canonical stream set, signal events have no transport and the pipeline is broken.

### 2. Signal Durable Consumer Binding

**Rule:** `store-signal-rsi` must exist and must be bound to `SIGNAL_EVENTS`.

- Validated by: `runtime-bindings` (consumer-binding check) and `drift-detect` (signal-contracts-drift)
- Severity: Error
- Rationale: The store service needs a durable consumer to read signal events for projection. Wrong stream binding causes silent data loss.

### 3. Signal Query Subject Presence

**Rule:** `signal.query.rsi.latest` must exist in source.

- Validated by: `runtime-bindings` (query-routing check) and `drift-detect` (signal-contracts-drift)
- Severity: Error
- Rationale: Gateway needs this subject to query store for latest RSI values. Missing subject breaks the HTTP → NATS request/reply chain.

### 4. Signal Adapter File Completeness

**Rule:** All five signal NATS adapter files must exist:

| File | Purpose |
|------|---------|
| `signal_registry.go` | Stream, consumer, and query specs |
| `signal_publisher.go` | Publishes to SIGNAL_EVENTS |
| `signal_consumer.go` | Durable consumer in store |
| `signal_gateway.go` | NATS request/reply gateway |
| `signal_kv_store.go` | KV bucket store for projections |

- Validated by: `runtime-bindings` (adapter-files check) and `drift-detect` (signal-adapter-drift)
- Severity: Warning (runtime-bindings), Error (drift-detect)

### 5. Signal Domain File Completeness

**Rule:** Signal domain, application, port, and HTTP layer files must exist.

- Validated by: `drift-detect` (signal-domain-drift)
- Severity: Error
- Files checked:
  - `internal/domain/signal/signal.go`
  - `internal/domain/signal/events.go`
  - `internal/application/signal/rsi_sampler.go`
  - `internal/application/signalclient/contracts.go`
  - `internal/application/signalclient/get_latest_signal.go`
  - `internal/application/ports/signal.go`
  - `internal/interfaces/http/handlers/signal.go`
  - `internal/interfaces/http/routes/signal.go`

### 6. Signal Actor Completeness

**Rule:** Signal actors must exist in both derive and store scopes.

- Validated by: `drift-detect` (signal-domain-drift)
- Severity: Error
- Actors checked:
  - `internal/actors/scopes/derive/signal_sampler_actor.go`
  - `internal/actors/scopes/derive/signal_publisher_actor.go`
  - `internal/actors/scopes/store/signal_consumer_actor.go`
  - `internal/actors/scopes/store/signal_projection_actor.go`

### 7. Signal Config Symmetry

**Rule:** `signal_families` must appear in both `derive.jsonc` and `store.jsonc`, or in neither.

- Validated by: `drift-detect` (signal-config-drift)
- Severity: Error for asymmetry, Warning for absence
- Rationale: Derive producing signals without store consuming them (or vice versa) is a configuration bug.

### 8. Signal Documentation Completeness

**Rule:** All canonical signal architecture docs must exist.

- Validated by: `drift-detect` (signal-docs-drift)
- Severity: Error
- Required docs: see [cli-signal-drift-rules.md](cli-signal-drift-rules.md) for the full list.

### 9. Signal KV Bucket Presence

**Rule:** `SIGNAL_RSI_LATEST` must appear in signal_kv_store.go source.

- Validated by: `drift-detect` (signal-contracts-drift)
- Severity: Error
- Rationale: Without the bucket definition, projection writes silently fail.

### 10. Signal Coverage Map

**Rule:** Changes to `internal/domain/signal/` trigger architecture, contracts, and drift validation.

- Validated by: `coverage-map` (domain-signal area)
- Dimensions required: architecture, contracts, drift

## Gate Integration

All signal guardrails run in the **Fast** quality gate profile — no infrastructure required. They are included in both `make check` and CI pipelines.

## What the CLI Cannot Yet Protect

| Gap | Reason |
|-----|--------|
| Signal sampler correctness | Requires Go test execution, not static analysis |
| Actor wiring correctness | No Go runtime inspection; validated by smoke tests only |
| KV monotonicity guard logic | Internal to Go; validated by unit tests |
| Multi-signal family consistency | Only RSI is checked; future families need constant additions |
| Signal-to-signal derivation | Not yet implemented; deferred |
