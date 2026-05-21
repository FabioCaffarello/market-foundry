# Production Hardening and Mainnet Readiness Audit Wave -- Charter and Scope Freeze

## Wave Identity

| Field | Value |
|---|---|
| Wave | Production Hardening and Mainnet Readiness Audit |
| Charter Stage | S427 |
| Planned Stages | S428--S431 |
| Predecessor Wave | Futures Venue Execution Proof (Post-Simplification), S421--S426 |
| Predecessor Verdict | PASS -- FULL DELIVERY |
| Date Opened | 2026-03-23 |

## Strategic Context

The Foundry has accumulated eleven consecutive wave passes since S370:

1. Multi-binary orchestration (S370--S375): PASS
2. Exchange listening + dry-run (S376--S381): PASS
3. OMS Foundation (S382--S388): PASS
4. Binance segmentation (S390--S395): PASS
5. Unified segment runtime (S398--S403): PASS, FULL DELIVERY
6. Testnet venue execution, Spot-first (S404--S409): PASS, SUBSTANTIAL DELIVERY
7. Production readiness hardening (S410--S414): PASS, FULL DELIVERY
8. Futures venue execution proof (S415--S420): PASS, SUBSTANTIAL DELIVERY
9. Runtime simplification and consolidation (S421--S420): PASS, FULL DELIVERY
10. Futures venue execution proof, post-simplification (S421--S426): PASS, FULL DELIVERY

The system has a complete, proven execution chain for both Spot and Futures segments: exchange ingress through lifecycle outcome to persistence and read-path on a unified multi-segment runtime with real venue connectivity on Binance testnet.

The S426 evidence gate closed with FULL DELIVERY (10/10 capabilities FULL), but carried forward 16 residual gaps (1 medium-severity, 15 low-severity). The single medium-severity gap (RG-13: fee semantic divergence between Spot commission and Futures cumQuote) is the most senior operational debt remaining in the system.

This wave exists to close the prioritized operational gaps, produce an explicit mainnet readiness audit, and establish a decision surface for future mainnet authorization -- without performing mainnet enablement itself.

## Wave Objective

1. Normalize fee semantics across Spot and Futures segments so that analytical consumers operate on a consistent model.
2. Establish per-segment health and operational readiness signals that allow operators to assess runtime state without manual inspection.
3. Conduct a formal mainnet readiness audit that evaluates the system against an explicit production checklist, identifies what must be resolved before any mainnet decision, and renders a readiness verdict.
4. Evaluate the KV history strategy (RG-3) and render a final keep-or-change decision for production use.
5. Close the wave with an evidence gate that consolidates all findings.

## Scope Freeze

### In Scope (Frozen)

The wave is organized into four execution blocks:

#### Block 1: Fee Normalization and Cross-Segment Consistency (S428)

Close RG-13 (the only medium-severity residual gap).

- Define a canonical fee model that accounts for Spot per-leg commission and Futures cumQuote semantics.
- Introduce a normalized fee representation in the FillRecord domain so that consumers do not need to branch on source.
- Preserve venue-specific raw fields in metadata for auditability.
- Align ClickHouse write-path to persist both raw and normalized fee values.
- Prove that read-path queries return consistent fee semantics regardless of segment.

**Exit Criteria**: RG-13 closed. Normalized fee field populated for both Spot and Futures fills. Read-path parity proven.

#### Block 2: Per-Segment Health and Operational Readiness Signals (S429)

Address the deferred per-segment health monitoring gap.

- Implement segment-scoped health status that surfaces per-source connectivity, last-event-age, and error rate.
- Wire health signals into the existing actor health tracker infrastructure.
- Expose health summary via a NATS query subject (e.g., `execution.query.health.segment`).
- Prove that a degraded segment does not mask the health of the other segment.
- Validate fail-closed behavior: if health signals are absent, the segment is reported as unhealthy.

**Exit Criteria**: Per-segment health query operational. Isolation between segment health states proven. Fail-closed on missing signals proven.

#### Block 3: Mainnet Readiness Audit and KV History Strategy Decision (S430)

Produce a formal readiness assessment. This stage does NOT enable mainnet; it produces the audit artifact.

