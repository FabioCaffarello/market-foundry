# Derive Integration Wave — Charter and Scope Freeze

> Opens the Derive Integration Wave with frozen scope, governing
> questions, ordered block plan, and binding constraints.
>
> Predecessor: S363 (Strategy/Signal Integration Evidence Gate — ALL OBJECTIVES MET).
> Wave type: Implementation (bounded).
> Date: 2026-03-22.

---

## 1. Strategic Context

The Strategy/Signal Integration Wave (S358–S363) closed with all objectives
met: 8/8 governing questions answered at HIGH confidence, 8/10 capabilities
FULL, 2/10 SUBSTANTIAL, 31 tests introduced (all PASS), 11/11 contract
invariants verified, zero regressions.

That wave proved the **consumer side** of the strategy-to-execution path:
`StrategyConsumerActor` subscribes to `STRATEGY_EVENTS`, evaluates via
`PaperOrderEvaluator`, produces `ExecutionIntent`, and routes through the
full safety gate pipeline to the venue adapter. The E2E tests (S362) publish
**synthetic** `StrategyResolvedEvent` payloads into NATS JetStream and verify
the execute scope handles them correctly.

What remains unproven is the **producer side**: the derive binary's
strategy resolvers (`MeanReversionEntryResolverActor`) publishing real
`StrategyResolvedEvent` payloads that flow through the proven consumer path.
The code exists — resolvers, publisher actor, NATS adapter — but it has never
been exercised as a connected pipeline with the execute scope consuming the
output.

| Proven (S358–S363) | Unproven (this wave) |
|---|---|
| Execute consumes StrategyResolvedEvent correctly | Derive produces StrategyResolvedEvent that satisfies S359 contract |
| Direction-to-side mapping is deterministic | Derive publisher output matches field-level contract |
| 11 invariants hold on consumer side | 11 invariants hold on producer side |
| Safety gates preserve behavior for strategy-sourced intents | Store materializes derive-produced events correctly |
| Prometheus observability tracks strategy execution flow | Gateway read path returns derive-produced strategy state |
| E2E proof with synthetic events | E2E proof with real derive-produced events |

The S363 evidence matrix explicitly flagged **DG-11** (no derive-side strategy
event production) as the primary deferred gap and recommended the Derive
Integration Wave as the natural next step.

---

## 2. Wave Identity

| Field | Value |
|---|---|
| **Wave name** | Derive Integration Wave |
| **Wave type** | Implementation (bounded) |
| **Phase** | 37 |
| **Predecessor wave** | Strategy/Signal Integration (S358–S363) |
| **Predecessor verdict** | ALL OBJECTIVES MET (8/8 questions HIGH, zero regressions) |
| **Opening stage** | S364 (this document) |
| **Estimated stages** | S364 (charter) → S365 → S366 → S367 → S368 → S369 (gate) |
| **Strategic goal** | Prove derive produces StrategyResolvedEvent that drives execution end-to-end |

---

## 3. What This Wave Must Prove

One sentence: **the derive binary is a correct, contract-compliant producer of
`StrategyResolvedEvent` and the full analytical-to-execution pipeline works as
a connected system.**

This means:

1. Derive's `MeanReversionEntryResolverActor` produces events whose fields
   satisfy every S359 contract invariant (INV-1 through INV-11).
2. Derive's `StrategyPublisherActor` publishes to the correct NATS subject
   with correct deduplication keys.
3. The store scope materializes derive-produced events into KV buckets with
   monotonicity guards.
4. The gateway read path returns derive-produced strategy state via HTTP.
5. An end-to-end test demonstrates: signal ingestion → derive processing →
   strategy event production → execute consumption → paper order fill.

---

## 4. Ordered Block Plan

### DI-1: Producer Spec and Derive Ownership (S365)

**Objective**: Audit derive's strategy resolver and publisher against the S359
contract. Produce a field-level compliance matrix showing which invariants
the derive producer satisfies and which need adjustment.

**Deliverables**:
- Field-level compliance matrix (derive output vs. S359 contract)
- Invariant coverage report (INV-1 through INV-11 on producer side)
- Any contract mismatches documented with fix plan

**Scope boundary**: Audit and document only. No code changes in this block
unless a contract violation is trivially fixable (< 5 lines).

### DI-2: Canonical Derive Producer Wiring (S366)

**Objective**: Fix any contract mismatches found in DI-1. Add unit tests
proving derive's strategy resolver output satisfies all 11 S359 invariants.
Ensure the publisher actor produces correctly-shaped NATS messages.

**Deliverables**:
- Code fixes for any contract mismatches
- Unit tests for derive strategy resolver (invariant-by-invariant)
- Unit tests for derive strategy publisher (subject format, dedup key, payload shape)

**Scope boundary**: Derive scope only. No changes to execute, store, or
gateway. No new strategy families.

