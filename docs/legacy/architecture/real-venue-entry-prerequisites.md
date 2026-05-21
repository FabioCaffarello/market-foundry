# Real Venue Entry Prerequisites

> **Stage:** S89
> **Date:** 2026-03-19
> **Authority:** Post-hardening action boundary gate
> **Scope:** Prerequisites that must be satisfied before the activation gate ceremony (AG-1..AG-17) can be initiated

---

## Overview

This document enumerates the concrete prerequisites for transitioning from paper-only execution to the first guarded real-venue adapter. Prerequisites are organized into three tiers:

| Tier | Meaning | When |
|------|---------|------|
| **PRE-A** | Must be complete before activation gate ceremony begins | Before AG-1 |
| **PRE-G** | Must be verified during guarded phase (first 72h of real venue) | During shadow/guarded |
| **PRE-O** | Must be complete before transitioning to operational phase | Before full operation |

---

## Tier PRE-A: Pre-Activation Prerequisites

These must be **implemented and verified** before running the 17-gate activation ceremony.

### PRE-A1: Credential Infrastructure

- **Requirement:** `LoadCredentials()` function implemented in `internal/application/execution/` that reads environment variables with prefix `MF_VENUE_{TYPE}_{NAME}`.
- **Validation:** Execute binary fails fast on startup if venue type is not `paper_simulator` and credentials are missing.
- **Docker Compose:** `env_file` template exists at `deploy/configs/execute.env.example` with placeholder values.
- **Security:** `.gitignore` includes `*.env` files. No credential values in config files, logs, or error messages.
- **Verification:** Unit test confirms fail-fast behavior. Manual test confirms env_file loading in Docker Compose.

### PRE-A2: Reconciliation Invariant Enforcement

- **Requirement:** FillProjectionActor enforces reconciliation invariants RC-1, RC-2, RC-4, and RC-6.
- **RC-1 (fill-to-intent correlation):** Fill projection validates that a corresponding execution intent exists before materializing. Orphan fills are rejected with counter increment.
- **RC-2 (quantity boundary):** Cumulative filled quantity does not exceed requested quantity. Overflow fills are rejected.
- **RC-4 (orphan handling):** Fills without matching intent are logged at WARN level and counted as `orphaned`.
- **RC-6 (dedup idempotency):** JetStream message dedup key configured on `EXECUTION_FILL_EVENTS` stream.
- **Verification:** Unit tests for each invariant. Integration test with intentional violation confirms rejection.

### PRE-A3: Embedded NATS Integration Tests

- **Requirement:** Test harness using embedded NATS server with `go test -tags=integration` isolation.
- **Scenarios (8 minimum):**
  1. Consumer receives and delivers event to handler.
  2. Kill switch gate halts processing.
  3. Staleness guard rejects old intent.
  4. Venue adapter submits order and publishes fill.
  5. Redelivery on handler failure (NAK path).
  6. Message termination on decode failure (TERM path).
  7. Multi-symbol isolation (no cross-bleed).
  8. Stats invariant holds after mixed success/failure batch.
- **Verification:** All 8 scenarios pass. `make test-integration` target exists in Makefile.

### PRE-A4: Real Venue Adapter Implementation

- **Requirement:** A concrete `VenuePort` implementation for the target exchange.
- **Contract:**
  - `SubmitOrder(ctx, request) → (receipt, error)` with configurable timeout context.
  - Returns `VenueOrderReceipt` with real venue order ID.
  - Handles API errors with appropriate problem classification.
  - Rate limiting respects exchange API limits.
- **Security:** API keys loaded via PRE-A1 credential infrastructure. No hardcoded secrets.
- **Verification:** Unit tests with mock HTTP server. Manual test against exchange sandbox/testnet.

### PRE-A5: Venue Submit Timeout

- **Requirement:** `VenueAdapterActor` passes a configurable timeout context to `VenuePort.SubmitOrder()` instead of `context.Background()`.
- **Default:** 10 seconds (configurable via settings).
- **Verification:** Unit test confirms timeout cancellation. Integration test with slow mock confirms timeout behavior.

### PRE-A6: Staleness MaxAge Configurable

- **Requirement:** Staleness threshold is configurable in `execute.jsonc` rather than hardcoded.
- **Schema:** `venue.staleness_max_age_seconds` field in settings schema with validation (minimum 30s, maximum 600s).
- **Default:** 120 seconds (backward compatible).
- **Verification:** Config validation test. Startup test with custom value.

### PRE-A7: Venue Architecture Documentation

- **Requirement:** Architecture document `docs/architecture/venue-{name}-adapter-design.md` describing:
  - Exchange API characteristics (REST/WebSocket, rate limits, authentication).
  - Order type support (market only for first adapter).
  - Fill delivery mechanism (polling vs WebSocket).
  - Error classification (retryable vs terminal).
  - Sandbox/testnet availability.
- **Verification:** Document exists and is referenced in raccoon-cli execution docs registry.

### PRE-A8: Drift Rules Extended

- **Requirement:** Raccoon-CLI drift detection updated with:
  - New venue type in `knownVenueTypes` registry.
  - Venue-specific adapter file existence checks.
  - Venue-specific subject naming conventions.
