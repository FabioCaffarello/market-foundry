# Execute Supervisor: Composition of Retry, Reconciler, and Observability

> **Stage:** S328
> **Date:** 2026-03-21
> **Scope:** Mechanical composition of tested decorators into the production pipeline
> **Predecessor:** S327 (Production Wiring Tranche Charter)

---

## Overview

This document describes how the RetrySubmitter, Post200Reconciler, and observability
hooks are composed into the venue adapter actor's submit pipeline. The composition
is mechanical: no new interfaces, no new retry semantics, no new reconciliation logic.
Each decorator was implemented and tested in isolation during the venue progression
(S319, S322-S325); this stage wires them together.

---

## Composition Site

The composition happens in `VenueAdapterActor.start()` inside
`internal/actors/scopes/execute/venue_adapter_actor.go`.

This location was chosen because:

1. The control store (needed for `WithHaltChecker`) is created here.
2. The safety gate is already assembled here.
3. The raw venue adapter is available via `cfg.Venue`.
4. The tracker and logger are available for hook attachment.

The composed venue is stored in `a.venue` and used by `onIntent()` for all
submit calls, replacing the previous direct use of `a.cfg.Venue`.

---

## Composition Stack

```
VenueAdapterActor.onIntent()
  |
  |  Gate 1+2: SafetyGate.Check() [kill switch + staleness]
  |
  v
  a.venue.SubmitOrder()
  = Post200Reconciler.SubmitOrder()           [outer layer]
    |
    |  Intercepts body-read-failure-after-200
    |  Queries venue via VenueQueryPort
    |
    v
    RetrySubmitter.SubmitOrder()               [middle layer]
      .WithHaltChecker(controlStore)           [PWT-3]
      .WithLogger(logger)                      [PWT-3]
      .WithTracker(tracker)                    [PWT-3]
      |
      |  Retries retryable failures with backoff, deadline, halt check
      |
      v
      rawAdapter.SubmitOrder()                 [innermost layer]
      = BinanceFuturesTestnetAdapter
```

### Why This Order

| Layer | Position | Rationale |
|-------|----------|-----------|
| rawAdapter | Innermost | The actual venue HTTP call |
| RetrySubmitter | Middle | Retries transient failures before they surface |
| Post200Reconciler | Outermost | body-read-failure-after-200 is **non-retryable** (venue already accepted), so it passes through RetrySubmitter and is caught here |

If the order were reversed (reconciler inside retry), the reconciler would
attempt recovery on every retry iteration, which is incorrect: the retry loop
should only see the raw adapter's responses.

---

## Bootstrap Flow

### `cmd/execute/run.go`

```go
venueResult := buildVenueAdapter(config)
// venueResult.submit = rawAdapter (VenuePort)
// venueResult.query  = rawAdapter (VenueQueryPort, nil for paper)

NewExecuteSupervisor(config, venueResult.submit, venueResult.query, trackers)
```

### `execute_supervisor.go`

Passes both ports through to `VenueAdapterConfig`:

```go
VenueAdapterConfig{
    Venue:      s.venue,       // raw VenuePort
    VenueQuery: s.venueQuery,  // raw VenueQueryPort (or nil)
    ...
}
```

### `venue_adapter_actor.go` — `start()`

```go
rawVenue := a.cfg.Venue

// 1. Wrap with RetrySubmitter
retrySubmitter := NewRetrySubmitter(rawVenue, DefaultRetryPolicy())
retrySubmitter.WithHaltChecker(gateChecker)  // if available
retrySubmitter.WithLogger(logger)
retrySubmitter.WithTracker(tracker)          // if available

// 2. Wrap with Post200Reconciler (if query port available)
composedVenue := retrySubmitter
if a.cfg.VenueQuery != nil {
    composedVenue = NewPost200Reconciler(retrySubmitter, a.cfg.VenueQuery, 0)
}

a.venue = composedVenue
```

---

## Observability Hooks (PWT-3)

| Hook | Attached To | Purpose |
|------|-------------|---------|
| `WithHaltChecker(controlStore)` | RetrySubmitter | Checks kill switch between retry attempts |
| `WithLogger(logger.With("component", "retry-submitter"))` | RetrySubmitter | Structured log events for retry lifecycle |
| `WithTracker(tracker)` | RetrySubmitter | Health counters: `retry_attempts`, `retry_exhausted`, `retry_halted`, `retry_deadline_exceeded`, `retry_success_after_retry` |

All hooks are nil-safe: they degrade gracefully when the component is unavailable
(e.g., control store fails to connect, tracker is nil).

---

## Startup Log

The venue adapter startup log now includes composition state:

```
venue adapter started
  staleness_max_age=2m0s
  submit_timeout=10s
  control_gate=true
  retry_submitter=true
  retry_halt_checker=true
  post200_reconciler=true
```

This makes the active decorator stack visible in operational logs without
requiring configuration inspection.

---

## Adapter-Specific Behavior

| Adapter | VenueQuery | Reconciler Active | Retry Active |
|---------|-----------|-------------------|-------------|
| PaperVenueAdapter | nil | No | Yes |
| BinanceFuturesTestnetAdapter | self | Yes | Yes |

The paper adapter has no query capability, so the reconciler is skipped.
RetrySubmitter is always active (paper failures are not expected to be
retryable, but the decorator is harmless as a no-op pass-through).

---

## Invariant Preservation

All 9 invariants tracked since S308 remain preserved:

| ID | Invariant | Impact |
|----|-----------|--------|
| EC-1 | Deterministic client order ID | Unchanged — ID generation is in the adapter |
| EC-3 | Per-request deadline | Unchanged — each layer has its own deadline |
| F-1 | No bare errors / Problem type | Unchanged — all decorators use Problem |
| F-4 | Credential redaction | Unchanged — adapter internals unchanged |
| RF-1 | Retryable flag accuracy | Unchanged — classification is in the adapter |
| PGR-08 | Intent immutability | Unchanged — decorators don't modify intents |
| INV-REC-1 | No duplicate execution | Unchanged — reconciler uses GET, not POST |
| INV-RC-1 | Deadline independence | Preserved — reconciler uses fresh context |
| INV-OBS-1 | Zero noise on success | Preserved — hooks are nil-safe, tested |
