# S345 — Operational Runbook Validation Report

> Venue Activation Wave — VA-8: Operational runbook validation against real/testnet environment.

## Executive Summary

S345 validates the activation runbook against the live stack, transforming the
enable/halt/rollback/verification procedures from informal knowledge into an
auditable, reproducible operator playbook.

All five runbook procedures (enable, halt, rollback, verification, pre-deployment
safety check) were executed and validated. Four documentation gaps were identified
and corrected. Six limitations were documented. The activation wave achieves
procedural clarity sufficient for evidence gate closure.

## Objective

Execute, validate, and document the operational runbook for the venue activation
lifecycle against a real/testnet environment, ensuring every step is clear,
sufficient, and produces the expected outcome.

## What Was Done

### 1. Runbook Formalization

Created the canonical operator runbook (`docs/architecture/operational-runbook-validation.md`)
covering five procedures:

| Procedure | Scope |
|-----------|-------|
| Enable (halted → active) | Gate transition + activation surface verification |
| Halt (active → halted) | Emergency stop + audit trail |
| Rollback (active → halted → paper) | Gate halt + optional binary restart for full paper retreat |
| Verification / health check | One-liner, full diagnostic, and automated smoke |
| Pre-deployment safety check | Detect venue_live and block deploy |

Each procedure includes:
- Exact commands with expected outputs.
- Success criteria.
- Failure mode table with recovery actions.

### 2. Runbook Validation

Executed all procedures against the live Docker Compose stack. Results documented
in `docs/architecture/activation-runbook-checklist-results-and-limitations.md`.

| Procedure | Result |
|-----------|--------|
| Enable | PASS — gate transitions immediately, audit fields round-trip |
| Halt | PASS — instantaneous, idempotent |
| Rollback (gate-only) | PASS — sufficient for emergency stop |
| Verification | PASS — all fields present, `make smoke-activation` passes |
| Pre-deployment check | PASS — correctly detects venue_live |

### 3. Gaps Found and Corrected

| Gap | Correction |
|-----|------------|
| 503 handling not in runbook | Added failure mode entry with recovery action |
| Reason field convention not formalized | Established `runbook-{action}-{context}` convention |
| Idempotency not documented | Verified and documented: gate operations are idempotent |
| Smoke-to-runbook cross-reference unclear | Clarified that smoke phases 2–4 ARE the runbook validation |

### 4. Limitations Documented

| ID | Limitation | Impact |
|----|-----------|--------|
| L1 | No automated circuit breaker / self-halt | Low for testnet |
| L2 | No activation history endpoint | Low; KV revisions available but not exposed |
| L3 | Full paper rollback requires binary restart | Acceptable; JetStream queues events |
| L4 | No push notifications for gate changes | Low for manual ops |
| L5 | Credentials immutable per process | Same restart constraint as L3 |
| L6 | Global gate, no per-venue isolation | Acceptable for single-venue |

## Artifacts

| Artifact | Path |
|----------|------|
| Operational runbook | [`docs/architecture/operational-runbook-validation.md`](../architecture/operational-runbook-validation.md) |
| Checklist results and limitations | [`docs/architecture/activation-runbook-checklist-results-and-limitations.md`](../architecture/activation-runbook-checklist-results-and-limitations.md) |
| Stage report | This document |

## Evidence

### Smoke Execution

The canonical smoke (`make smoke-activation`) was executed as part of validation.
It covers 9 phases:

1. Stack and control surface readiness
2. AC-1: Inactive → Active (enable)
3. AC-2: Active → Halt (halt)
4. AC-3: Halt → Rollback (rollback)
5. S340 unit test gate
6. S341 controlled activation verification (integration)
7. S342 real venue activation (integration)
8. S343 extended observation window (integration)
9. S344 activation surface queryability (HTTP)

All phases pass. The smoke script IS the automated runbook validation.

### Gate Idempotency

Verified by executing consecutive identical PUT operations:
- PUT halted (already halted) → 200, no error.
- PUT active (already active) → 200, no error.

### Audit Field Round-Trip

Verified that `reason`, `updated_by`, and `updated_at` are preserved through:
- PUT → GET on `/execution/control`
- PUT → GET on `/activation/surface` (gate sub-object)

## Relationship to Prior Stages

| Stage | Contribution to S345 |
|-------|---------------------|
| S337 | Charter: defined activation wave scope |
| S338 | Policy: rollout/rollback model that the runbook operationalizes |
| S339 | Surface: three-dimensional model that the runbook queries |
| S340 | Smoke: acceptance scenarios (AC-1, AC-2, AC-3) |
| S341 | Integration: controlled activation on real actor path |
| S342 | Integration: real venue adapter with HTTP interactions |
| S343 | Observation: sustained operation without drift |
| S344 | Queryability: HTTP endpoint that the runbook uses for verification |

S345 integrates all prior wave deliverables into a validated operational procedure.

## Remaining Limits

- The runbook validates testnet operations only. Production deployment introduces
  additional concerns (monitoring, alerting, multi-replica coordination) that are
  out of scope for this wave.
- No automated CI runbook regression — the smoke script provides the automated
  validation, but there is no dedicated "runbook test" beyond it.
- Full paper rollback (binary restart) was validated conceptually but not exercised
  end-to-end in this validation session (gate-only rollback was exercised).

## Preparation for S346

The activation wave is now positioned for evidence gate closure:

1. **Charter deliverables covered**: S337 charter authorized enable, halt, rollback,
   verification, and queryability. All are now validated with runbook procedures.
2. **Procedural clarity achieved**: Every operational step is documented with exact
   commands, expected outputs, and failure recovery.
3. **Limitations are honest**: No hidden manual steps or fragile procedures remain
   undocumented.
4. **Recommended S346 scope**: Evidence gate — formal closure of the venue activation
   wave with deliverable reconciliation against the S337 charter.
