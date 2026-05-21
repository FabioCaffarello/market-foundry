# Activation Smoke Usage, PASS/FAIL Criteria, and Operational Limitations

> S340 — Venue Activation Wave (S337–S342)

## Purpose

This document describes how to run the activation smoke, interpret its
output, understand PASS/FAIL criteria, and acknowledge current operational
limitations.

## Running the Smoke

### Quick Start

```bash
# Prerequisite: stack must be running
make up && make seed

# Run activation smoke
make smoke-activation
```

### Alternative Invocations

```bash
# Direct script execution
./scripts/smoke-activation.sh

# With custom gateway URL
BASE_URL=http://192.168.1.100:8080 make smoke-activation

# Unit tests only (no stack required)
go test -count=1 -run "TestActivationAcceptance_" ./internal/domain/execution/...
```

## Output Format

The smoke uses structured stdout output with colored prefixes:

```
[PASS] message    — assertion passed
[FAIL] message    — assertion failed (increments error counter)
[INFO] message    — progress or context information
[WARN] message    — non-fatal observation
```

### Example Successful Output

```
═══ Activation Smoke (S340 canonical) ═══
[INFO] Canonical entrypoint: make smoke-activation
[INFO] Expected setup before running: make up && make seed
[INFO] Runtime context: BASE_URL=http://127.0.0.1:8080 phases=5s

═══ Phase 1: Stack and Control Surface Readiness ═══
[PASS] Gateway is ready
[PASS] NATS is healthy
[PASS] GET /execution/control → 200 (status=active)

═══ Phase 2: AC-1 — Inactive → Active (off→on transition) ═══
[PASS] [AC-1/step-1] gate confirmed halted — venue_halted posture
[PASS] [AC-1/step-2] gate confirmed active — activation transition proven
[PASS] [AC-1] audit fields preserved (updated_by=smoke-activation)

═══ Phase 3: AC-2 — Active → Halt (on→halt transition) ═══
[PASS] [AC-2/pre] gate confirmed active — precondition met
[PASS] [AC-2] active → halted transition proven (reason=smoke-s340-ac2-halt)

═══ Phase 4: AC-3 — Halt → Rollback (controlled return to safe state) ═══
[PASS] [AC-3/pre] gate confirmed halted — precondition met
[PASS] [AC-3] halt → rollback (gate restored) transition proven

═══ Phase 5: Activation Unit Test Gate ═══
[PASS] S340 activation acceptance tests pass

[PASS] Activation smoke completed (S340 canonical surface)
[INFO] Scenarios validated: AC-1 off→on | AC-2 on→halt | AC-3 halt→rollback
[INFO] Full path: HTTP control surface → NATS KV gate → state transitions proven
```

## PASS/FAIL Criteria

### PASS (exit code 0)

All of the following must be true:

| Criterion | What it means |
|-----------|--------------|
| Gateway responds 200 on /readyz | Stack is running |
| NATS healthz returns OK | JetStream available |
| GET /execution/control returns 200 | Control surface wired |
| AC-1: halted→active round-trip succeeds | Off-to-on transition works |
| AC-2: active→halted with reason preserved | On-to-halt transition works |
| AC-3: halted→active (rollback) succeeds | Rollback gate dimension works |
| AC-1 audit fields non-empty | Audit trail survives NATS KV |
| All TestActivationAcceptance_ tests pass | Domain logic correct |

### FAIL (exit code 1)

Any single failure increments the error counter. The smoke runs all phases
regardless of intermediate failures (does not abort on first fail), then
reports the total failure count at the end.

Common failure causes:

| Symptom | Likely Cause | Fix |
|---------|-------------|-----|
| Gateway not ready | Stack not running | `make up` |
| NATS unreachable | Stack not running | `make up` |
| Control surface 404/503 | Gateway not seeded or NATS KV bucket missing | `make seed` |
| PUT returns non-200 | Gateway routing or NATS KV write failure | Check `make logs SERVICE=gateway` |
| Reason mismatch | NATS KV eventual consistency (very rare) | Retry once |
| Unit tests fail | Domain code regression | Check `go test -v ./internal/domain/execution/...` |

## What the Smoke Covers

| Dimension | Covered | How |
|-----------|---------|-----|
| Gate off→on | Yes | Phase 2 (AC-1) |
| Gate on→halt | Yes | Phase 3 (AC-2) |
| Gate halt→rollback | Yes (gate only) | Phase 4 (AC-3) |
| Audit field round-trip | Yes | Phase 2 audit check |
| Domain truth table | Yes | Phase 5 unit tests |
| Full binary rollback | Partial | AC-3 covers gate; adapter+creds require restart |
| Real venue order | No | Out of scope for S340 |
| Drain semantics on halt | No | L-2 limitation (documented) |
| Per-symbol gate | No | L-3 limitation (single global gate) |

## What the Smoke Does NOT Cover

These are intentional non-goals for S340:

1. **Real venue order submission** — S341 (controlled live verification).
2. **Binary restart with config change** — AC-3 validates the gate dimension only; full rollback requires operator action documented in S338.
3. **Concurrent gate mutations** — Single-operator model assumed.
4. **Fail-open behavior on KV unavailability** — Documented as accepted risk in S339 (L-4).
5. **Drain semantics** — In-flight intents during halt may still complete (S339 L-2).
6. **Credential rotation** — Credentials are immutable per process lifetime.

## Operational Limitations

| ID | Limitation | Severity | Mitigation |
|----|-----------|----------|-----------|
| L-S340-1 | Smoke validates gate dimension only, not full three-dimension surface | Low | Unit tests cover all dimensions; binary restart is operator procedure |
| L-S340-2 | Reason field match in AC-2 is best-effort (warns on mismatch, does not fail) | Low | NATS KV is strongly consistent for single-key writes |
| L-S340-3 | No negative test for degraded mode via HTTP | Low | Unit test AC-5 covers this path |
| L-S340-4 | Smoke requires live stack (not hermetic) | Medium | Unit tests are hermetic; smoke is operational proof |

## Relationship to Rollout Phases

From S338, the activation rollout has three phases:

| Phase | What happens | S340 coverage |
|-------|-------------|---------------|
| Phase 0: Halted Activation | Binary starts with venue adapter, gate halted | AC-1 step 1 proves this posture |
| Phase 1: Single-Order Enablement | Gate active, one fill, gate halted | Not covered (S341) |
| Phase 2: Observation Window | Gate active, sustained operation | Not covered (S341+) |

The S340 smoke proves that the **control surface is operationally sound**
before any of these phases execute with real venue orders.
