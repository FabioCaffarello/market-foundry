# Next Platform Wave Recommendations

> Evidence-based recommendations for what should follow the S101–S105 hardening wave. Based on the post-S100 platform readiness review and the current state of open debts.

---

## Decision Framework

Two consolidation waves (S96–S99, S101–S105) have established the Foundry's structural and operational foundation. The platform has:
- Canonical runtime lifecycle, composition roots, and catalog-driven assembly.
- Documented operational contracts, observability surfaces, and error handling policy.
- Hardened config validation with exported catalog API.
- Refined governance playbooks with decision gates and anti-patterns.

**What the platform does NOT have:** Evidence that the pipeline works end-to-end.

The structural investment is complete. Continuing to refine infrastructure without running the pipeline would be diminishing returns — there are no significant structural debts blocking progress, and the remaining debts (correlation IDs, integration tests, error classification) are best addressed in the context of real operational experience.

**Criteria for the next wave:**
1. It must produce user-visible or operationally-measurable value (not more infrastructure).
2. It must exercise the structural patterns established in S96–S105 under real conditions.
3. It must surface integration issues that static analysis and unit tests cannot detect.
4. Its success or failure must be observable through the diagnostic surfaces built in S102–S103.

---

## Priority 1: Vertical Slice Execution (RECOMMENDED NOW)

**What:** Run the complete pipeline chain end-to-end: `candle → rsi → rsi_oversold → mean_reversion_entry → position_exposure → paper_order`.

**Why this is the right next step:**

1. **Validates two waves of infrastructure investment.** S96–S105 reduced the cost of evolution and hardened operational contracts. The return on that investment is realized only when the system actually processes market data through the full chain.

2. **Exposes integration issues.** Cross-runtime event routing (ingest → derive → store, derive → derive → derive → execute) through NATS JetStream cannot be validated by static analysis, raccoon-cli, or unit tests. Only running the pipeline reveals subject mismatches, consumer configuration errors, and actor message routing bugs.

3. **Exercises the diagnostic surfaces.** The `/diagz` endpoint, runtime-tagged logs, and RecordError tracking from S102–S103 are designed for pipeline debugging. Running the slice validates that these tools actually help.

4. **Uses the playbooks.** If any family in the chain requires adjustment, the expansion playbooks from S99/S105 are exercised under real conditions. This is the best test of playbook quality.

**Scope:**

| Component | Family | Runtime | Status |
|-----------|--------|---------|--------|
| Market data ingestion | candle, tradeburst, volume | ingest | Implemented |
| Evidence derivation | candle, volume | derive | Implemented |
| Signal generation | rsi | derive | Implemented |
| Decision evaluation | rsi_oversold | derive | Implemented |
| Strategy resolution | mean_reversion_entry | derive | Implemented |
| Risk assessment | position_exposure | derive | Implemented |
| Execution | paper_order | derive + execute | Implemented |
| Read model projection | All families | store | Implemented |
| HTTP query surface | All families | gateway | Implemented |
| Config management | configctl-sync/v1 | configctl | Implemented |

All domain implementations exist. The work is integration, configuration, and validation — not new code.

**What this validates:**
- Catalog-driven assembly activates all families from config.
- Cross-domain event routing through NATS JetStream is correct.
- Health trackers report meaningful activity status.
- `/diagz` shows pipeline health across all runtimes.
- Config activation and dependency validation work as documented.

**Estimated scope:** 2–4 stages (configuration, integration testing, smoke testing, validation).

---

## Priority 2: Operational Confidence (RECOMMENDED AFTER VERTICAL SLICE)

**What:** Based on what the vertical slice reveals, add the minimal observability and testing needed for confident ongoing operation.

**Why conditional on vertical slice:**

The specific observability and testing investments that matter will be determined by what breaks, what's hard to debug, and what the diagnostic surfaces fail to show during the vertical slice execution. Investing in observability before running the pipeline risks building the wrong diagnostic tools.

**Likely investments (based on current debt analysis):**

| Investment | Trigger | Estimated Cost |
|-----------|---------|---------------|
| Correlation ID in logs | Cross-runtime event tracing is painful | 1 stage |
| Composition root smoke tests | Wiring bugs found during slice setup | 1 stage |
| Cross-registration coherence test | Family registration mismatch found | Half stage |
| Pipeline throughput counter | Cannot determine if pipeline is flowing vs. stalled | Half stage |

**Guard rail:** Only invest in what the vertical slice proves is needed. Do not add observability speculatively.

---

## Priority 3: MarketMonkey Absorption (CONDITIONAL)

**What:** Absorb the MarketMonkey codebase into the Foundry monorepo, applying S96–S105 structural patterns.

**Preconditions:**
1. Vertical slice running end-to-end reliably.
2. Operational confidence layer adequate for ongoing debugging.
3. Expansion playbooks validated through at least one real expansion during the slice.

**Why not now:** Absorbing external code into a system that hasn't proven its runtime behavior compounds integration risk. MarketMonkey patterns may conflict with Foundry conventions — these conflicts are easier to resolve when the Foundry's patterns are battle-tested, not just documented.

**Estimated scope:** 2–4 stages (assessment, planning, phased absorption, validation).

---

## Not Recommended Now

### Additional Infrastructure Hardening

The S101–S105 wave was a complete hardening pass. The remaining debts (PD-1 through PD-5 in `platform-gains-tradeoffs-and-open-debts.md`) are either low severity or best addressed in the context of real operational experience. Another infrastructure wave without running the pipeline would be diminishing returns.

**When to reconsider:** Only if the vertical slice reveals infrastructure gaps that block progress.

### Event Schema Formalization

Protobuf/Avro/JSON Schema for domain events. Still not justified — single producer per event type, single cluster, single language.

**When to reconsider:** When a second team or non-Go consumer needs domain events.

### OpenTelemetry / Distributed Tracing

Full tracing infrastructure with span propagation. The S102 observability foundation (runtime-tagged logs, /diagz, lifecycle signals) should be tested first.

**When to reconsider:** When the vertical slice proves that log-based debugging is insufficient for cross-runtime issues.

### Raccoon-CLI New Analyzers

The existing 18 analyzers are comprehensive. Adding new analyzers increases maintenance burden.

**When to reconsider:** When a specific pattern violation is caught only by manual review repeatedly.

### Documentation Consolidation / Pruning

The ~30 architecture documents + 105 stage reports are significant volume. However, pruning risks removing context that may be needed later.

**When to reconsider:** When documentation staleness becomes a measurable problem (developers following outdated docs, or docs contradicting code). Not before.

---

## Recommended Sequence

| Priority | Wave | Precondition | Expected Stages |
|----------|------|--------------|-----------------|
| 1 | Vertical slice execution | None | 2–4 |
| 2 | Operational confidence (targeted) | Vertical slice running | 1–2 |
| 3 | MarketMonkey absorption | Slice + confidence | 2–4 |

**Total estimated scope for next phase: 5–10 stages.**

This sequence maximizes the return on the S96–S105 infrastructure investment. Each wave is bounded, falsifiable, and builds on the previous one. The vertical slice validates the structure; the confidence layer validates the diagnostics; the absorption validates the growth playbooks.

---

## Closing Note

Two consolidation waves (10 stages) have built a platform that is documented, governed, and mechanically enforced. The platform is ready to run. The next decision should be driven by operational evidence, not structural impulse.

---

## Related Documents

- `post-s100-technical-platform-readiness-review.md` — readiness assessment
- `platform-gains-tradeoffs-and-open-debts.md` — gains, costs, and debts
- `next-technical-wave-recommendations.md` — S100 equivalent (still valid, unchanged)
