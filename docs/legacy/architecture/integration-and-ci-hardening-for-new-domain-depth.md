# Integration and CI Hardening for New Domain Depth

## Purpose

Document the CI hardening measures applied in S237 to ensure the domain depth
introduced in S234–S236 (decision severity/rationale, strategy decision context
propagation, risk decision traceability) is continuously validated beyond local
informal proofs.

## Context

After S236, three layers of domain depth exist:

| Layer     | New Fields                                | Introduced |
|-----------|-------------------------------------------|------------|
| Decision  | `severity`, `rationale`                   | S234       |
| Strategy  | `DecisionInput.Severity`, `.Rationale`    | S235       |
| Risk      | `StrategyInput.DecisionSeverity`, `.DecisionRationale` | S236 |

Until S237, proof of this depth relied on:

- Local `make test` (unit tests)
- Local `make test-integration` (embedded NATS integration tests)
- Local `make smoke-analytical` (E2E with docker-compose)

Remote CI validated unit tests, codegen golden equivalence, and the smoke-analytical
E2E — but the smoke script did **not** verify the new domain fields, and integration
tests were not run remotely at all.

## Changes Applied

### 1. `make test-integration` added to CI (`.github/workflows/ci.yml`)

A new `integration-tests` job runs `make test-integration` on every push/PR to `main`.

- Runs in parallel with `unit-tests` and `codegen-golden` (no dependency)
- Uses the same Go version as all other jobs
- Requires no external infrastructure (embedded NATS via Go test)
- Cost: ~30s additional CI time

**Rationale:** The integration test suite (`pipeline_integration_test.go`) exercises
the full execution pipeline (risk → evaluate → simulate → fill) with embedded NATS.
This is the only test path that validates cross-actor message propagation with a real
message broker, making it critical for confidence in the derive flow.

### 2. Smoke-analytical script updated (`scripts/smoke-analytical-e2e.sh`)

#### 2a. Decision required fields expanded

The `validate_analytical_family` call for decisions now requires `severity` and
`rationale` in the HTTP response structure. A response missing these fields will
fail the smoke test.

Before:
```
type|source|symbol|timeframe|outcome|confidence|signals|metadata|final|timestamp
```

After:
```
type|source|symbol|timeframe|outcome|confidence|severity|rationale|signals|metadata|final|timestamp
```

#### 2b. New Phase 7: Domain Depth Validation

A dedicated validation phase checks that the new domain depth survives the full
pipeline (evaluator → NATS → writer → ClickHouse → reader → HTTP):

1. **Decision depth**: Verifies `severity` and `rationale` are non-empty in all
   returned decision rows.
2. **Strategy → Decision context**: Verifies the `decisions` JSON array inside
   strategy responses contains `severity` and `rationale` fields.
3. **Risk → Decision context**: Verifies `metadata.decision_severity` is present
   in risk assessment responses.

Each check uses a three-tier result classification:
- `ALL_OK` — all rows carry the expected depth → PASS
- `PARTIAL` — some rows carry it (expected during transition) → WARN
- `NO_DATA` — no data available to validate → WARN (non-blocking)
- `NONE`/`ERROR` — field missing despite data existing → FAIL

## What Is NOT Covered

- **ClickHouse column-level assertions**: The smoke test validates JSON structure
  at the HTTP layer, not raw ClickHouse column contents. This is intentional — the
  HTTP layer is the contract surface.
- **Performance regression gates**: No latency thresholds or query time assertions.
  The `Server-Timing` header is checked for presence only.
- **Mutation testing or fault injection**: Not in scope for light hardening.
- **Multi-family breadth**: Only RSI Oversold / Mean Reversion Entry / Position
  Exposure are validated. New families will need their own validation when added.

## CI Job Matrix After S237

| Job                 | Trigger            | What It Proves                                    |
|---------------------|--------------------|---------------------------------------------------|
| `unit-tests`        | push/PR to main    | All module unit tests pass                        |
| `integration-tests` | push/PR to main    | Cross-actor pipeline with embedded NATS           |
| `codegen-golden`    | push/PR to main    | Spec validation + golden snapshot equivalence     |
| `smoke-analytical`  | push/PR (after unit) | Full E2E: NATS → writer → ClickHouse → HTTP + domain depth |

## Risk Assessment

**Risk of adding `integration-tests` to CI:** Low. The test suite uses embedded NATS
(no external deps), runs in ~30s, and has been passing locally throughout S234–S236.

**Risk of stricter smoke validation:** Low. The new Phase 7 uses WARN for partial/no-data
scenarios, so it won't block CI during transition periods. Only a complete absence of
the fields despite data existing will cause a failure.
