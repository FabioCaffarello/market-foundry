# Strategy-to-Execution Decision Quality Wave -- Charter and Scope Freeze

**Wave**: Strategy-to-Execution Decision Quality
**Charter stage**: S469
**Status**: OPEN -- SCOPE FROZEN
**Date**: 2026-03-25
**Predecessor wave**: Session Access & Verification Closure (S464--S468)

---

## 1. Strategic Context

The Foundry runtime reached operational maturity through a sequence of waves that established:

- **OMS Foundation** (S382--S388): canonical order model, lifecycle invariants, persistence, rejection path.
- **Venue execution proof** (S389--S426): testnet and futures venue connectivity, fill/rejection/partial-fill evidence on unified runtime.
- **Production hardening and mainnet** (S427--S448): mainnet dry-run, live trading authorization, supervised live sessions.
- **Session intelligence** (S459--S463): session metadata, PO automation, audit bundles with consistency checks.
- **Session access & verification** (S464--S468): HTTP-accessible audit surface, batch audit, verification parameterization.

The runtime, venue path, session surfaces, and operational explainability are mature. The remaining systemic weakness is in the **quality, traceability, and consistency of the signal-to-execution decision chain**.

Today the pipeline flows:

```
Signal -> Decision -> Strategy -> Risk -> ExecutionIntent -> Venue
```

Each layer preserves semantic context (CorrelationID, CausationID, StrategyType, DecisionSeverity). However:

1. **Decision lineage is implicit** -- there is no dedicated query to trace a fill back to its originating signal chain.
2. **Decision review is scattered** -- the operator must reconstruct the chain from multiple KV buckets, NATS subjects, and audit bundle fragments.
3. **Cross-domain consistency is unchecked** -- no runtime validation that a Strategy's DecisionInput[] matches persisted Decision events, or that RiskInput severity matches its source Decision.
4. **Evidence bundle lacks decision context** -- SessionAuditBundle covers lifecycle and fees but not the decision rationale that produced each intent.

This wave targets these gaps with a small, disciplined scope.

---

## 2. Problem Statement

**The Foundry can execute orders reliably but cannot yet answer "why was this specific order placed, based on which signals, through which decision path, and was the chain internally consistent?" in a single, auditable query.**

This matters because:
- Post-session review requires manual reconstruction across domain boundaries.
- Consistency between decision severity at strategy resolution time and decision severity at execution time is assumed but never verified.
- PriceSource used for fill simulation is not recorded in the ExecutionIntent, making price realism opaque.
- The audit bundle answers "what happened" but not "why it happened."

---

## 3. Wave Blocks

The wave is structured as 4 ordered blocks plus 1 evidence gate:

### Block 1: Decision Lineage and Causality Model (S470)

Formalize the lineage model that connects a VenueOrderFilledEvent back through ExecutionIntent -> RiskAssessment -> Strategy -> Decision -> Signal using the existing CorrelationID/CausationID chain.

**Deliverables**:
- Domain type: `DecisionLineage` -- a read-only projection that assembles the full chain for a given CorrelationID.
- Lineage assembler that reads from existing KV buckets (DECISION_LATEST, STRATEGY_LATEST, EXECUTION_INTENT_LATEST).
- Unit tests proving chain integrity for: triggered path, flat/no-op path, rejection path.

**Does NOT include**: new NATS subjects, new persistence backends, UI.

### Block 2: Decision Review Surface and Evidence Bundle (S471)

Expose the decision lineage as a queryable HTTP surface and integrate it into the session audit bundle.

**Deliverables**:
- HTTP endpoint: `GET /session/{id}/decisions` -- returns decision lineage entries for a session.
- Augmented `SessionAuditBundle` with `DecisionSummary` field (count by outcome, severity distribution, confidence histogram).
- Per-intent decision context in lifecycle entries (strategy type, decision severity, confidence at evaluation time).
- Tests for lineage query, bundle augmentation, and missing-chain graceful degradation.

**Does NOT include**: cross-session decision comparison, analytics dashboards, historical decision storage beyond KV.

### Block 3: Cross-Domain Consistency Checks (S472)

Add runtime validation that the decision chain is internally consistent at the execution boundary.