- **Verification:** `make quality-gate` passes with new venue type registered.

---

## Tier PRE-G: Guarded Phase Prerequisites

These are verified **during** the shadow (24h) and guarded (72h) phases after activation.

### PRE-G1: Shadow Mode Validation (24h)

- **Requirement:** Real venue adapter runs in shadow mode — submits real orders but with minimum possible quantity.
- **Validation criteria:**
  - All orders reach the exchange (no silent drops).
  - Fill events are received and projected correctly.
  - Reconciliation invariants RC-1..RC-4 hold for 24h continuous operation.
  - Error rate < 1% of submissions.
  - No orphan fills detected.
  - Kill switch halt/resume cycle tested with real venue.
- **Evidence:** `/statusz` counters snapshot at 0h, 12h, 24h. Stats invariant holds at each checkpoint.

### PRE-G2: Guarded Mode Validation (72h)

- **Requirement:** Real venue adapter runs with reduced quantity limits (e.g., 10% of paper mode quantities).
- **Validation criteria:**
  - All shadow mode criteria continue to hold.
  - Partial fill handling works correctly (if exchange produces partials).
  - Latency distribution is acceptable (p50, p95, p99 from logs).
  - No stuck intents detected (or background reconciliation catches them).
  - Multi-symbol operation is stable.
- **Evidence:** Daily `/statusz` snapshots. Log analysis for error patterns.

### PRE-G3: Kill Switch Responsiveness

- **Requirement:** Kill switch halts all venue submissions within one consumer poll cycle (~10s).
- **Validation:** Manual halt during guarded phase. Verify no orders submitted after halt. Verify resume works cleanly.

### PRE-G4: Rollback Capability

- **Requirement:** Documented rollback procedure to return to paper-only mode.
- **Steps:**
  1. Set kill switch to `halted` via `PUT /execution/control`.
  2. Change `venue.type` back to `paper_simulator` in execute.jsonc.
  3. Restart execute service.
  4. Verify paper fills resume.
- **Verification:** Execute rollback procedure at least once during guarded phase.

---

## Tier PRE-O: Pre-Operational Prerequisites

These must be complete before transitioning from guarded to full operational mode.

### PRE-O1: Prometheus Metrics Endpoint

- **Requirement:** `/metrics` endpoint exposing execution counters, fill latency histogram, and error rates in Prometheus format.
- **Rationale:** Time-series metrics required for operational alerting and capacity planning.

### PRE-O2: CI Pipeline Automation

- **Requirement:** GitHub Actions workflow implementing the 7-stage pipeline designed in S88.
- **Stages:** build → test → quality-gate → config-validate → docker-compose → smoke → teardown.
- **Gate criteria:** All 8 CI gate criteria (CI-1..CI-8) enforced.

### PRE-O3: Background Reconciliation Actor

- **Requirement:** Background actor that periodically checks for stuck intents (RC-5) and orphan fills (RC-4 enforcement at runtime).
- **Frequency:** Every 60 seconds.
- **Alerting:** Logs at ERROR level on stuck intents older than 5 minutes.

### PRE-O4: Consumer-Projection Transactional Coupling

- **Requirement:** Address the consumer-ACK-before-projection-write gap (SR-S89-1).
- **Approach:** Either NAK on KV failure (retry path) or request/reply coupling.
- **Impact:** Eliminates the data loss window for fill events.

---

## Prerequisite Dependency Graph

```
PRE-A1 (credentials) ──────────────┐
PRE-A4 (venue adapter) ────────────┤
PRE-A5 (submit timeout) ───────────┤
PRE-A6 (staleness config) ─────────┤
PRE-A7 (venue arch doc) ───────────┼──→ Activation Gate Ceremony (AG-1..AG-17)
PRE-A8 (drift rules) ──────────────┤
PRE-A2 (reconciliation enforcement)┤
PRE-A3 (NATS integration tests) ───┘
                                    │
                                    ▼
                          PRE-G1 (shadow 24h)
                                    │
                                    ▼
                          PRE-G2 (guarded 72h)
                          PRE-G3 (kill switch test)
                          PRE-G4 (rollback test)
                                    │
                                    ▼
                          PRE-O1 (Prometheus)
                          PRE-O2 (CI automation)
                          PRE-O3 (background reconciliation)
                          PRE-O4 (consumer-projection coupling)
                                    │
                                    ▼
                          OPERATIONAL MODE
```

---

## Activation Gate Ceremony Reference

The 17-gate activation ceremony (AG-1..AG-17, designed in S88) can only be initiated after all PRE-A prerequisites are satisfied. The ceremony itself validates:

- AG-1..AG-5: Structural gates (adapter, type, config, drift, docs)
- AG-6..AG-10: Behavioral gates (kill switch, staleness, validation, trace, events)
- AG-11..AG-17: Operational gates (health, metrics, shutdown, smoke, shadow, guarded, ceremony sign-off)

Each gate produces a PASS/FAIL verdict with evidence. All 17 must pass for real-venue activation.
