# Operational Smoke, CI, and Runbook Closure

> S208 — Consolidation of smoke tests, CI pipeline, diagnostics scripts,
> and runbook alignment with actual system behavior.

## Purpose

This document records the operational closure of the testing and
observability toolchain. It maps what each script/CI job covers, where
gaps existed and were closed, and what remains intentionally out of scope.

---

## 1. Smoke Test Coverage Map

### 1.1 `smoke-first-slice.sh` (make smoke)

**Scope:** Single-symbol NATS KV path (no ClickHouse, no writer).

| Dimension        | Coverage                       |
|------------------|-------------------------------|
| Symbols          | btcusdt only                  |
| Timeframes       | 60, 300, 900, 3600            |
| Families         | Evidence candles only         |
| Infrastructure   | NATS + gateway + pipeline     |
| Persistence      | NATS KV (not ClickHouse)      |
| Error handling   | Missing timeframe → 400       |

**When to use:** Quick validation of core pipeline without ClickHouse.

### 1.2 `smoke-multi-symbol.sh` (make smoke-multi)

**Scope:** Multi-symbol, multi-family, full NATS KV path.

| Dimension        | Coverage                       |
|------------------|-------------------------------|
| Symbols          | btcusdt + ethusdt             |
| Timeframes       | 60, 300, 900, 3600            |
| Families         | All 8 (evidence through exec) |
| Infrastructure   | NATS + gateway + all pipeline |
| Persistence      | NATS KV (not ClickHouse)      |
| Error handling   | Per-domain error cases        |

**When to use:** Comprehensive pipeline validation before analytical tests.

### 1.3 `smoke-analytical-e2e.sh` (make smoke-analytical)

**Scope:** Full analytical layer (ClickHouse + writer persistence + reader queries).

| Dimension        | Coverage                       |
|------------------|-------------------------------|
| Symbols          | btcusdt only                  |
| Timeframes       | 60 (primary)                  |
| Families         | All 6 analytical families     |
| Infrastructure   | Full stack + ClickHouse       |
| Persistence      | ClickHouse (write + read)     |
| Error handling   | Missing params, invalid limit, since > until |
| Filters          | outcome, direction, disposition, side, status |
| Observability    | Writer /diagz, degradation check |
| Error logs       | Compose log scan (added S208) |

**When to use:** Analytical layer validation, CI gate.

### 1.4 `live-pipeline-activate.sh` (make live / make live-check)

**Scope:** Full orchestration + comprehensive diagnostics.

| Dimension        | Coverage                        |
|------------------|---------------------------------|
| Symbols          | btcusdt (or +ethusdt with --multi-symbol) |
| Modes            | Full build, skip-build, check-only |
| Diagnostics      | /statusz + /diagz all services  |
| Error logs       | Compose error log scan          |
| Memory           | Container memory snapshot       |
| Trackers         | Per-tracker detail + counters   |

**When to use:** Full operational activation and diagnostics.

### 1.5 `diag-check.sh` (make diag)

**Scope:** Lightweight snapshot of running stack.

| Dimension        | Coverage                       |
|------------------|-------------------------------|
| Readiness        | All 6 pipeline services        |
| Phases           | /statusz per service           |
| Diagnostics      | /diagz per service             |
| Error logs       | Compose error log scan         |
| Modes            | Docker compose or --local      |

**When to use:** Quick health check without starting or seeding.

---

## 2. CI Pipeline Structure

```
┌─────────────┐   ┌────────────────────┐
│ unit-tests  │   │ codegen-golden     │   (parallel)
└──────┬──────┘   └────────────────────┘
       │
       ▼
┌──────────────────────┐
│ smoke-analytical     │   (depends on unit-tests)
│  1. build            │
│  2. compose up       │
│  3. wait ClickHouse  │
│  4. seed configctl   │
│  5. wait flush       │
│  6. smoke-analytical │
│  7. error log scan   │   ← Added S208
│  8. collect on fail  │
│  9. teardown         │
└──────────────────────┘
```

### CI jobs

| Job                | Trigger                  | What it validates                    |
|--------------------|--------------------------|--------------------------------------|
| `unit-tests`       | push/PR to main          | All Go module unit tests             |
| `codegen-golden`   | push/PR to main          | Spec validation + golden comparison + integrated slices |
| `smoke-analytical` | push/PR to main (after unit-tests) | Full analytical E2E path  |

### S208 CI changes

