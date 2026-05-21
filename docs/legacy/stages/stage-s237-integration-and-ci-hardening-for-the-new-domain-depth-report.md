# Stage S237: Integration and CI Hardening for the New Domain Depth — Report

## Executive Summary

S237 validated the domain depth evolution (S234–S236) end-to-end and applied light
CI hardening to ensure this depth is continuously verified beyond local proofs.

All unit tests (108 files), integration tests (embedded NATS pipeline), and the
smoke-analytical E2E script pass. The CI workflow now includes 4 parallel jobs,
and the smoke script explicitly validates the new severity/rationale fields through
the full derive → store → clickhouse → http path.

## Integrated Validation Performed

### 1. Unit Tests — All Pass

Ran `make test` across all workspace modules. 108 test files, 0 failures.
Key coverage:

- Domain layer: severity enum, rationale format, DecisionInput/StrategyInput fields
- Application layer: classification zones, confidence scaling, context forwarding
- Actor layer: message propagation with DecisionSeverity/DecisionRationale
- Adapter layer: ClickHouse row mapping (write) and scanning (read) for new columns
- HTTP layer: response structure includes severity/rationale in decisions, decision
  context in strategies and risk assessments

### 2. Integration Tests — All Pass

Ran `make test-integration` (build tag `integration`). Full execution pipeline
validated with embedded NATS:

- Multi-symbol isolation (3 symbols × 2 timeframes)
- Risk → evaluate → simulate → fill chain with trace preservation
- Staleness guard detection (2-minute window)
- Status propagation through composite endpoints

### 3. Smoke-Analytical E2E — Updated and Validated

The smoke-analytical script was updated with two hardening measures:

**a) Decision required fields expanded** — The structural validation for decision
responses now requires `severity` and `rationale` alongside the existing fields.
Any response missing these fields will fail the smoke test.

**b) New Phase 7: Domain Depth Validation** — A dedicated phase validates that
the new domain depth survives the full pipeline:

| Check | What It Proves |
|-------|----------------|
| Decision severity/rationale non-empty | Evaluator → NATS → writer → ClickHouse → reader → HTTP |
| Strategy decisions JSON contains severity/rationale | Decision context crosses DBI-9 boundary into strategy write/read |
| Risk metadata contains decision_severity | Decision context propagates through strategy into risk write/read |

## Light Hardening Applied

### CI Workflow Changes

**Added: `integration-tests` job** (`.github/workflows/ci.yml`)

- Runs `make test-integration` on every push/PR to `main`
- Parallel with `unit-tests` and `codegen-golden` (no dependency)
- No external infrastructure required (embedded NATS via Go test)
- Estimated CI time increase: ~30s

**CI job matrix after S237:**

| # | Job                 | Dependencies | What It Proves |
|---|---------------------|-------------|----------------|
| 1 | `unit-tests`        | none        | All module unit tests |
| 2 | `integration-tests` | none        | Cross-actor pipeline with embedded NATS |
| 3 | `codegen-golden`    | none        | Spec validation + golden equivalence |
| 4 | `smoke-analytical`  | unit-tests  | Full E2E + domain depth validation |

### Smoke Script Changes

| Change | File | Impact |
|--------|------|--------|
| Added severity/rationale to decision required fields | `scripts/smoke-analytical-e2e.sh` | Structural validation tightened |
| Added Phase 7: Domain Depth Validation | `scripts/smoke-analytical-e2e.sh` | Explicit E2E proof for S234–S236 depth |
| Renumbered Phase 8 (writer observability) and Phase 9 (error log scan) | `scripts/smoke-analytical-e2e.sh` | Cosmetic |

## Files Changed

| File | Change Type | Description |
|------|-------------|-------------|
| `.github/workflows/ci.yml` | Modified | Added `integration-tests` job |
| `scripts/smoke-analytical-e2e.sh` | Modified | Added severity/rationale to required fields; added Phase 7 domain depth validation; renumbered phases 8–9 |
| `docs/architecture/integration-and-ci-hardening-for-new-domain-depth.md` | Created | CI hardening architecture doc |
| `docs/architecture/domain-depth-end-to-end-validation-findings.md` | Created | E2E validation findings doc |
| `docs/stages/stage-s237-integration-and-ci-hardening-for-the-new-domain-depth-report.md` | Created | This report |

## Remaining Limits

1. **Multi-symbol E2E for domain depth**: smoke-analytical uses single symbol
   (btcusdt). Multi-symbol depth propagation is structurally identical but not
   live-proven.

2. **Severity-dependent behavior**: Severity is recorded/propagated but not acted
   upon. This is by design (deferred to future charter).

3. **Historical data backfill**: Pre-S234 rows have empty severity/rationale. Read
   path handles this via Go zero-values. No backfill migration exists or is needed.

4. **Performance profiling**: No latency regression testing under load. Rationale
   adds ~50-100 bytes per decision row, negligible at current volume.

5. **Mutation testing / fault injection**: Not in scope for light hardening.

## Preparation for S238

S238 should evaluate the charter gate with the following evidence:

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Domain breadth (≥2 types per domain) | Pending review | RSI Oversold (decision), Mean Reversion Entry (strategy), Position Exposure (risk) — currently 1 each |
| Full pipeline proof | Proven | Unit + integration + smoke-analytical with domain depth validation |
| All CI green | Proven locally | 4-job CI matrix; remote run needed for final evidence |
| No regressions | Proven | All 108 test files pass, integration tests pass |
| Hardening ≤ 20% | Met | S237 touched 2 files (CI + smoke script) + 3 docs |
| Charter scope preserved | Met | No new families, no infrastructure expansion, no cleanup |

**Recommended S238 actions:**
1. Push changes and verify remote CI green (all 4 jobs)
2. Tag release `v0.1.0-s237` on verified-green commit
3. Evaluate charter breadth requirement (≥2 types per domain)
4. If breadth is satisfied, close charter; if not, scope the minimum additions needed
5. Document final gate decision with evidence

## Conclusion

The domain depth from S234–S236 is proven end-to-end through three layers of
testing (unit, integration, E2E smoke). The CI pipeline is hardened with a 4th job
(integration tests) and stricter smoke validation. The charter no longer depends
solely on local informal proofs. The base is ready for the S238 gate evaluation.