**Deliverables**:
- Consistency check: DecisionSeverity in RiskInput matches the severity in the referenced Decision event.
- Consistency check: StrategyType in RiskInput matches the type in the referenced Strategy event.
- Consistency check: CorrelationID chain is unbroken from intent back to at least one signal.
- Consistency check: PriceSource reference recorded in ExecutionIntent metadata.
- Integration into existing session verification checks (S461 verification framework).
- Tests for each check: pass, fail, degraded (missing upstream data).

**Does NOT include**: blocking execution on consistency failure (checks are observational, not fail-closed).

### Block 4: Evidence Gate (S473)

Formal gate evaluating whether the wave achieved its objectives.

**Deliverables**:
- Evidence matrix with capability grading.
- Residual gap register.
- Verdict and recommendation for next direction.

---

## 4. Scope Freeze

### What is IN scope

| Item | Block | Rationale |
|------|-------|-----------|
| DecisionLineage read-only projection | 1 | Core capability gap |
| Lineage assembler from existing KV | 1 | Uses existing persistence |
| Decision lineage HTTP endpoint | 2 | Operator access |
| DecisionSummary in audit bundle | 2 | Answers "why" alongside "what" |
| Per-intent decision context | 2 | Enriches lifecycle view |
| Severity consistency check | 3 | Validates chain integrity |
| StrategyType consistency check | 3 | Validates chain integrity |
| CorrelationID chain check | 3 | Validates causality |
| PriceSource metadata recording | 3 | Closes price realism opacity |
| Evidence gate | 4 | Wave closure discipline |

### What is NOT in scope (frozen)

See [decision-quality-capabilities-questions-and-non-goals.md](decision-quality-capabilities-questions-and-non-goals.md) for the full non-goals list.

**Summary of exclusions**: OMS expansion (limit orders, cancel path, modify path), multi-exchange support, new instrument families, broad analytics dashboards, structural refactoring, new persistence backends, fail-closed consistency enforcement, real-time alerting, ML-based decision scoring.

---

## 5. Ordering and Dependencies

```
S469 (this charter) ─── scope frozen
  │
  ├─► S470: Decision lineage and causality model
  │     │
  │     └─► S471: Decision review surface and evidence bundle
  │           │
  │           └─► S472: Cross-domain consistency checks
  │                 │
  │                 └─► S473: Evidence gate
```

S470 must complete before S471 (lineage model is a dependency for the HTTP surface).
S471 must complete before S472 (consistency checks integrate into the verification framework established in S471).
S473 depends on all prior blocks.

**Estimated test budget**: 25-35 new tests across S470-S472.

---

## 6. Success Criteria

The wave succeeds when:

1. An operator can query the decision lineage for any execution intent via HTTP.
2. The session audit bundle includes decision summary context.
3. Cross-domain consistency checks are integrated into session verification.
4. PriceSource is recorded in ExecutionIntent metadata.
5. Zero regressions in existing test suites.

---

## 7. Constraints and Guard Rails

1. **No new NATS subjects** -- lineage is assembled from existing KV projections.
2. **No new persistence backends** -- KV is sufficient for decision lineage at current scale.
3. **No blocking enforcement** -- consistency checks are observational only.
4. **No UI or dashboard work** -- HTTP API only.
5. **No changes to the derive binary** -- all work is in the execute binary and gateway.
6. **Additive only** -- no refactoring of existing domain types unless strictly necessary for lineage assembly.

---

## 8. Risk Register

| Risk | Severity | Mitigation |
|------|----------|------------|
| KV bucket data may be missing for older correlation chains | LOW | Graceful degradation: lineage assembler returns partial chain with explicit gaps |
| Lineage assembly may be slow for high-throughput sessions | LOW | Assemble on-demand per query, not as background projection |
| Consistency checks may surface false positives from timing | LOW | Checks tolerate eventual consistency with configurable grace window |

---

## 9. References

- [Decision Quality Capabilities, Questions, and Non-Goals](decision-quality-capabilities-questions-and-non-goals.md)
- [S468 Evidence Gate Report](../stages/stage-s468-session-access-verification-evidence-gate-report.md)
- [OMS Foundation Charter](../stages/stage-s382-oms-foundation-charter-report.md)
- [Session Intelligence Charter](../stages/stage-s459-session-intelligence-charter-report.md)
- [Session Access & Verification Charter](../stages/stage-s464-session-access-verification-charter-report.md)
