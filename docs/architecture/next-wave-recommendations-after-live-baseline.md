# Next Wave Recommendations After Live Baseline

> Evidence-based recommendation for the Foundry's next phase, grounded in what the live pipeline wave (S113–S117) actually proved and what the operational baseline enables.

---

## 1. The Question

After S113–S117, the Foundry faces four possible directions:

1. **More operational hardening** — soak tests, failure injection, endurance validation
2. **New capability on the proven mesh** — multi-symbol, strategy performance, execution analytics
3. **Disciplined absorption of external capabilities** — MarketMonkey patterns, new venues, new sources
4. **Broader live proof** — multi-symbol sustained operation, additional venue types

The right answer depends on what the live wave proved, what it left open, and where the highest value lies.

---

## 2. What the Evidence Says

### 2.1 The Architecture Works Under Controlled Conditions

- Full event chain (observation → fill) runs with real market data
- All 7 runtimes start, communicate, and shut down correctly
- Safety gates protect execution under paper trading
- Diagnostic surfaces accurately report pipeline health
- Config-driven activation works without restart
- 950 architectural rules enforced mechanically

**Implication:** There is no structural or operational reason to delay forward progress under controlled conditions.

### 2.2 Endurance and Resilience Are Unproven

- No sustained-load test (hours/days)
- No failure injection (NATS disconnect, container restart)
- No memory/goroutine leak detection
- Cold-start behavior undocumented

**Implication:** Operations that require sustained reliability (production trading, continuous monitoring) cannot be assumed safe. But operations that tolerate controlled restarts (development, feature iteration, internal demos) are safe.

### 2.3 No Deferred-Item Trigger Has Fired

All 7 deferred items from S116 and all prior deferrals remain un-triggered. This validates the deferral decisions — the system has not suffered from any of the deferred items.

**Implication:** A dedicated hardening wave to resolve deferred items is not justified by evidence. Address them when their triggers fire.

### 2.4 Documentation Volume Is Approaching Overhead

~50 architecture documents and ~22 stage reports exist. Discovery is becoming harder than authorship. Operators start at the S117 baseline doc, but researchers or auditors face significant volume.

**Implication:** Future stages should prefer updating existing documents over creating new ones. The next wave should produce fewer documents, not more.

### 2.5 Zero Features Delivered Since S96

22 stages (S96–S117) of architecture, consolidation, governance, hardening, slicing, live activation, and baseline consolidation — with zero user-facing feature delivery.

**Implication:** The architectural investment is mature. Continuing to invest in architecture without delivering capability will produce diminishing returns. The architecture exists to serve the product, not the other way around.

---

## 3. Recommendation

### Direction: New Capability on the Proven Mesh

The evidence supports **controlled capability delivery** as the next wave. Not more hardening. Not absorption. Not broader live proof.

Here's why the alternatives don't win:

**More operational hardening** — No deferred-item trigger has fired. Soak testing requires infrastructure that doesn't exist and would delay feature delivery further. The system works under controlled conditions; hardening for conditions we haven't encountered yet is speculative investment.

**Broader live proof (multi-symbol sustained)** — The architecture already supports multi-symbol (validated via `make smoke-multi`). Sustained multi-symbol requires soak infrastructure. The delta between "smoke-tested" and "soak-tested" is not architectural — it's operational infrastructure. Build it when the feature requires it.

**Disciplined absorption (MarketMonkey)** — Absorption is integration work that should happen after the Foundry has proven it can deliver capability on its own patterns. Absorbing external code into an architecture that has never shipped a feature creates coupling before validation.

**Controlled capability delivery wins because:**
- The architecture is proven and ready
- Features create the operational pressure that exposes remaining weaknesses
- The next architectural insights will come from building on the mesh, not from examining it further
- 22 stages of architecture work needs to translate into value delivery

### What "Controlled Capability" Means

Not a feature roadmap. Not a sprint plan. One capability at a time, using the proven patterns, with the evidence discipline established in S110/S115:

1. **Pick one capability** that uses the existing pipeline
2. **Build it** using expansion playbooks
3. **Validate it** under live conditions
4. **Capture friction** if playbooks or patterns are inadequate
5. **Refactor only what evidence justifies**

### Candidate Capabilities (Ordered by Risk)

