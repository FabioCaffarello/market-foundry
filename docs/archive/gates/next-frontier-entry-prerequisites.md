# Next Frontier Entry Prerequisites

**Stage**: S86
**Date**: 2026-03-18
**Scope**: Gate criteria for advancing beyond paper-integrated execution

---

## 1. Context

The paper-integrated execution phase (S74–S85) established a complete pipeline: derive → execute → store → gateway. This document defines the prerequisites that must be met before crossing the next frontier.

The next frontier is **not** real venue implementation. It is the resolution of post-paper gaps that would make real venue design and eventual implementation safe.

---

## 2. Frontier Definition

### Current Position: Paper Execution Complete
- 4 binaries operational (derive, execute, store, gateway)
- Paper mode pipeline end-to-end functional
- Kill switch, staleness guard, fill projection, status propagation implemented
- 154+ unit tests, 8 integration tests, 22 smoke steps

### Next Frontier: Operational Hardening Gate
- Infrastructure-level validation
- CI-integrated multi-binary testing
- Observability foundation
- Fill integrity verification

### Subsequent Frontier: Real Venue Design Gate
- Credential infrastructure design
- Async fill model design
- Transitional bridge migration
- Activation gate ceremony prerequisites

---

## 3. Operational Hardening Gate — Prerequisites

These must ALL pass before the system can be considered operationally hardened.

### PRE-H1: Docker Compose Integration

**Gate criterion**: `execute` service present in docker-compose.yaml with:
- [ ] Service definition with correct binary path
- [ ] NATS dependency declaration
- [ ] Health check endpoint configured (:8084)
- [ ] Correct config file mount (execute.jsonc)
- [ ] Smoke tests pass without manual execute startup

**Verification**: `docker compose up` starts all 4 services; smoke tests run unconditionally.

---

### PRE-H2: Embedded NATS Integration Tests

**Gate criterion**: Test suite with embedded NATS server validates:
- [ ] KV monotonicity guard rejects stale writes
- [ ] KV monotonicity guard accepts newer writes
- [ ] Consumer redelivery works within MaxDeliver limit
- [ ] Consumer terminates after MaxDeliver exhaustion
- [ ] JetStream deduplication window rejects duplicate MsgId
- [ ] Stream creation with correct retention/limits
- [ ] Full consumer → projection round-trip with real KV

**Verification**: `go test ./...` includes NATS integration tests (build-tagged or conditional).

---

### PRE-H3: Observability Surface

**Gate criterion**: HTTP endpoint on execute binary (:8084) exposes:
- [ ] Actor counter stats (processed, filled, skipped_stale, skipped_halt, errors)
- [ ] Consumer delivery stats (delivered, redelivered, terminated, nakked)
- [ ] Uptime and last-event timestamps

**Verification**: Smoke test queries observability endpoint and validates non-zero counters after pipeline run.

---

### PRE-H4: Fill Reconciliation Verification

**Gate criterion**: Composite status endpoint validates intent-fill consistency:
- [ ] When intent exists but result does not: propagation = intent.Status
- [ ] When both exist: propagation = result.Status
- [ ] When result exists with matching partition key to intent: verified as correlated
- [ ] Staleness detection: submitted intent older than 120s with no result triggers diagnostic field

**Verification**: Integration test exercises all 4 scenarios; smoke test validates composite endpoint.

---

### PRE-H5: CI Pipeline Integration

**Gate criterion**: CI pipeline runs full 4-binary validation:
- [ ] All unit tests pass
- [ ] All integration tests pass (with embedded NATS where applicable)
- [ ] Docker Compose starts all services
- [ ] Smoke test completes all steps (including execute-specific steps)
- [ ] Drift rules pass (raccoon-cli)

**Verification**: CI green on all checks; no manual steps required.

---

## 4. Real Venue Design Gate — Prerequisites

These must ALL pass before real venue adapter design can proceed. They build on the Operational Hardening Gate.

### PRE-V1: Operational Hardening Gate Passed

**Gate criterion**: All PRE-H* prerequisites verified and passing in CI.

---

### PRE-V2: Transitional Bridge Resolution Design

**Gate criterion**: Design document for migrating execute consumer from paper_order subjects to venue-specific subjects:
- [ ] New event type: `VenueOrderIntentEvent` defined
- [ ] New subject: `execution.events.venue_market_order.submitted` specified
- [ ] Migration steps documented (dual-consumer phase, cutover, cleanup)
- [ ] Backward compatibility strategy (derive continues paper_order, execute adds venue subject)
- [ ] Drift rule updates planned

**Verification**: Architecture document exists; drift rules updated to track new event type.

---

### PRE-V3: Async Fill Model Design

**Gate criterion**: Design document for asynchronous venue order lifecycle:
- [ ] SubmitOrder returns receipt (order ID + status), fill arrives separately
- [ ] Fill channel design (websocket subscription, polling, or callback)
- [ ] Timeout and cancellation semantics
- [ ] Partial fill accumulation model
- [ ] VenuePort interface extension (or new interface) defined

**Verification**: Architecture document exists; VenuePort interface changes documented.

---

### PRE-V4: Credential Infrastructure Design

**Gate criterion**: Design document for exchange credential management:
- [ ] Credential storage mechanism (encrypted config, secret manager, env vars)
- [ ] Credential validation at startup (buildVenueAdapter fails if credentials invalid)
- [ ] Credential rotation support
- [ ] Per-venue credential isolation
- [ ] No credentials in source control (gitignore rules)

**Verification**: Architecture document exists; schema.go changes documented.

---

### PRE-V5: Activation Gate Ceremony Prerequisites

**Gate criterion**: All 17 gates from S75 activation gate design have been evaluated:
- [ ] Gates 1–10: Already passed (S78–S79)
- [ ] Gates 11–17: Already passed (S80–S84)
- [ ] Remaining gates (if any) identified and tracked
- [ ] Ceremony initiation criteria defined

**Verification**: Gate checklist with pass/fail per gate; ceremony ready to initiate.

---

## 5. Gate Sequencing

```
Current State (S86)
    │
    ▼
Operational Hardening Gate (S87–S88)
    PRE-H1: Docker Compose
    PRE-H2: NATS integration tests
    PRE-H3: Observability surface
    PRE-H4: Fill reconciliation
    PRE-H5: CI pipeline
    │
    ▼
Real Venue Design Gate (S88–S89)
    PRE-V1: Hardening gate passed
    PRE-V2: Transitional bridge resolution design
    PRE-V3: Async fill model design
    PRE-V4: Credential infrastructure design
    PRE-V5: Activation gate ceremony prerequisites
    │
    ▼
Activation Gate Ceremony (S89+)
    17 formal gates from S75
    │
    ▼
First Real Venue Implementation (S90+)
    Guarded, single-venue, single-symbol
```

---

## 6. Anti-Patterns to Avoid

1. **Do not skip hardening to reach venue design faster.** Infrastructure gaps masked in paper mode will surface as production incidents with real money.

2. **Do not implement real venue adapter before design gate passes.** The async fill model and credential infrastructure are foundational — implementing without design creates rework.

3. **Do not conflate paper hardening with venue readiness.** Hardening makes paper mode reliable. Venue readiness requires additional design work on top of reliable paper mode.

4. **Do not initiate activation gate ceremony before all prerequisites are met.** The ceremony is a formal gate, not a formality. Initiating with known gaps defeats its purpose.

5. **Do not add multiple venue types simultaneously.** First real venue should be single-venue, single-symbol, heavily guarded. Multi-venue routing is a subsequent frontier.
