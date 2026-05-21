# Live Smoke and Operational Verification After Production Wiring

> **Stage:** S330
> **Date:** 2026-03-21
> **Type:** Operational verification architecture
> **Phase:** 31b (Production Wiring Tranche)

---

## Purpose

This document describes the operational smoke verification model used after
the production wiring tranche (S327-S329) to confirm that the final composed
venue pipeline remains functional, auditable, and stable.

The smoke does not introduce new functional capabilities. It reconfirms
existing composed capabilities via a single, reproducible entrypoint.

---

## Smoke Architecture

### Entrypoint

```
make smoke-composed       # canonical target
scripts/smoke-composed-pipeline.sh   # direct script
```

### Prerequisites

- Go 1.23+ toolchain installed
- No running stack, NATS, or ClickHouse required

### Phase Structure

| Phase | Description | Tests | Stage Origin |
|-------|-------------|-------|--------------|
| 1 | Build verification | `go vet` on execution + actor packages | Baseline |
| 2 | Supervisor composition | SC-01..SC-07 | S328 |
| 3 | Venue path verification | VP-01..VP-09 | S329 |
| 4 | Error code classification | EC-S325-1..EC-S325-10 | S325 |
| 5 | Full regression gate | All execution package tests | Cumulative |

### Exit Criteria

- All 5 phases report PASS
- Zero regressions in full suite (Phase 5)
- Single exit code: 0 = PASS, 1 = FAIL

---

## Composed Pipeline Under Verification

```
VenueAdapterActor.onIntent()
  |
  +-- Safety gate: kill switch + staleness guard
  +-- Call a.venue.SubmitOrder()  <-- composed pipeline
  |     |
  |     +-- Post200Reconciler (outermost)
  |     |     +-- RetrySubmitter (middle)
  |     |           +-- .WithHaltChecker
  |     |           +-- .WithLogger
  |     |           +-- .WithTracker
  |     |           +-- rawAdapter (innermost)
  |     |
  |     +-- On error: extract retry metadata -> structured log
  |
  +-- Construct VenueOrderFilledEvent
  +-- Publish fill event (NATS)
  +-- Track filled counters
```

### Assembly Site

`VenueAdapterActor.start()` in `internal/actors/scopes/execute/venue_adapter_actor.go` (lines 105-141).

### Decorator Order Invariants

1. **INV-DO-1:** Retry wraps the raw adapter, not the reconciler
2. **INV-DO-2:** Reconciler wraps the retry layer, not the raw adapter
3. **INV-DO-3:** Safety gate is outside the decorator chain
4. **INV-DO-4:** Halt checker operates at two levels independently

---

## Operational Verification Model

### What This Smoke Proves

| Capability | Proven By | Evidence |
|-----------|-----------|----------|
| Decorator composition correct | SC-01..SC-07 | Full stack interplay, happy path, halt, paper mode |
| Venue path end-to-end | VP-01..VP-09 | Retry recovery, post-200 recovery, field preservation |
| Safety gate independence | VP-09 | Staleness + kill switch block before decorators |
| Error classification accuracy | EC-S325-1..10 | Venue-specific HTTP+code mapping |
| No regressions | Phase 5 full suite | Entire execution package passes |

### What This Smoke Does Not Prove

| Excluded Scope | Reason |
|---------------|--------|
| Live NATS publish | Requires stack; covered by `make smoke-live-stack` |
| Real venue HTTP call | Requires credentials; covered by `make smoke-venue` |
| ClickHouse persistence | Requires stack; covered by `make smoke-round-trip` |
| Gateway composite surface | Requires stack; covered by `make smoke-live-stack` |

---

## Relationship to Other Smokes

| Smoke Target | Stack Required | Scope |
|-------------|----------------|-------|
| `make smoke` | Yes | Baseline single-symbol E2E |
| `make smoke-live-stack` | Yes | Live stack + gateway + NATS + ClickHouse |
| `make smoke-venue` | Credentials | Real venue integration |
| `make smoke-composed` | **No** | Composed pipeline at application layer |
| `make smoke-round-trip` | Yes | Full persistence round-trip |

The composed pipeline smoke (`smoke-composed`) is the only smoke that
validates the production-wired decorator pipeline without infrastructure
dependencies. It is the fastest feedback loop for pipeline integrity.

---

## Guard Rails

- No stack dependency introduced
- No new test inflation beyond existing suites
- No dashboard or alerting surface
- Single script, single exit code
- Run time: ~35 seconds (dominated by retry backoff in full suite)
