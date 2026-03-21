# Venue Progression Evidence Matrix, Residual Gaps, and Next Ceremony

> Stage: S326 | Date: 2026-03-21
> Companion to: [venue-progression-evidence-gate-after-closure-tranche.md](venue-progression-evidence-gate-after-closure-tranche.md)

## Evidence Matrix

### Per-Stage Evidence Summary

| Stage | Deliverable | Tests Added | Tests Total After | Regressions | Gaps Created | Gaps Closed |
|-------|------------|-------------|-------------------|-------------|-------------|-------------|
| S316 | Real testnet integration proof | 11 | 80+ | 0 | 4 (R-S316-1..4) | — |
| S317 | Writer consumer, persistence round-trip | 4 | 80+ | 0 | — | R-S316-1 |
| S318 | Smoke script (6 phases) | — | 80+ | 0 | — | — |
| S319 | RetrySubmitter decorator | 9 | 80+ | 0 | — | — |
| S320 | 19 failure path verifications | 19 | 80+ | 0 | 6 (R-S320-1..6) | — |
| S321 | Closure tranche charter | — | 80+ | 0 | — | — |
| S322 | Post200Reconciler + QueryOrder | 9 | 80+ | 0 | — | R-S320-1 |
| S323 | Deadline + halt check | 8 | 80+ | 0 | — | R-S320-2, R-S320-3 |
| S324 | Structured logs + counters | 6 | 80+ | 0 | — | R-S320-5 |
| S325 | Venue error code overrides | 10 | 80+ | 0 | — | R-S320-4 |

**Final test count (S326 gate run): 186 passing, 0 failing.**

### Per-Criterion Evidence Matrix

| Criterion | Evidence Level | Test Count | Key Test IDs |
|-----------|---------------|-----------|--------------|
| Round-trip (submit → fill → persist → read) | FULL | 15 | S316_VQ*, S317_VenueFill_* |
| Smoke (reproducible stack validation) | FULL | — | make smoke-live-stack (6 phases) |
| Retry loop (backoff, deadline, abort) | FULL | 23 | TestRetry_*, TestRetryObservability_* |
| Failure path (19 scenarios) | FULL | 19 | FP01–FP19 |
| Reconciliation (post-200 recovery) | FULL | 9 | RC-01–RC-09 |
| Observability (structured retry signals) | FULL | 6 | TestRetryObservability_* |
| Classification (venue error codes) | FULL | 10 (22 w/ subtests) | EC-S325-1 through EC-S325-10 |
| Safety gates (kill switch, staleness) | FULL | 35 | TestSafetyGate_*, TestStalenessGuard_* |
| Error classification (HTTP-based) | FULL | 19 | VA1_*, RF1_* |
| Credential redaction | FULL | 5+ (cross-cutting) | VA1_13, FP19, EC-S325-9 |

---

## Gap Lifecycle

### Closed Gaps

| Gap ID | Description | Priority | Opened | Closed | Evidence |
|--------|------------|----------|--------|--------|----------|
| R-S316-1 | Writer consumer missing for venue fills | Medium | S316 | S317 | WriterVenueMarketOrderFillConsumer() |
| R-S316-2 | Commission uses cumQuote proxy | Low | S316 | Accepted | Market orders on testnet; real fee endpoint out of scope |
| R-S316-3 | Partial fill not observed | Low | S316 | Accepted | Market orders fill atomically on testnet |
| R-S316-4 | Kill switch tested with mock only | Low | S316 | S323 | WithHaltChecker with GateChecker interface |
| R-S320-1 | No reconciliation for body-read-failure-after-200 | Medium | S320 | S322 | Post200Reconciler + QueryOrder |
| R-S320-2 | No global retry deadline | Low | S320 | S323 | RetryPolicy.Deadline (default 10s) |
| R-S320-3 | Kill switch blind during retry backoff | Low | S320 | S323 | WithHaltChecker checked between attempts |
| R-S320-4 | Venue error codes unused for classification | Low | S320 | S325 | classifyByVenueErrorCode (3 overrides) |
| R-S320-5 | No structured retry metrics | Low | S320 | S324 | 5 log events + 5 counters |

### Open Gaps (Residual)

