# Stage S65 — Risk Projection Hardening

| Field | Value |
|-------|-------|
| Stage | S65 |
| Title | Risk Projection Hardening |
| Status | Complete |
| Date | 2026-03-18 |

## 1. Executive Summary

Hardened the `risk` domain's runtime and documentation to make projection authority, idempotency, replay safety, and latest-only semantics explicit and verifiable. No new features added — this stage consolidates the S64 first slice into a mature, explainable foundation.

The goal: risk must prove structural reliability before being considered as input for an `execution` layer.

---

## 2. Files Changed

### Runtime (3 files)

| File | Change |
|------|--------|
| `internal/adapters/nats/risk_consumer.go` | Added redelivery detection: logs warning when `NumDelivered > 1` with subject, count, and stream sequence |
| `internal/actors/scopes/store/risk_projection_actor.go` | Added projection authority assertion at start; added `checkStatsInvariant()` at shutdown |
| `internal/adapters/nats/risk_kv_store.go` | Added post-read validation in `Get()`: domain-validates assessment before returning |

### Documentation (3 files, new)

| File | Content |
|------|---------|
| `docs/architecture/risk-projection-pattern.md` | Projection authority, latest-only semantics, materialization gates, observability counters, query path, health tracking |
| `docs/architecture/risk-replay-idempotency-rules.md` | Three-layer replay safety (JetStream dedup, durable consumer, monotonicity guard), replay scenarios, invariant catalog |
| `docs/stages/stage-s65-risk-projection-hardening-report.md` | This report |

---

## 3. Hardening Applied

### H1: Redelivery Detection (Consumer)

**Before**: Redelivered messages were silently processed. No observability into JetStream redelivery behavior.

**After**: When `NumDelivered > 1`, the consumer logs a structured warning with subject, delivery count, and stream sequence. This makes consumer health and replay pressure visible without external tooling.

### H2: Projection Authority Assertion (Projection Actor)

**Before**: The projection actor started silently. Its role as sole writer to the KV bucket was implicit.

**After**: At startup, the actor logs its projection authority, bucket ownership, and latest-only semantics. This makes the ownership model grep-able and auditable.

### H3: Stats Invariant Check (Projection Actor)

**Before**: Stats were logged at shutdown but never validated. A bug that dropped a message between gates would be invisible.

**After**: At shutdown, the actor verifies `received == materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors`. A violation is logged at ERROR level. This catches code-path bugs where a message is received but no outcome is recorded.

### H4: Post-Read Validation (KV Store)

**Before**: `Get()` deserialized the KV entry and returned it without validation. Corrupted or schema-drifted data would be served to clients.

**After**: `Get()` runs `RiskAssessment.Validate()` after deserialization. If the stored data fails domain validation, an Internal error is returned instead of invalid data. This protects the query path from serving garbage.

### H5: Invariant Documentation

**Before**: Replay safety, projection ownership, and latest-only semantics were emergent properties of the code — correct but undocumented.

**After**: Two architecture documents (`risk-projection-pattern.md` and `risk-replay-idempotency-rules.md`) define 8 named invariants (RI-1 through RI-8), document 5 replay scenarios, and establish the authority model as a first-class architectural decision.

---

## 4. Tests

All existing tests pass after hardening:

| Package | Result |
|---------|--------|
| `internal/actors/scopes/store` | OK (includes 11 risk projection tests) |
| `internal/adapters/nats` | OK (includes risk KV store tests) |
| `internal/domain/risk` | OK (12 tests) |
| `internal/application/risk` | OK (10 tests) |
| `internal/application/riskclient` | OK (5 tests) |
| `internal/interfaces/http/handlers` | OK (4 risk tests) |
| `internal/interfaces/http/routes` | OK (3 risk tests) |

Build: clean (`go build internal/...` passes).

---

## 5. Limitations Remaining

| Limitation | Impact | Mitigation Path |
|-----------|--------|----------------|
| No history projection | Cannot query past risk assessments via API | Use `RISK_EVENTS` stream directly (72h retention); add history bucket in future stage if needed |
| Single risk family | Only `position_exposure` is implemented | Architecture supports multiple families; add drawdown_guard (RF-02) in S66+ |
| No multi-strategy aggregation | Each evaluator sees one strategy at a time | Current design intentional for first slice; aggregate risk is a separate family |
| Monotonicity guard is read-before-write | Tiny race window under concurrent writers | Mitigated by single-writer authority; KV revision-based CAS available if needed |
| No automated replay tooling | Manual ops intervention required for full replay | Acceptable at current scale; add replay operator command in future |
| Stream retention bounded | 72h / 2GB — data older than this is lost | Sufficient for operational use; long-term archival is a separate concern |

---

## 6. Preparation for Next Readiness Review

Before `risk` can be considered input for `execution`, the following should be validated:

### Ready Now

- [x] Projection authority is explicit and documented
- [x] Replay/idempotency is safe at all three layers
- [x] Latest-only is a documented, intentional choice
- [x] Query path validates data before serving
- [x] Health trackers cover both consumer and projection
- [x] Stats invariant is enforced at shutdown
- [x] Redelivery is observable

### Recommended Before Execution Layer

- [ ] Run risk pipeline under sustained load (>1000 events/min) and verify counters
- [ ] Verify graceful degradation: stop NATS mid-stream, confirm no data corruption after recovery
- [ ] Consider adding a second risk family (e.g., drawdown_guard) to validate multi-family projection patterns
- [ ] Evaluate whether `execution` needs history or if latest-only is sufficient for order placement decisions
- [ ] Review correlation-ID propagation end-to-end from strategy through risk to ensure traceability

### Guard Rails Respected

- [x] No `execution` layer implemented
- [x] No history projection opened
- [x] No framework or abstraction introduced
- [x] No redesign — all changes are surgical hardening
- [x] Limitations documented honestly