- Evaluate the system against a mainnet readiness checklist covering:
  - Credential separation (testnet vs mainnet adapter configuration).
  - Capital controls and position-limit enforcement.
  - Kill switch and staleness guard behavior under mainnet-like conditions.
  - Audit trail retention depth (ClickHouse historical data lifecycle).
  - Graceful degradation under venue outages.
  - Incident response readiness (alerting, monitoring hooks).
  - Fee normalization completeness (depends on S428).
  - Segment health observability (depends on S429).
- Render a KV history strategy decision for RG-3:
  - Evaluate whether latest-only NATS KV semantics are sufficient for production operations.
  - If insufficient, define the migration path (JetStream stream history, dedicated KV history bucket, or delegation to ClickHouse only).
  - If sufficient, formally close RG-3 with rationale.
- Produce a mainnet readiness verdict document with explicit PASS/FAIL per checklist item and an overall readiness classification.

**Exit Criteria**: Mainnet readiness audit document produced. KV history decision rendered. All checklist items evaluated with evidence references.

#### Block 4: Evidence Gate (S431)

Evaluate the wave against its charter.

- Score each block's exit criteria.
- Classify residual gaps (if any) by severity.
- Render a wave verdict (PASS/FAIL with delivery classification).
- Recommend next ceremony direction based on audit findings.

**Exit Criteria**: Evidence matrix produced. Wave verdict rendered. Next-ceremony recommendation documented.

### Out of Scope (Frozen)

The following are explicitly excluded from this wave and must not be opened under any circumstance:

| Exclusion | Rationale |
|---|---|
| Mainnet enablement | This wave audits readiness; it does not activate mainnet paths. NG-1 remains enforced. |
| Multi-exchange support | Binance-only scope. NG-2 remains enforced. |
| OMS expansion (limit orders, amendments, cancel API) | Market-order-only scope. NG-6 remains enforced. |
| Advanced order types | Out of wave scope. |
| Dashboard or UI development | Operational signals are NATS-queryable; visualization is deferred. |
| Config or compose surface re-expansion | S416--S418 simplified to 3+3; this wave must not re-expand. |
| Large structural refactoring | Surgical changes only. |
| `/fapi/v1/userTrades` integration for true Futures commission | If fee normalization in S428 determines this is required, it becomes a recommendation for a future stage, not in-scope work for this wave. |
| Documentation governance (97 untracked docs, RG-16) | Separate governance ceremony, no runtime impact. |

## Dependency Chain

```
S427 (charter) ─→ S428 (fee normalization)
                      │
                      ├─→ S429 (health signals)
                      │       │
                      └───────┴─→ S430 (mainnet audit)
                                      │
                                      └─→ S431 (evidence gate)
```

S428 and S429 are sequentially ordered because the mainnet audit (S430) depends on both being complete. S428 precedes S429 because fee normalization is the highest-priority residual gap.

## Success Criteria

The wave passes if:

1. RG-13 (fee semantic divergence) is closed with a normalized model.
2. Per-segment health signals are operational and proven.
3. A mainnet readiness audit artifact exists with explicit per-item verdicts.
4. RG-3 (KV history strategy) has a formal decision.
5. The evidence gate renders a verdict with zero high-severity or medium-severity residual gaps.

## Risk Mitigation

| Risk | Mitigation |
|---|---|
| Fee normalization requires breaking domain changes | Additive field strategy: add normalized field, preserve raw. No removal of existing fields. |
| Health signals add overhead to the hot path | Health queries are pull-based (request/reply), not injected into the execution pipeline. |
| Mainnet audit reveals blockers beyond wave scope | The audit produces findings and recommendations. If blockers are found, they become input to the next wave charter, not scope expansion for this wave. |
| KV history decision requires JetStream migration | If migration is needed, this wave documents the plan; implementation is a future stage. |

## Ceremony Rules

- No stage may expand beyond its block definition without a charter amendment.
- Charter amendments require explicit justification and a documented decision in the stage report.
- The evidence gate (S431) must evaluate against this frozen scope, not against any informally expanded scope.
- All test evidence must be reproducible from the committed codebase.
