# Next Wave Recommendations After Vertical Slice 01

> Evidence-based recommendation for the Foundry's next phase, grounded in what the `candle-to-paper-order` slice actually proved and where gaps remain.

---

## 1. The Question

After S107–S111, the Foundry faces four possible next directions:

1. **Another vertical slice** — exercise a different pipeline path or multi-binding scenario
2. **Capability absorption** — bring in new domains, sources, or external systems (e.g., MarketMonkey)
3. **Technical hardening** — close remaining debts, add observability, formalize schemas
4. **Product evolution** — build user-facing features on top of the proven architecture

The right answer depends on what the first slice actually proved and what it left unresolved.

---

## 2. What the Evidence Says

### 2.1 The Architecture Is Structurally Sound

- Zero domain logic bugs found
- All 6 runtimes compile and pass tests
- Actor, publisher, projection, and query patterns are repeatable
- Governance tooling enforces structural invariants
- Expansion playbooks exist and have been validated through the slice process

**Implication:** There is no structural reason to delay forward progress. The patterns work.

### 2.2 The Architecture Is Operationally Unproven

- No live pipeline run has been performed
- Cross-runtime behavior under real messaging is untested
- Cold-start, failure recovery, and timing edge cases are unexercised

**Implication:** Any expansion that adds new runtimes, domains, or integrations compounds operational uncertainty. The existing pipeline must run live first.

### 2.3 One Safety Debt Is Blocking

- Execute actor safety logic (kill switch, staleness guard, timeout) has zero unit tests
- This code gates order placement — the most consequential action in the system

**Implication:** Expanding the execution pipeline without testing this code is irresponsible.

### 2.4 Remaining Debts Are Bounded

- 8 deferred items are tracked with explicit triggers
- None of the P2/P3 debts block forward progress
- The P1 debts (query client generics, composition root tests) can be addressed incrementally

**Implication:** A comprehensive hardening wave is not justified. Address debts as they become relevant.

---

## 3. Recommendation

### Phase 1: Close the Operational Gap (Immediate)

**Objective:** Prove the pipeline works live, not just structurally.

| Step | Action | Rationale |
|------|--------|-----------|
| 1 | Write execute actor unit tests (D1) | Clear the only P0 safety debt before any execution-path work |
| 2 | Run `docker compose up` with full stack | First live pipeline execution |
| 3 | Activate `binancef.btcusdt.60` binding via HTTP | Exercise config-driven activation under real conditions |
| 4 | Observe pipeline for 2+ candle windows | Verify event flow: observation → evidence → signal → decision → strategy → risk → execution → fill |
| 5 | Query all endpoints, verify data consistency | Prove read models materialize correctly |
| 6 | Verify `/statusz` and `/diagz` surfaces | Confirm diagnostic visibility under real load |
| 7 | Document any runtime issues found | Capture evidence for next decisions |

**Exit criteria:** Pipeline runs end-to-end with real NATS and real market data. All query endpoints return valid, consistent data. Diagnostic surfaces show accurate pipeline health.

**If this phase reveals significant issues:** Fix them before proceeding. Do not expand on a pipeline that doesn't run.

### Phase 2: Controlled Product Evolution (After Operational Proof)

**Objective:** Use the proven architecture to deliver value, not to build more architecture.

The evidence supports **controlled product evolution** as the next wave, not another vertical slice or technical hardening wave. Here's why:

- **Another vertical slice** is unnecessary — the first slice already exercised all 8 domain families, all 6 runtimes, and all communication patterns. A second slice covering a different signal/strategy combination would exercise the same patterns with different data. The marginal architectural proof is low.

- **Capability absorption (MarketMonkey)** is premature — absorbing external code into an architecture that hasn't run live compounds integration risk. MarketMonkey patterns are easier to reconcile after Foundry patterns are battle-tested.

- **Technical hardening** is not justified as a dedicated wave — remaining debts are bounded and can be addressed incrementally. A standalone hardening wave would delay value delivery without proportionate risk reduction.

- **Product evolution** is the right forcing function — building real features (even internal-facing) creates the operational pressure that exposes remaining architectural weaknesses. Features like multi-symbol monitoring, strategy performance tracking, or execution analytics would exercise the architecture under real usage patterns.

**Guiding principles for Phase 2:**

1. **Start with one feature, not a feature roadmap.** Pick the single most valuable capability that uses the existing pipeline and build it.
2. **Address debts as they become blocking.** When adding a new query operation, extend the generic UseCase pattern. When adding a new family, implement the cross-registration coherence test. Don't front-load debt resolution.
3. **Keep the expansion playbook honest.** Every new domain or family must follow the documented expansion playbook. If the playbook is wrong, fix it — don't work around it.
4. **Maintain the evidence discipline.** Continue the S110 pattern: validate, capture friction, refactor only what evidence justifies.

### Phase 2 Candidate Features (Ordered by Architectural Coverage)

| Feature | What It Exercises | New Code Required |
|---------|-------------------|-------------------|
| Multi-symbol live monitoring | Config activation for 2+ bindings, concurrent pipeline paths | Config only (architecture already supports it) |
| Strategy performance dashboard | Store read models, gateway query aggregation | New query endpoints, possibly new KV projections |
| Execution audit trail | Execute → store → gateway chain, historical fill queries | New projection + query surface |
| Alert on pipeline stall | Diagnostic surface consumption, health threshold logic | New gateway endpoint + polling logic |

---

## 4. What to Defer

| Item | When to Reconsider |
|------|-------------------|
| ClickHouse integration | When NATS KV read models are insufficient for analytical queries |
| Event schema formalization | When a second producer for the same event type is introduced |
| OpenTelemetry | When log-based debugging fails for a real operational issue |
| Full E2E test automation | When the pipeline runs regularly and manual validation becomes a bottleneck |
| MarketMonkey absorption | After Phase 1 operational proof and at least one Phase 2 feature delivery |
| Raccoon-CLI new analyzers | When existing 18 analyzers are insufficient for governance needs |
| Documentation consolidation | When doc staleness causes a concrete mistake |

---

## 5. Decision Framework for Future Waves

After Phase 2, use this framework to decide the next wave:

```
Is the pipeline running live and stable?
├── No  → Fix operational issues first
├── Yes → Are debts blocking feature delivery?
│         ├── Yes → Targeted hardening (not a wave — fix what blocks)
│         └── No  → Is there demand for new capabilities?
│                   ├── Yes → Build the capability using expansion playbooks
│                   └── No  → The architecture is serving its purpose. Stop building architecture.
```

**The goal is not to perfect the architecture. The goal is to make the architecture serve the product.**

---

## 6. Anti-Patterns to Avoid

1. **Hardening as procrastination.** If the pipeline runs and features can be built, additional hardening is avoidance of product risk, not technical necessity.

2. **Slice repetition for comfort.** A second vertical slice that exercises the same patterns with different data is busywork, not validation.

3. **Absorption before proof.** Bringing MarketMonkey code into an unproven pipeline creates two problems instead of one.

4. **Generic extraction without pain.** Don't extract abstractions because the code looks repetitive. Extract when the repetition causes a concrete bug or blocks a concrete feature.

5. **Documentation as progress.** Documents describe reality; they don't create it. Write docs when decisions are made. Don't make decisions in order to write docs.
