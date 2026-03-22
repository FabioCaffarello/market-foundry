# Stage S341 — Controlled Activation Verification Report

> Proves that activation controls real event flow through the live actor pipeline.

## Executive Summary

S341 closes the gap between domain-level acceptance (S340) and live infrastructure flow (S333) by proving that gate transitions during a running supervisor actually control whether events produce fills or are blocked. Five integration tests exercise the full lifecycle — halted → enabled → halted — on the real NATS → Hollywood actor → VenueAdapterActor pipeline. The activation surface is verified as an operational capability, not just a design artifact.

## Entry State

At S340 exit:

- Activation model (3 dimensions, 4 effective modes) fully defined and unit-tested
- 6 acceptance scenarios (AC-1 through AC-6) proven at domain level
- HTTP control surface smoke validates gate transitions via `/execution/control`
- Live consumer flow tests (LF-1 through LF-4) prove NATS → actor pipeline for single gate states
- 202+ tests green across the repository

**Gap**: no test proves that changing the gate at runtime changes real event flow through the actor pipeline.

## Controlled Verification Validated

### Integration Tests (5 scenarios)

| Test | Scenario | Result |
|------|----------|--------|
| CAV-1 | Halted gate blocks live actor path | Gate=halted → event reaches actor, skipped_halt incremented, filled=0 |
| CAV-2 | Gate open enables live flow | Gate=active → VenueOrderFilledEvent received, filled incremented |
| CAV-3 | Gate halt blocks after enable | Active→halted runtime transition → subsequent events blocked |
| CAV-4 | Full lifecycle (halted→enabled→halted) | Three phases on single supervisor, counters consistent |
| CAV-5 | Audit fields observable through NATS KV | reason, updated_by, updated_at round-trip and compose into surface |

### Smoke Integration (Phase 6)

The `smoke-activation.sh` script now includes Phase 6 which runs the S341 integration tests when NATS is reachable. Single entry point: `make smoke-activation`.

## Files Changed

| File | Type | Change |
|------|------|--------|
| `internal/actors/scopes/execute/controlled_activation_verification_test.go` | New | 5 integration tests (CAV-1 through CAV-5) |
| `scripts/smoke-activation.sh` | Modified | Phase 6 added; banner/summary updated |
| `Makefile` | Modified | smoke-activation target description updated |
| `docs/architecture/controlled-activation-verification-with-live-venue-path.md` | New | Verification strategy and scenario definitions |
| `docs/architecture/live-activation-evidence-behavior-and-limitations.md` | New | Evidence, behavior, and limitation analysis |
| `docs/stages/stage-s341-controlled-activation-verification-report.md` | New | This report |

## Principal Evidence

1. **Gate transitions control real flow**: CAV-3 and CAV-4 prove that changing the NATS KV gate value while a supervisor is running immediately affects whether the next event produces a fill or is blocked.

2. **Per-intent evaluation**: the SafetyGate reads the gate on every intent, not per-session. This means halt is effective within the KV propagation window (~1-10ms).

3. **Counter integrity**: `processed = filled + skipped_halt` holds across all scenarios. Health tracker counters accurately reflect gate decisions on the real path.

4. **Audit trail completeness**: gate reason, operator ID, and timestamp survive the NATS KV round-trip and are accessible through the ActivationSurface composite.

5. **Full lifecycle on single supervisor**: CAV-4 proves halted → enabled → halted without supervisor restart, demonstrating that the gate is the sole runtime control lever.

## Remaining Limitations

| Limitation | Severity | Notes |
|-----------|----------|-------|
| Paper adapter used (no real venue HTTP) | Medium | Real venue response parsing untested |
| Binary restart rollback untested in integration | Low | Proven at domain level (AC-3) |
| Extended observation window not exercised | Low | Tests run in seconds |
| Multi-venue gating not available | Low | Single global gate by design |
| HTTP → KV path not tested in integration | Low | Smoke script covers HTTP; integration tests use KV directly |
| Fail-open on KV unavailability accepted | Accepted | NATS outage also blocks event arrival |

## Preparation for S342

S341 converts activation readiness into proven operational capability. Recommended next steps:

1. **Real venue adapter smoke** — exercise the activation lifecycle with the Binance Futures testnet adapter instead of the paper simulator. This closes the "paper adapter only" limitation.

2. **Extended observation proof** — sustained operation (minutes, not seconds) with active gate, monitoring counter drift and resource behavior.

3. **Composite observability** — verify that activation state is queryable through the gateway HTTP surface during live operation, not just through NATS KV directly.

4. **Operational runbook validation** — execute the S338 pre-activation checklist against a real testnet deployment, documenting any gaps between documented procedure and actual operator workflow.

## Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Controlled activation proven on real path | Met | CAV-1 through CAV-4 |
| Enablement, halt, rollback auditable | Met | CAV-5 + AC-3 (domain) |
| Readiness converted to operational capability | Met | Gate transitions control real flow |
| Gaps honestly documented | Met | 6 limitations cataloged with severity |