| Capability | What It Exercises | New Code | Risk |
|------------|-------------------|----------|------|
| **Multi-symbol live monitoring** | Config activation for 2+ bindings, concurrent pipeline paths | Config only — architecture already supports it | **Low** — validated by smoke-multi |
| **Candle history query enrichment** | Store read models, gateway query aggregation, time-range queries | New query endpoints, ClickHouse integration possible | **Low-Medium** — extends proven patterns |
| **Strategy performance tracking** | New KV projection, new query surface, execution → performance chain | New projection actor + query endpoint | **Medium** — follows projection playbook |
| **Execution audit trail** | Historical fill queries, execution event replay | New projection + query surface | **Medium** — follows projection playbook |
| **New signal family** (e.g., MACD, Bollinger) | Derive expansion, new sampler, signal → decision → strategy chain | New domain types + derive processor | **Medium** — exercises expansion playbook |
| **Live venue adapter (testnet)** | Execute actor with real venue API, submit timeout under latency | Venue adapter implementation, credential management | **High** — unproven execution path |

### Recommended First Capability

**Multi-symbol live monitoring** is the lowest-risk, highest-signal choice:
- Exercises config activation with 2+ concurrent bindings under sustained operation
- Creates natural pressure for soak testing (operator will want to run it for hours)
- Requires zero new code — only configuration changes
- Validates that the architecture scales horizontally as designed
- Provides immediate operational value (real-time monitoring of multiple markets)

If multi-symbol works smoothly, the next capability should exercise a new code path (strategy performance tracking or a new signal family) to validate the expansion playbooks under real delivery conditions.

---

## 4. What to Do Before Starting

Two small preparatory items have earned their investment:

| Item | Cost | Why Now |
|------|------|---------|
| Inject correlation ID into slog context (~15 files) | 1 day | Multi-symbol debugging across 4+ runtimes will be painful without it. Evidence from S114/S115. |
| Document cold-start behavior (1 doc section) | 2 hours | Operators activating multi-symbol will encounter cold-start window confusion. |

Everything else should be addressed only if the capability delivery surfaces the need.

---

## 5. What to Defer

| Item | When to Reconsider |
|------|-------------------|
| Soak test infrastructure | When multi-symbol operation runs long enough to care about stability |
| ClickHouse write path | When KV read models are insufficient for analytical queries |
| OpenTelemetry / distributed tracing | When correlation ID in slog is insufficient for debugging |
| Event schema formalization | When a second producer for the same event type exists |
| MarketMonkey absorption | After at least one capability is delivered on Foundry's own patterns |
| Generic supervisor framework | When supervisor pattern causes a concrete bug |
| Use-case pattern unification | When a new domain is added and the developer is confused |
| Script hardening | When CI/CD pipeline is set up |
| Automated RecordError lint | When a RecordError regression occurs |

---

## 6. Decision Framework (Updated)

```
Is the pipeline running live and stable?
├── No  → Fix operational issues first
├── Yes → Have we delivered any capability on the architecture?
│         ├── No  → Deliver one capability (we are HERE)
│         └── Yes → Did delivery expose architectural pain?
│                   ├── Yes → Targeted fix (not a wave)
│                   └── No  → Deliver next capability
```

---

## 7. Anti-Patterns to Avoid in the Next Wave

1. **Architecture as procrastination.** If the mesh works and playbooks exist, build the feature. Don't find reasons to refine the mesh first.

2. **Absorption before delivery.** MarketMonkey patterns are easier to reconcile after Foundry has proven it can ship. Don't integrate external code to avoid the harder problem of building your own features.

3. **Hardening for hypothetical load.** Soak testing for a system that hasn't shipped a feature is premature optimization. Build the feature, then test its endurance.

4. **Documentation as deliverable.** The next wave should produce code, not documents. Update existing docs when the system changes. Don't create new architecture documents unless a genuinely new pattern emerges.

5. **Feature roadmap before first feature.** Pick one thing. Build it. Learn from it. Then pick the next thing. Planning 5 features before delivering 1 is inventory, not progress.

---

## 8. Success Criteria for the Next Wave

The next wave succeeds if:

- [ ] At least one new capability is delivered and operational
- [ ] The capability was built using expansion playbooks (or playbooks were updated where inadequate)
- [ ] Friction was captured honestly (not hidden or rationalized)
- [ ] Deferred items were resolved only if their triggers fired during delivery
- [ ] The architecture served the capability, not the other way around
- [ ] Fewer new architecture documents were created than capabilities delivered
