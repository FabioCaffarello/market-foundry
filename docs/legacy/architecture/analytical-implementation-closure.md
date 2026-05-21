# Analytical Implementation Closure

> S206 — Comprehensive closure of the analytical layer before the next phase.

## Purpose

This document records the closure actions taken on the analytical path
(writer, reader, gateway, migrations, diagnostics, smoke, CI) to ensure
no critical implementation gaps remain before the project moves to the
next phase (S207 generated-path decision).

## Scope

The closure covers six areas:

1. **Writer service** — pipeline, supervisor, consumer, inserter, mappers
2. **Read path** — ClickHouse adapters, analytical use cases, HTTP handlers/routes
3. **Gateway composition** — analytical wiring, optional ClickHouse lifecycle
4. **Migrations & schema** — cmd/migrate, internal/migrate, DDL catalog
5. **Smoke, CI & diagnostics** — scripts, CI workflow, .http test files
6. **Configuration & tooling** — settings schema, config files, .gitignore

## Actions Taken

### 1. Compile-Time Interface Assertions (Reader)

**File:** `cmd/gateway/analytical_reader_test.go`

Added missing compile-time assertions for 4 of 6 readers that were
absent. All 6 ClickHouse adapters now have compile-time proof that they
satisfy the `analyticalclient.*Reader` interfaces:

- CandleReader (existed)
- SignalReader (existed)
- DecisionReader (added)
- StrategyReader (added)
- RiskReader (added)
- ExecutionReader (added)

### 2. Binary Artifact Exclusion

**File:** `.gitignore`

The compiled binary `cmd/writer/writer` was staged in git. Added
gitignore rules for `/writer`, `/migrate`, and `cmd/*/writer`,
`cmd/*/migrate` patterns. The binary was unstaged.

### 3. Writer in Health/Diagnostics Scripts

**File:** `scripts/live-pipeline-activate.sh`

- Added `writer` to the Phase 2 health wait loop (was missing)
- Added `writer:8085` to Phase 3 internal readiness probes
- Added `writer:8085` to Phase 5 diagnostics checks
- Added `writer:8085` to Phase 8 tracker activity summary

### 4. Analytical Endpoint Validation in Live Pipeline

**File:** `scripts/live-pipeline-activate.sh`

Added Phase 6 validation of all 6 `/analytical/*` endpoints per symbol:
- `/analytical/evidence/candles`
- `/analytical/signal/history`
- `/analytical/decision/history`
- `/analytical/strategy/history`
- `/analytical/risk/history`
- `/analytical/execution/history`

Previously, only operational (NATS KV) endpoints were validated in the
live pipeline script. The analytical (ClickHouse) read path was only
tested by `smoke-analytical-e2e.sh` but not by the live activation flow.

### 5. Gateway Port in Shared Library

**File:** `scripts/utils/lib.sh`

Added `[gateway]=8080` to the `SVC_PORTS` map. Gateway was listed in
`ALL_SERVICES` but had no port entry, making `svc_port gateway` return
the fallback value silently.

## Architectural Invariants Preserved

- **Optionality:** ClickHouse remains optional for the gateway. R-02
  compliance (readiness does not depend on ClickHouse) is unchanged.
- **Boundaries:** Writer owns write-path; adapter owns storage↔domain
  translation; use case owns validation; handler owns HTTP concerns.
- **Graceful degradation:** Analytical endpoints return 503 when
  ClickHouse is unavailable; gateway continues serving operational path.
- **Runtime activation:** All families remain opt-in via pipeline config.
  No new families were introduced.

## What This Closure Does NOT Do

- Does not upgrade clickhouse-go version alignment (v2.30.0 vs v2.43.0)
  — this is a conscious freeze documented in the open-items inventory.
- Does not add transaction wrapping to migration runner — ClickHouse has
  limited transaction support; this is a known constraint.
- Does not introduce dead-letter queues or backpressure in the writer —
  these are post-closure hardening items.
- Does not expand the codegen integration or add new families.