| Gap ID | Description | Priority | Risk | Disposition |
|--------|------------|----------|------|-------------|
| R-S320-6 | Per-error-class differentiated retry policies | Low | Low | **Deferred** — standard exponential backoff sufficient for single-venue testnet scope |
| R-S322-1 | Single recovery attempt (no retry on query failure) | Low | Low | **Accepted** — query failure returns enriched error; retrying query could mask deeper issues |
| R-S322-2 | No persistence of ambiguous state when recovery fails | Low | Low | **Accepted** — requires OMS infrastructure; out of scope for venue progression |
| R-S322-3 | Fill detail granularity differs between submit and query | Low | Very Low | **Accepted** — both use same parseOrderResponse; consistent but less granular on query |
| R-S322-4 | Theoretical race (order not yet queryable after 200) | Low | Very Low | **Accepted** — Binance processes market orders synchronously |
| R-S323-1 | Deadline does not cancel in-flight submits | Low | Low | **Accepted** — per-submit context deadline handles this independently |
| R-S323-2 | Halt check timeout fixed at 2s | Low | Very Low | **Accepted** — consistent with SafetyGate gateReadTimeout |
| R-S323-3 | Production wiring of WithHaltChecker in actor pipeline | Medium | Low | **Deferred** — composition task, not design gap |
| R-S324-1 | WithLogger/WithTracker not wired in bootstrap | Low | Low | **Deferred** — composition task |
| R-S324-2 | No per-symbol breakdown in retry counters | Low | Very Low | **Accepted** — high cardinality; actor-level per-symbol counters exist |
| R-S325-1 | No real-world error code corpus | Low | Low | **Accepted** — conservative mapping; unmapped codes default safely |
| R-S325-2 | No Retry-After header extraction | Low | Low | **Accepted** — exponential backoff sufficient for testnet |
| R-S325-3 | Mapping is Binance-specific | Low | N/A | **By design** — each venue gets own adapter-scoped mapping |

### Gap Classification Summary

| Classification | Count | Examples |
|---------------|-------|---------|
| Closed (implemented + tested) | 9 | R-S316-1, R-S320-1 through R-S320-5 |
| Accepted (risk acknowledged, no action needed) | 10 | R-S322-1..4, R-S323-1..2, R-S324-2, R-S325-1..3 |
| Deferred (composition/wiring for future stage) | 3 | R-S320-6, R-S323-3, R-S324-1 |
| **Total tracked** | **22** | |

---

## Production Wiring Gap Analysis

The audit revealed that all S319–S325 components are **implemented and tested in isolation** but the composition chain is not wired in the production actor pipeline.

### Current Production Path

```
execute_supervisor.go
  └── VenueAdapterActor(venuePort)    ← raw adapter, no retry/reconciliation
        └── SafetyGate → BinanceFuturesTestnetAdapter.SubmitOrder()
```

### Target Production Path

```
execute_supervisor.go
  └── VenueAdapterActor(post200Reconciler)
        └── SafetyGate → Post200Reconciler(retrySubmitter, queryPort)
              └── RetrySubmitter(adapter)
                    .WithHaltChecker(gateChecker)
                    .WithLogger(logger)
                    .WithTracker(tracker)
                    └── BinanceFuturesTestnetAdapter.SubmitOrder()
```

### Wiring Tasks (Not Part of This Gate)

| Task | Files | Complexity | Risk |
|------|-------|-----------|------|
| Compose RetrySubmitter around adapter | execute_supervisor.go | Low | Low (decorator pattern) |
| Compose Post200Reconciler around RetrySubmitter | execute_supervisor.go | Low | Low (decorator pattern) |
| Wire WithHaltChecker from control store | execute_supervisor.go | Low | Low (interface already defined) |
| Wire WithLogger and WithTracker | execute_supervisor.go | Low | Low (optional, nil-safe) |

These are **mechanical wiring tasks** that compose already-tested components.
They do not require new design, new interfaces, or new tests. They are
appropriate for a production-readiness stage, not for the venue progression gate.

---

## Next Ceremony Recommendation

### Gate Verdict: VENUE PROGRESSION CLOSED

The venue progression has achieved its minimum scope:
- Real venue interaction proven end-to-end
- Persistence round-trip wired
- Retry infrastructure complete with deadline, halt, observability
- Failure paths comprehensively verified (19 scenarios)
- Post-200 reconciliation implemented
- Venue error code classification enriched
- 186 tests passing, 0 regressions

### Candidates for Next Ceremony

The next ceremony should be selected based on strategic project priorities.
The following candidates emerge from the evidence, **ordered by observed proximity
to production readiness**, not by preference:

| Candidate | What It Addresses | Scope |
|-----------|------------------|-------|
| **Production wiring tranche** | Compose RetrySubmitter + Post200Reconciler + observability into actor pipeline | Small (1–2 stages, ~1 file changed) |
| **Next vertical slice** | New domain capability beyond execution (analytics, risk, multi-symbol) | Medium–Large |
| **Multi-venue expansion** | Second venue adapter (Bybit, dYdX) | Medium |
| **CI/CD hardening** | Automated smoke, test gates in pipeline | Small–Medium |

### Recommendation

The production wiring tranche is the **most mechanically scoped** next step.
All code exists and is tested; the only work is composition in
`execute_supervisor.go`. This could be a single stage (S327) or folded into
a broader production-readiness ceremony.

However, the choice depends on whether the project's strategic priority is
**depth** (production-readiness of venue execution) or **breadth** (new
capabilities). This decision belongs to the project owner, not to the evidence
gate.

---

## Appendix: Test Suite Snapshot (2026-03-21)

```
$ go test ./internal/application/execution/... -count=1 -timeout 120s
ok   internal/application/execution   32.031s

Total passing:  186
Total failing:  0
Total skipped:  0 (E2E tests skip without credentials)
```
