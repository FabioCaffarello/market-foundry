# Production Wiring Items — Exit Criteria and Non-Goals

> **Tranche:** PWT-1 (Production Wiring Tranche)
> **Charter:** S327
> **Date:** 2026-03-21
> **Status:** Active

---

## 1. Item Registry

### PWT-1: RetrySubmitter Wiring

**Objective:** Compose `RetrySubmitter` around `BinanceFuturesTestnetAdapter` in
the execute supervisor bootstrap so that venue submissions pass through the
retry loop in the production actor path.

**Current state:** `VenueAdapterActor` receives the raw adapter via its config.
The `RetrySubmitter` exists, is tested (23 tests), but is not composed in the
bootstrap.

**Target composition:**
```go
retrySubmitter := execution.NewRetrySubmitter(adapter, execution.DefaultRetryPolicy())
```

**Exit criteria:**
1. `execute_supervisor.go` constructs a `RetrySubmitter` wrapping the venue adapter.
2. The `VenueAdapterActor` receives the `RetrySubmitter` as its `VenuePort`.
3. Existing unit tests for `RetrySubmitter` continue to pass (23 tests).
4. No new retry policy parameters introduced; `DefaultRetryPolicy()` used.

**Deferred gap closure:** This item closes the production-path portion of the
retry loop that was unit-tested in S319 and hardened in S323.

---

### PWT-2: Post200Reconciler Wiring

**Objective:** Compose `Post200Reconciler` around the `RetrySubmitter` in
bootstrap so that body-read-failure-after-200 events trigger automatic recovery
via `QueryOrder` in the production actor path.

**Current state:** `Post200Reconciler` exists, is tested (9 tests including
composition test with `RetrySubmitter`), but is not composed in the bootstrap.

**Target composition:**
```go
reconciler := execution.NewPost200Reconciler(retrySubmitter, adapter, 10*time.Second)
```

**Exit criteria:**
1. `execute_supervisor.go` constructs a `Post200Reconciler` wrapping the `RetrySubmitter`.
2. The reconciler receives the raw adapter as its `QueryPort` (for `QueryOrder`).
3. The reconciler receives an independent deadline (10s) for recovery queries.
4. The `VenueAdapterActor` receives the reconciler (not the retry submitter) as its `VenuePort`.
5. Existing unit tests for `Post200Reconciler` continue to pass (9 tests).
6. INV-REC-1 (no duplicate execution) preserved: reconciler uses `QueryOrder` (GET), not `SubmitOrder`.

**Deferred gap closure:** This item closes the production-path portion of the
reconciliation capability delivered in S322.

---

### PWT-3: Observability Hook Wiring

**Objective:** Wire `WithHaltChecker`, `WithLogger`, and `WithTracker` on the
`RetrySubmitter` in bootstrap so that the production retry loop checks the kill
switch between attempts and produces structured observability signals.

**Current state:**
- `WithHaltChecker` exists and is tested (S323) but not wired to the control store in bootstrap.
- `WithLogger` exists and is tested (S324) but not wired to the actor's logger in bootstrap.
- `WithTracker` exists and is tested (S324) but not wired to the actor's health tracker in bootstrap.

**Target composition:**
```go
retrySubmitter := execution.NewRetrySubmitter(adapter, execution.DefaultRetryPolicy()).
    WithHaltChecker(controlStore).
    WithLogger(logger).
    WithTracker(adapterTracker)
```

**Exit criteria:**
1. `RetrySubmitter` is constructed with `.WithHaltChecker(controlStore)` in bootstrap.
2. `RetrySubmitter` is constructed with `.WithLogger(logger)` in bootstrap.
3. `RetrySubmitter` is constructed with `.WithTracker(tracker)` in bootstrap.
4. The `controlStore` argument is the same execution control gate used by the `SafetyGate`.
5. INV-OBS-1 preserved: first-attempt success produces zero retry signals.
6. INV-OBS-2 preserved: nil-safe fallback still works (tested, not broken by wiring).
7. Existing unit tests for observability hooks continue to pass (6 tests).

**Deferred gap closure:** This item closes R-S323-3 (WithHaltChecker not wired
in actor pipeline) and R-S324-1 (WithLogger/WithTracker not wired in bootstrap).

---

### PWT-4: Integration Verification

