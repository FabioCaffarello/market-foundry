# Stage S379 — Dry-Run Execution Path by Config

**Wave:** Exchange Listening and Dry-Run Foundation
**Predecessor:** S378 (Compose Live Exchange Listening Proof)
**Status:** COMPLETE

---

## Executive Summary

S379 delivers a first-class, config-governed dry-run execution path. The
`DryRunSubmitter` decorator intercepts all venue calls when `venue.dry_run`
is true (the default), producing auditable receipts without contacting any
real venue. The implementation is fail-closed: omitting the flag defaults to
dry-run. The pipeline preserves full correlation, causation, and activation
semantics.

## Objective

Design, implement, validate, and document a canonical dry-run path governed
by central configuration, in which execution intents traverse the real
pipeline but real venue submission is replaced by an auditable dry-run sink.

## Deliverables

### Code Changes

| File | Change |
|---|---|
| `internal/shared/settings/schema.go` | Added `DryRun *bool` to `VenueConfig`, `IsDryRun()` helper, validation for contradictory configs |
| `internal/application/execution/dry_run_submitter.go` | **New.** `DryRunSubmitter` VenuePort decorator with logging, counters, audit trail |
| `cmd/execute/run.go` | Wiring: compose `DryRunSubmitter` as outermost decorator when `IsDryRun()`, log `dry_run` in activation surface |
| `deploy/configs/execute.jsonc` | Added `"dry_run": true` with documentation comment |

### Tests

| File | Tests |
|---|---|
| `internal/application/execution/dry_run_submitter_test.go` | 6 unit tests: buy/sell/noop interception, no delegation, correlation preservation, unique IDs |
| `internal/actors/scopes/execute/s379_dry_run_config_test.go` | 4 config tests (fail-closed semantics), 1 validation test, 1 pipeline traversal test, 1 bomb-adapter test |

### Documentation

| Document | Content |
|---|---|
| `docs/architecture/dry-run-execution-path-by-config.md` | Config model, pipeline composition, interaction with activation surface, audit trail, limitations |
| `docs/architecture/dry-run-submitter-fail-closed-semantics-and-auditability.md` | Fail-closed properties (FC-8 through FC-11), auditability model, correlation preservation, configuration matrix |

## Test Evidence

### Unit Tests (DryRunSubmitter)

```
=== RUN   TestDryRunSubmitter_InterceptsBuyIntent         --- PASS
=== RUN   TestDryRunSubmitter_InterceptsSellIntent        --- PASS
=== RUN   TestDryRunSubmitter_InterceptsNoActionIntent    --- PASS
=== RUN   TestDryRunSubmitter_NeverDelegatesToInner        --- PASS
=== RUN   TestDryRunSubmitter_PreservesCorrelationFields   --- PASS
=== RUN   TestDryRunSubmitter_UniqueOrderIDs               --- PASS
```

### Config and Pipeline Tests (S379)

```
=== RUN   TestS379_DryRunConfig_FailClosed
    --- PASS: nil_dry_run_defaults_to_true
    --- PASS: explicit_true
    --- PASS: explicit_false_with_venue_adapter
    --- PASS: empty_type_defaults_to_dry-run
=== RUN   TestS379_DryRunConfig_ValidationRejectsPaperWithDryRunFalse  --- PASS
=== RUN   TestS379_DryRunSubmitter_PipelineTraversal                   --- PASS
=== RUN   TestS379_DryRunSubmitter_NeverCallsRealAdapter               --- PASS
```

### Full Test Suite

All existing tests pass. No regressions introduced.

## Fail-Closed Properties

| ID | Property | Mechanism |
|---|---|---|
| FC-8 | Default is dry-run | `IsDryRun()` returns true when `DryRun` is nil |
| FC-9 | Paper + dry_run=false rejected | Config validation rejects contradictory combo |
| FC-10 | DryRunSubmitter never delegates | `inner.SubmitOrder` never called; proven by bomb-adapter test |
| FC-11 | DryRunSubmitter is outermost | Composed last in decorator stack; intercepts before any inner layer |

## Acceptance Criteria Verification

| Criterion | Status |
|---|---|
| Canonical dry-run governed by configuration exists | PASS |
| Pipeline traversed without real trading request | PASS |
| Controls and explainability preserved | PASS |
| Ready for live-listen + dry-run end-to-end proof | PASS |

## Guard Rails Compliance

| Guard rail | Status |
|---|---|
| No OMS opened | PASS |
| No venue live trading as stage goal | PASS |
| No ad-hoc flags scattered in code | PASS — single `DryRun` field in `VenueConfig` |
| No invariants broken | PASS — full test suite green |

## Remaining Limitations

1. **Price realism.** Dry-run fills use `Price: "0"`. For realistic P&L
   simulation, a future stage could inject last-known market price.

2. **No runtime toggle.** Changing `dry_run` requires binary restart.
   A NATS KV-based runtime toggle was considered but deferred — it adds
   complexity and risk surface to a safety-critical flag.

3. **Prefix convention.** The `dryrun-` prefix is a string convention, not
   a type-level guarantee. Downstream consumers should filter by prefix
   or `Simulated` flag, not by type assertion.

4. **No latency simulation.** DryRunSubmitter returns instantly. For
   realistic latency modeling, a configurable delay could be added (same
   pattern as `PaperVenueAdapter.fillDelay`).

## Preparation for S380

S380 should deliver the end-to-end proof: **live-listen + dry-run pipeline**.

Prerequisites met by S379:
- Config-governed dry-run path (`venue.dry_run: true`)
- DryRunSubmitter wired as outermost decorator
- Activation surface logs dry_run state at startup
- Fill events flow through store and writer identically

Recommended S380 scope:
1. Start full compose stack with live exchange listening (S378) and
   `dry_run: true` (S379 default).
2. Activate bindings via configctl (seed).
3. Prove: market data flows from exchange through ingest→derive→execute,
   and execute produces `dryrun-` prefixed fills stored in NATS and
   materialized in store/writer.
4. Verify no real venue calls via absence of HTTP egress to testnet.
5. Document the end-to-end evidence as the wave closure gate.
