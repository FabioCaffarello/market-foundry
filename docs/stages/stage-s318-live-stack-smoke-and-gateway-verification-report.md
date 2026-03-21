# Stage S318: Live Stack Smoke and Gateway Verification — Report

**Status:** DELIVERED
**Predecessor:** S317 (Full Persistence Round-Trip)
**Phase:** 31 — Implementation Wave

---

## Objective

Transform the end-to-end proofs from S316–S317 into a single, reproducible
operational smoke that validates the full live stack: venue path, analytical
persistence, and gateway composite read surface.

This stage closes the operability gap identified by S317: while S317 proved
the round-trip works structurally, it left continuous operational verification
as a manual multi-step process.

## Deliverables

| Deliverable | Path | Status |
|------------|------|--------|
| Smoke script | `scripts/smoke-live-stack.sh` | DELIVERED |
| Makefile target | `make smoke-live-stack` | DELIVERED |
| Architecture doc | `docs/architecture/live-stack-smoke-and-gateway-verification.md` | DELIVERED |
| Operational guide | `docs/architecture/stack-smoke-operational-usage-prerequisites-and-limitations.md` | DELIVERED |
| Stage report | `docs/stages/stage-s318-live-stack-smoke-and-gateway-verification-report.md` | DELIVERED |
| Stage index update | `docs/stages/INDEX.md` | DELIVERED |
| Smoke-help update | `Makefile` (smoke-help target) | DELIVERED |

## What Changed

### New Files

| File | Purpose |
|------|---------|
| `scripts/smoke-live-stack.sh` | Single-command live stack smoke with 6 validation phases |
| `docs/architecture/live-stack-smoke-and-gateway-verification.md` | Design and rationale document |
| `docs/architecture/stack-smoke-operational-usage-prerequisites-and-limitations.md` | Operator guide with prerequisites, variables, troubleshooting |

### Modified Files

| File | Change |
|------|--------|
| `Makefile` | Added `smoke-live-stack` target, updated `.PHONY`, updated `smoke-help` |
| `docs/stages/INDEX.md` | Added S318 entry to Phase 31 |

## Smoke Design

### Six Phases

| Phase | Validates | Failure mode |
|-------|-----------|-------------|
| 1. Stack Readiness | ClickHouse, writer, gateway, NATS health | `die` (blocks further phases) |
| 2. NATS Streams | EXECUTION_FILL_EVENTS, EXECUTION_EVENTS, venue-fill consumer | WARN if absent |
| 3. ClickHouse Data | Row counts in all 6 analytical tables | WARN if empty |
| 4. Composite Surface | `/composite/chains`, `/composite/funnel`, `/composite/dispositions` | FAIL on non-200 |
| 5. Family Endpoints | All 6 `/analytical/{family}/*` endpoints | FAIL on non-200 |
| 6. Structural Tests | S317 Go round-trip tests | FAIL on test failure |

### Coverage Matrix

| Path Segment | Validated By |
|-------------|-------------|
| Venue adapter → NATS | Phase 2 (stream/consumer existence) |
| NATS → writer → ClickHouse | Phase 3 (table row counts) |
| ClickHouse → CompositeReader → HTTP | Phase 4 (composite endpoints) |
| ClickHouse → single-family reader → HTTP | Phase 5 (analytical endpoints) |
| Mapper/reader contracts | Phase 6 (Go structural tests) |

## Acceptance Criteria Evaluation

| Criterion | Verdict |
|-----------|---------|
| Exists a reproducible and useful smoke | PASS — `make smoke-live-stack` |
| Gateway + ClickHouse + venue path exercised together | PASS — phases 2-5 |
| Reduces operational verification cost | PASS — single command replaces multi-smoke manual flow |
| Smoke remains small and disciplined | PASS — ~280 lines, 6 focused phases, no data injection |

## Guard Rail Compliance

| Guard rail | Status |
|-----------|--------|
| Not a production pipeline | COMPLIANT — stdout PASS/FAIL only |
| No dashboards | COMPLIANT |
| Not inflated into large suite | COMPLIANT — single script, focused scope |
| No undocumented manual steps | COMPLIANT — full operator guide delivered |

## Residual Limitations

| ID | Description | Severity | Notes |
|----|------------|----------|-------|
| R-S318-1 | Smoke does not inject synthetic data; relies on pipeline having produced events | MEDIUM | Acceptable: smoke validates operational state, not data generation |
| R-S318-2 | Single symbol/source only (btcusdt/binancef) | LOW | Use `smoke-multi-symbol` for broader coverage |
| R-S318-3 | No real venue credential validation in this smoke | LOW | Use `smoke-venue-integration` for testnet proof |
| R-S318-4 | Server-Timing headers not validated | LOW | Use `smoke-analytical` for observability checks |

## Relationship to Prior Stages

| Stage | Relationship |
|-------|-------------|
| S316 | Venue integration proof — S318 validates the infrastructure that S316 proved |
| S317 | Persistence round-trip — S318 subsumes S317's smoke scope and adds full endpoint coverage |
| S296 | Composite read model — S318 validates the composite HTTP surface |
| S297 | HTTP explainability surface — S318 exercises chains/funnel/dispositions |
| S298 | Attribution — S318 validates disposition breakdown endpoint |

## Recommendations for S319

1. **Synthetic data injection smoke**: Create a smoke that publishes a known
   event to NATS and traces it through ClickHouse to HTTP, proving causality
   without requiring live market data.
2. **CI integration**: Wire `make smoke-live-stack` into the CI pipeline after
   `make up && make seed` with appropriate wait times.
3. **Retry infrastructure**: The venue path still lacks production retry
   semantics (R-S316 residual); a future stage should address this.
4. **Multi-venue expansion**: Current proofs are single-venue (Binance Futures
   testnet); expansion to additional venues is deferred.