**Objective:** Verify the fully composed pipeline through an actor-level
integration test that exercises the complete decorator chain with a mock venue.

**Current state:** Individual components are tested in isolation. The actor has
tests. But no test exercises the full composition chain as wired in the
production bootstrap.

**Exit criteria:**
1. At least one test constructs the full composition chain (adapter → RetrySubmitter → Post200Reconciler) and passes it to `VenueAdapterActor`.
2. The test exercises a successful submit path through the composed chain.
3. The test exercises at least one retry path (transient failure → retry → success).
4. The test exercises at least one reconciliation path (body-read-failure-after-200 → QueryOrder → recovery).
5. All 9 invariants verified in the composed path.
6. Test suite total remains at 186+ with 0 failures.
7. No regressions in any existing test file.

---

## 2. Exit Criteria Summary Matrix

| Item | Key Exit Criterion | Verification Method |
|------|--------------------|---------------------|
| PWT-1 | RetrySubmitter wraps adapter in bootstrap | Code review of `execute_supervisor.go` |
| PWT-2 | Post200Reconciler wraps RetrySubmitter in bootstrap | Code review of `execute_supervisor.go` |
| PWT-3 | WithHaltChecker/Logger/Tracker wired in bootstrap | Code review of `execute_supervisor.go` |
| PWT-4 | Full chain exercised in integration test | Test execution: retry + reconciliation paths pass |

---

## 3. Tranche Gate Criteria (S331)

The tranche gate (S331) verifies all of the following:

| # | Gate criterion | Evidence |
|---|---------------|----------|
| 1 | All 4 PWT items completed | Stage reports S328, S329, S330 |
| 2 | Test suite ≥ 186 tests, 0 failures | `go test ./...` output |
| 3 | All 9 invariants preserved | Invariant checklist in S331 report |
| 4 | No new interfaces introduced | Diff review |
| 5 | No scope inflation | Charter item count unchanged (4) |
| 6 | Retry metadata flows in actor logs | Integration test log assertions or code review |
| 7 | No changes to retry policy parameters | Diff review |

---

## 4. Non-Goals (Explicit)

### 4.1 Capability Non-Goals

| Non-Goal | Boundary | Rationale |
|----------|----------|-----------|
| OMS / order lifecycle management | Not a wiring task | Requires dedicated charter per S309 |
| Multi-venue adapter (Bybit, dYdX, etc.) | Different capability wave | Only Binance Futures testnet in scope |
| Mainnet activation | Different risk profile | Requires independent risk review and authorization |
| New signal families or domain capabilities | Different wave | Production wiring is infrastructure, not domain |
| Dashboard or monitoring UI | Consumer concern | Hooks produce counters; visualization is downstream |

### 4.2 Design Non-Goals

| Non-Goal | Boundary | Rationale |
|----------|----------|-----------|
| Supervisor redesign | Architecture change | Actor and supervisor shapes are stable |
| New interfaces or ports | Design change | All interfaces already exist and are tested |
| New configuration knobs | Scope inflation | Existing options are sufficient |
| Retry policy tuning | Parameter change | DefaultRetryPolicy is tested and adequate for testnet |
| Per-error-class differentiated policies | Future design | R-S320-6: requires production evidence |
| Retry-After header extraction | Future enhancement | R-S325-2: exponential backoff sufficient |

### 4.3 Process Non-Goals

| Non-Goal | Boundary | Rationale |
|----------|----------|-----------|
| CI/CD pipeline changes | Infrastructure change | Existing CI sufficient for verification |
| New documentation categories | Governance overhead | Existing architecture + stages docs sufficient |
| Refactoring of existing test files | Scope creep | Tests are passing; wiring does not require test refactoring |
| New wave or charter opening | Ceremony inflation | This tranche is self-contained; next wave is a separate decision |

---

## 5. Escalation Protocol

If any wiring step reveals a gap that prevents composition using existing
interfaces:

1. **Stop.** Do not solve the gap inline.
2. **Document** the gap with a unique ID (e.g., `R-PWT-1`).
3. **Classify** as blocking or deferrable.
4. If **blocking**: pause the tranche, open a dedicated resolution stage.
5. If **deferrable**: add to accepted gaps, continue wiring with a documented workaround.
6. **Report** in the stage report with full traceability.

This protocol prevents the tranche from silently expanding into design work.
