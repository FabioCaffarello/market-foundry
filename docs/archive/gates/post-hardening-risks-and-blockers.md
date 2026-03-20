# Post-Hardening Risks and Blockers

> **Stage:** S89
> **Date:** 2026-03-19
> **Context:** Residual risks and blockers after S87 operational hardening and S88 design hardening

---

## Hard Blockers

Hard blockers **must** be resolved before the activation gate ceremony (AG-1..AG-17) can be initiated.

### HB-S89-1: Credential Infrastructure Not Implemented

- **Source:** S88 design (`venue-credentials-and-activation-prerequisites.md`)
- **Impact:** No real venue adapter can be constructed without credentials.
- **Scope:** `LoadCredentials()` function, `CredentialSet` struct, env_file template, fail-fast validation in `cmd/execute/run.go`.
- **Evidence:** Zero implementation exists — S88 produced design only.
- **Resolution:** Implement credential loading, Docker Compose env_file, and startup validation.
- **Effort:** Small (design is complete; implementation is mechanical).

### HB-S89-2: Reconciliation Invariants Not Enforced at Runtime

- **Source:** S88 design (`pre-venue-fill-reconciliation-model.md`)
- **Impact:** Real venue fills can violate RC-1..RC-7 silently. Paper mode masks this because fills always trivially succeed.
- **Scope:** RC-1 (fill-to-intent correlation), RC-2 (quantity boundaries), RC-4 (orphan fill handling), RC-6 (dedup idempotency).
- **Evidence:** FillProjectionActor applies monotonicity guard but does not validate fill-to-intent existence, quantity bounds, or orphan detection.
- **Resolution:** Add validation gates to FillProjectionActor for RC-1, RC-2, RC-4. Configure JetStream dedup for RC-6. RC-3 (status transitions) and RC-5 (stuck detection) can be deferred to post-activation monitoring.
- **Effort:** Medium (requires cross-bucket lookups in projection actor).

### HB-S89-3: No Embedded NATS Integration Tests

- **Source:** S86 blocker HB-POST-1, S88 design (`post-paper-ci-and-validation-baseline.md`)
- **Impact:** Full consumer → gate → venue → publisher flow is only validated by smoke tests (slow, external). No fast feedback loop for execution path regressions.
- **Scope:** 8 integration test scenarios designed in S88: consumer delivery, gate check, venue submit, fill publish, redelivery, staleness reject, kill switch halt, multi-symbol isolation.
- **Evidence:** All NATS interaction tests use mocks. No `go test -tags=integration` harness exists.
- **Resolution:** Implement embedded NATS test harness with `build tag: integration`.
- **Effort:** Medium (test harness setup + 8 scenarios).

---

## Structural Risks

Structural risks are **not blockers** but represent known limitations that must be monitored or addressed in future stages.

### SR-S89-1: Consumer-Projection Decoupling (Data Loss Window)

- **Description:** Consumer ACKs message before projection writes to KV. If KV write fails post-ACK, the event is lost.
- **Mitigation:** Error counters track KV failures. Stats invariant (`received == sum of outcomes`) catches divergence on actor stop.
- **Monitoring:** `/statusz` exposes error counts per projection tracker.
- **Future fix:** Transactional consumer-projection coupling (requires NATS request/reply pattern change).
- **Severity:** Low for paper mode (KV failures are rare). Medium for real venue (fill loss is unacceptable).
- **Timeline:** Must be addressed before real venue goes operational (can remain during guarded/shadow phase if monitoring is active).

### SR-S89-2: Transitional Bridge Coupling

- **Description:** Execute binary consumes `paper_order` subjects as a transitional bridge. This conflates paper and venue intent families at the consumer level.
- **Mitigation:** Explicitly documented in S85. Subject migration plan exists in S88 async fill design.
- **Impact:** Prevents independent scaling of paper vs venue consumers. Operational stats cannot distinguish family origin.
- **Timeline:** Must be migrated before second venue adapter or multi-family activation.

### SR-S89-3: No CI Pipeline Automation

- **Description:** CI pipeline is designed (7 stages, 8 gate criteria) but has no GitHub Actions backing. Validation relies on manual `make verify` and `make smoke-multi`.
- **Mitigation:** Local validation is comprehensive (unit tests + smoke + drift detection).
- **Impact:** No automated regression detection on PR. Manual discipline required.
- **Timeline:** Should be implemented before real venue goes operational. Not required for guarded/shadow phase.

### SR-S89-4: No Prometheus Metrics Endpoint

- **Description:** `/statusz` provides counter-based observability but no time-series metrics export. No latency percentiles, no error rate over time.
- **Mitigation:** Structured logging with slog provides queryable event data. `/statusz` counters detect stalls and error spikes.
- **Impact:** Adequate for paper mode and guarded phase. Insufficient for operational real-venue monitoring (needs latency histograms, fill timing distributions).
- **Timeline:** Implement before real venue transitions from guarded to operational phase.

### SR-S89-5: Staleness MaxAge Not Config-Driven

- **Description:** Default staleness threshold (120s) is hardcoded in `DefaultStalenessMaxAge` constant. Not configurable per deployment.
- **Mitigation:** Acceptable for paper mode where all timeframes use ≥60s candles.
- **Impact:** Real venue may need different staleness thresholds per timeframe or market conditions.
- **Timeline:** Make configurable before real venue activation.

### SR-S89-6: Single Adapter Per Binary

- **Description:** Execute binary supports one venue adapter type per instance. Multi-venue requires multiple execute instances.
- **Mitigation:** Config-driven selection via `knownVenueTypes` registry. Docker Compose can run multiple execute instances with different configs.
- **Impact:** Acceptable for first guarded venue (single exchange). Limits future multi-venue scenarios.
- **Timeline:** Design multi-venue routing if/when second venue is considered.

### SR-S89-7: No Venue Submit Timeout

- **Description:** `VenueAdapterActor` passes `context.Background()` to venue submit. No timeout on real API calls.
- **Mitigation:** Paper adapter is instant (zero latency). No impact today.
- **Impact:** Real venue API calls may hang indefinitely without timeout.
- **Resolution:** Pass configurable timeout context to `VenuePort.SubmitOrder()`.
- **Timeline:** Must be implemented in the real venue adapter itself (part of adapter development).

---

## Risk Heat Map

| Risk | Severity | Likelihood | Phase Impact |
|------|----------|------------|--------------|
| HB-S89-1: No credentials | **Critical** | Certain | Blocks activation |
| HB-S89-2: No reconciliation enforcement | **High** | Certain | Blocks activation |
| HB-S89-3: No NATS integration tests | **High** | Certain | Blocks activation |
| SR-S89-1: Consumer-projection decoupling | Medium | Low | Guarded phase |
| SR-S89-2: Bridge coupling | Low | N/A | Pre-operational |
| SR-S89-3: No CI automation | Medium | N/A | Pre-operational |
| SR-S89-4: No Prometheus metrics | Medium | N/A | Pre-operational |
| SR-S89-5: Staleness not configurable | Low | Low | Pre-activation |
| SR-S89-6: Single adapter per binary | Low | N/A | Future design |
| SR-S89-7: No venue submit timeout | Medium | Medium | Adapter development |

---

## Blocker Resolution Tracking

| Blocker | First Identified | Designed In | Implementation Target |
|---------|-----------------|-------------|----------------------|
| HB-S89-1 | S86 (HB-POST-4) | S88 | S90 |
| HB-S89-2 | S88 (new) | S88 | S90 |
| HB-S89-3 | S86 (HB-POST-1) | S88 | S90 |