1. **Error log scanning step** — Added `always()` step that scans compose logs
   for `"level":"error"` entries. This makes error-level log noise visible in
   CI runs regardless of pass/fail, matching what `diag-check.sh` and
   `live-pipeline-activate.sh` already do locally.

2. **Configurable writer flush wait** — The `sleep 120` is now driven by
   `WRITER_FLUSH_WAIT` env var (default 120), making it adjustable if CI
   environments need longer flush times.

---

## 3. Runbook Alignment

### Existing runbooks and accuracy

| Document                                              | Status  | Notes |
|-------------------------------------------------------|---------|-------|
| `current-baseline-runbook.md`                         | Current | Stack lifecycle, seeding, recovery procedures |
| `current-baseline-operational-diagnostics.md`         | Current | Health endpoint surface, signal interpretation |
| `current-baseline-recovery-and-restart-semantics.md`  | Current | Shutdown, restart, durable consumer guarantees |
| `analytical-read-path-runbook-and-signal-interpretation.md` | Current | ClickHouse read path scenarios (404/503/0 rows/latency/scan errors) |
| `analytical-runtime-runbook-and-signal-interpretation.md`   | Current | Writer failure modes (degraded/overflow/stalled) |
| `analytical-config-and-startup-validation-hardening.md`     | Current | Startup validation design and coverage |

**Assessment:** All runbooks are accurate and describe the system's current
behavior. No contradictions or staleness detected. The runbook set covers:

- How to start, seed, and validate the stack
- How to diagnose health, phases, and tracker states
- How to interpret error signals and recover from failures
- What operational guarantees exist and what they do not cover

### What runbooks do NOT cover (explicitly out of scope)

- Performance tuning or capacity planning
- Multi-environment deployment (only local/compose)
- External monitoring/alerting integration
- Load testing procedures

---

## 4. Diagnostic Script Alignment

### S208 fix: `PIPELINE_SERVICES` in `lib.sh`

The shared `PIPELINE_SERVICES` array was missing `writer`. This meant that
any script iterating over pipeline services for `/statusz`/`/diagz` checks
would skip the writer — despite writer having both endpoints.

**Before:** `("configctl" "ingest" "derive" "store" "execute")`
**After:**  `("configctl" "ingest" "derive" "store" "execute" "writer")`

Note: `diag-check.sh` uses its own `RUNTIME_PORTS` array (which already
included writer), so this fix primarily affects future scripts that use
`PIPELINE_SERVICES` from `lib.sh`.

### Script-to-endpoint coverage

| Script               | /healthz | /readyz | /statusz | /diagz | Error logs |
|---------------------|----------|---------|----------|--------|------------|
| smoke-first-slice   | yes      | yes     | —        | —      | —          |
| smoke-multi-symbol  | yes      | yes     | —        | —      | —          |
| smoke-analytical    | —        | yes     | yes      | yes    | yes (S208) |
| live-pipeline       | —        | yes     | yes      | yes    | yes        |
| diag-check          | —        | yes     | yes      | yes    | yes        |

---

## 5. Coverage Gaps (Intentionally Not Closed)

| Gap                                  | Rationale                                    |
|--------------------------------------|----------------------------------------------|
| Multi-symbol analytical E2E          | Single symbol sufficient for CI gate proof   |
| Multi-timeframe analytical E2E       | 60s timeframe proves the path; others are    |
|                                      | structurally identical                        |
| Performance assertions in CI         | No latency SLAs defined; premature to enforce|
| Data correctness validation          | OHLCV math, signal value ranges out of scope |
| smoke-multi-symbol in CI             | Requires longer runtime; local-only for now  |
| Filter effectiveness tests           | Status code validated; result set filtering  |
|                                      | not verified (would need known data fixtures)|
| Codegen semantic validation          | Structural equivalence sufficient; semantic  |
|                                      | validation would require runtime execution   |

---

## 6. Test Execution Quick Reference

```bash
# Quick pipeline health check (no start/seed)
make diag

# Single-symbol smoke (NATS KV only, ~2 min)
make up && make seed && make smoke

# Multi-symbol smoke (NATS KV, ~3 min)
make up && make seed-multi && make smoke-multi

# Analytical E2E (full stack + ClickHouse, ~5 min)
make up && make seed && sleep 120 && make smoke-analytical

# Full live activation with diagnostics
make live

# Full live with multi-symbol
make live-multi

# CI gate (unit tests + analytical E2E)
make ci-analytical
```