### DI-3: Store/Gateway/Read-Path Verification (S367)

**Objective**: Verify that derive-produced `StrategyResolvedEvent` flows
correctly through the store materialization pipeline and is queryable via
gateway HTTP endpoints.

**Deliverables**:
- Integration test: derive-produced event → store projection → KV bucket
- Verification of monotonicity guard on derive-produced timestamps
- Gateway query test: HTTP endpoint returns derive-produced strategy state

**Scope boundary**: Store and gateway scopes only. Uses derive-shaped payloads
but does not require derive binary running.

### DI-4: Analytical-to-Execution End-to-End Proof (S368)

**Objective**: Demonstrate the full connected pipeline: signal ingestion →
derive processing → strategy event production → NATS → execute consumption →
paper order fill. This is the capstone proof of the wave.

**Deliverables**:
- End-to-end integration test with real derive pipeline
- Correlation chain verification (signal → strategy → execution)
- Prometheus metrics verification across the full path
- Invariant verification across the full path

**Scope boundary**: Integration test only. No new features, no new families,
no multi-binary orchestration beyond what's needed for the test.

### DI-5: Evidence Gate Final (S369)

**Objective**: Formal evidence gate evaluating all wave deliverables against
this charter. Audit capabilities, governing questions, invariant coverage,
regression status, and residual gaps.

**Deliverables**:
- Evidence gate document
- Evidence matrix with capability classifications
- Residual gaps catalog
- Next ceremony recommendation

---

## 5. Binding Constraints

| Constraint | Rationale |
|---|---|
| **Single strategy family** (mean_reversion_entry) | Depth over breadth; prove the pattern, others follow mechanically |
| **Single signal family** (RSI) | Same rationale; canonical pair from S359 |
| **Paper execution only** | No mainnet; paper adapter is sufficient proof |
| **No new domain types** | Wiring existing domains, not creating new ones |
| **No derive runtime redesign** | Derive actors work; this wave proves contract compliance, not architecture |
| **No multi-binary orchestration** | Tests compose in-process; Docker Compose is a separate wave |
| **Sequential blocks** | Each block depends on the previous; no parallelism |

---

## 6. Prerequisites

All prerequisites are met:

| Prerequisite | Status | Evidence |
|---|---|---|
| S359 contract (field-level mapping, 11 invariants) | DELIVERED | `source-selection-and-canonical-integration-contract.md` |
| S360 consumer spec (StrategyConsumerActor, PaperOrderEvaluator) | DELIVERED | `controlled-source-to-execution-wiring.md` |
| Derive strategy resolver implementation | EXISTS | `strategy_resolver_actor.go` (MeanReversionEntryResolverActor) |
| Derive strategy publisher implementation | EXISTS | `strategy_publisher_actor.go`, `natsstrategy/publisher.go` |
| Store strategy projection | EXISTS | `strategy_projection_actor.go`, store pipeline declarations |
| NATS strategy registry | EXISTS | `natsstrategy/registry.go` (3 families registered) |
| All 5 binaries build | VERIFIED | S363 regression check (2025-03-22) |

---

## 7. Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Derive resolver output doesn't match S359 contract | MEDIUM | HIGH | DI-1 audit catches mismatches before code changes |
| NATS subject mismatch between derive publisher and execute consumer | LOW | HIGH | Both use the same registry; subject format is declarative |
| Store projection rejects derive-produced events (staleness, dedup) | LOW | MEDIUM | Monotonicity guard is timestamp-based; derive timestamps are source-derived |
| Test infrastructure insufficient for multi-scope integration | MEDIUM | MEDIUM | S362 E2E tests already prove the test infrastructure pattern |

---

## 8. Success Criteria

The wave succeeds when:

1. Derive's strategy resolver output satisfies all 11 S359 invariants (proven by unit tests).
2. Derive's strategy publisher produces correctly-shaped NATS messages (proven by unit tests).
3. Store materializes derive-produced events without rejection (proven by integration test).
4. Gateway returns derive-produced strategy state via HTTP (proven by integration test).
5. Full pipeline test demonstrates signal → derive → strategy → execute → fill (proven by E2E test).
6. Zero regressions across the codebase.
7. Evidence gate passes with all governing questions answered at HIGH confidence.

---

## References

- [Strategy/Signal Integration Evidence Gate (S363)](strategy-signal-integration-evidence-gate.md)
- [Strategy/Signal Integration Evidence Matrix](strategy-signal-integration-evidence-matrix-residual-gaps-and-next-ceremony.md)
- [Source Selection and Canonical Integration Contract (S359)](source-selection-and-canonical-integration-contract.md)
- [Controlled Source-to-Execution Wiring (S360)](controlled-source-to-execution-wiring.md)
- [Derive Pipeline Pattern](derive-pipeline-pattern.md)
- [Derive Family Processor Pattern](derive-family-processor-pattern.md)
