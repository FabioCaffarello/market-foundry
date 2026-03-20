# Live Baseline Gains, Trade-offs, and Open Debts

> Honest accounting of what the live pipeline wave (S113–S117) delivered, what it cost, and what remains unresolved. Third in the series after `structural-gains-tradeoffs-and-open-debts.md` (S96–S99) and `platform-gains-tradeoffs-and-open-debts.md` (S101–S105).

---

## Gains: What Actually Improved

### LG-1. Operational Proof of Architecture

**Before (S112):** The architecture was structurally proven by unit tests, static analysis, and 950 raccoon-cli rules. No live pipeline run had been performed. The statement "it works" was based on code review, not observation.

**After (S117):** The architecture has run end-to-end with real NATS, real Binance WebSocket market data, and real event flow across 7 services. The statement "it works" is now based on observation.

**Value:** This is the single most important gain of the entire wave. Every decision after this point can reference real operational evidence, not structural inference.

### LG-2. Execution Safety Hardening

**Before (S112):** Execute actor safety logic (kill switch, staleness guard, submit timeout) had zero unit tests. S112 called this the only P0 blocker.

**After (S113):** SafetyGate extracted as testable unit. 22 new tests covering all gate combinations, boundary conditions, and degraded modes. Gate evaluation order proven. Fail-open/fail-closed semantics verified.

**Value:** The most consequential code in the system (order placement control) is now the most tested. This removed the only blocker to live operation.

### LG-3. Quality Gate Signal-to-Noise

**Before (S115):** drift-detect produced ~265 warnings per run — ~260 were false positives from NATS "consumer" pattern strings matching old service names.

**After (S116):** drift-detect produces 5 warnings (all legitimate). Quality gate output is trustworthy.

**Value:** Developers can now trust the quality gate. A warning means something. Before R1, the gate was technically correct but practically useless for drift detection.

### LG-4. Explicit Operational Baseline

**Before (S116):** Operational knowledge was distributed across S114/S115/S116 reports. Retaking the system after a break required reading 3+ documents to reconstruct the operational picture.

**After (S117):** Single-document baseline (`minimal-operational-baseline.md`) covers topology, procedures, lifecycle, dependencies, safety gates, and the stable/experimental boundary. 10 named invariants and 11-step runbook in companion document.

**Value:** Reduces retake cost from "read and reconstruct" to "read and execute." This is the difference between 30 minutes and 5 minutes to restart the system after a break.

### LG-5. Bug Discovery and Fix Under Real Conditions

**S115 found 3 bugs** that all testing phases (S96–S112) missed:

| Bug | Category | Why Tests Missed It |
|-----|----------|-------------------|
| Stream heuristic scanned top-to-bottom, picked KV bucket over correct stream | Tooling | Only manifests with multiple candidate matches in same file region |
| Gateway imported from `interfaces/http/webserver` (layer violation) | Architecture | Import compiled successfully; only raccoon-cli detected it, but only after running against full compose topology |
| Test fixture missing 3 streams and 4 durables | Test infrastructure | Fixture was correct when written; pipeline expanded since then |

**Value:** These bugs validate the S112 decision to run live before expanding. Structural testing alone would not have caught them. The pattern — "run, observe, fix" — is proven as a necessary complement to static analysis.

---

## Trade-offs: What This Wave Cost

### LT-1. Elapsed Time Without Feature Delivery

S113–S117 span 5 stages of operational work with zero user-facing features delivered. The entire wave is infrastructure investment.

**Assessment:** Justified. The S112 review explicitly identified "operationally unproven" as the critical gap. Closing it was the right priority. But the investment is real — 5 stages of architecture work must now translate into capability delivery to justify the cost.

### LT-2. Documentation Volume (Continued)

S113–S117 produced 6 architecture documents and 5 stage reports. Total project documentation now stands at ~50 architecture documents and ~22 stage reports.

**Assessment:** The volume is proportionate to the decisions made, but the maintenance burden is cumulative. Each document is an assertion about the system state that can become stale. The two-tier approach (baseline docs consolidate, stage reports are immutable history) helps, but the total count is approaching the point where discovery becomes a problem.

**Mitigation:** S117's baseline consolidation partially addresses this — operators start there, not in the archive. Future stages should update existing docs rather than create new ones where possible.

### LT-3. Minimal Scope of Operational Proof

The live run validated single-symbol, paper-venue, short-duration operation. This is the minimum possible operational proof. The gap between "works in a parking lot" and "works on a highway" is significant.

**Assessment:** This is the correct trade-off for the current stage. A broader operational proof (multi-symbol, sustained, failure injection) requires infrastructure that doesn't exist and would delay feature delivery further. The minimal proof is sufficient to unlock the next wave.

### LT-4. Deferred Items Accumulating

Combined deferred items across all waves:

| Source | Deferred Items | Still Valid |
|--------|---------------|-------------|
| S96–S99 (structural) | 5 (D1–D5) | 3 open (D1 composition root tests, D2 cross-registration, D3 correlation ID) |
| S101–S105 (platform) | 5 (PD-1–PD-5) | 3 open (PD-1 = D1, PD-3 = D3, PD-5 raccoon maintenance) |
| S107–S111 (slice) | 8 (deferred) | 6 open (D1–D5 from refactors-deferred, plus D6–D8 from S112) |
| S113–S117 (live) | 7 (D1–D7) | 7 open |

**De-duplicated total: ~12 distinct deferred items.** Most are tracked across multiple documents (e.g., "correlation ID" appears in S102, S103, S112, and S116 deferrals). None have had their triggers fire.

**Assessment:** The count is manageable but the tracking is fragmented. Multiple documents reference the same deferred item with different IDs.

---

## Open Debts: What Remains Unresolved

### LD-1. Cross-Runtime Observability

**Status:** Open (carried from S102, S103, S112)
**Severity:** Medium (increasing with operational complexity)

Correlation IDs exist in domain events but are not injected into slog attributes. Log-based debugging across 3+ runtimes requires timestamp correlation. No distributed tracing, no per-event metrics, no dashboards.

**Why it matters now:** During S114/S115, debugging event flow required reading logs from 4+ services simultaneously. This was tolerable for single-symbol validation. It will become a bottleneck for multi-symbol operation or incident investigation.

**Smallest useful fix:** Inject correlation ID into slog context at consumer entry point (~15 files). Does not require OpenTelemetry or collector infrastructure.

### LD-2. Endurance / Soak Testing

**Status:** Open (carried from S116 D4)
**Severity:** Low (single-symbol paper) / Medium (if moving to multi-symbol or live)

No sustained-load test exists. Potential issues (goroutine leaks, NATS backlog growth, memory pressure) only manifest under continuous operation.

**Why it matters:** The live run was short-duration. Stability under hours/days of continuous operation is unproven.

**Trigger:** Moving to multi-symbol or live trading.

### LD-3. Failure Recovery Paths

**Status:** Open (carried from S112)
**Severity:** Low-Medium

JetStream consumer restart, NATS reconnection, and actor crash recovery paths are not exercised. The system may or may not recover correctly from infrastructure failures.

**Why it matters:** Real operation will encounter NATS disconnects, container restarts, and network partitions.

**Trigger:** Chaos/resilience testing phase, or first unplanned failure in sustained operation.

### LD-4. Cold-Start Behavior

**Status:** Open (carried from S112)
**Severity:** Low

RSI evaluator needs historical candles before producing meaningful signals. Behavior during the first 60-120s after activation is undocumented and untested.

**Why it matters:** Operators may be confused by empty responses during cold start. Staleness guard protects execution, but the user experience is unclear.

**Trigger:** Moving to live venue (where cold-start decisions have real financial impact).

### LD-5. Deferred Item Tracking Fragmentation

**Status:** New
**Severity:** Low

The same deferred items appear in multiple documents with different IDs (e.g., "correlation ID" is tracked in at least 4 documents). No single source of truth for what is deferred and why.

**Why it matters:** When evaluating whether to invest in a deferred item, the evaluator must check multiple documents to understand the full history.

**Mitigation:** Not worth a dedicated fix. The baseline docs (S117) serve as the current canonical list. Future stages should reference the baseline rather than creating new deferred-item lists.

---

## Refactors That Do NOT Warrant the Cost Now

| Refactor | Why Not Now | Trigger |
|----------|------------|---------|
| OpenTelemetry / distributed tracing | Correlation ID in slog is sufficient; no collector infra | Log-based debugging fails for real incident |
| Soak test infrastructure | Requires dedicated environment; single-symbol paper doesn't justify | Multi-symbol or live trading |
| Composition root smoke tests | Live run proved all roots work; no regression since | Wiring bug recurrence |
| Use-case pattern unification | No bugs from inconsistency; 20+ files touched for cosmetic gain | New domain where developer is confused |
| Generic supervisor framework | Each supervisor has domain-specific lifecycle | Supervisor pattern causes concrete bug |
| Event schema formalization | Single producer per event type; no schema drift | Multi-team or multi-language consumers |
| ClickHouse write path activation | KV read models sufficient for current queries | Analytical query requirements |
| Config parameterization | Single deployment environment; duplication hasn't caused a bug | Second environment |
| Automated RecordError lint | S103 fixes cover all actors; pattern followed by reference | RecordError regression |
| Script hardening | Scripts work for manual use; not in CI | CI/CD pipeline setup |

---

## Wave Summary: S113–S117 in Numbers

| Metric | Count |
|--------|-------|
| Stages completed | 5 |
| Code files modified | ~30 |
| New tests added | 22 (S113) |
| Bugs found in live operation | 3 (all fixed) |
| Domain logic bugs | 0 |
| Bounded refactors applied | 4 (S116) |
| Items evaluated and deferred | 7 (S116) |
| Architecture documents created | 6 |
| Stage reports created | 5 |
| Deferred item triggers fired | 0 of 7 |
| P0 blockers resolved | 1 (execute actor tests) |
| Features delivered | 0 |

---

## Related Documents

- `post-vertical-slice-01-architectural-readiness-review.md` — S112 review (predecessor)
- `post-live-architectural-and-refactoring-readiness-review.md` — S118 main review
- `next-wave-recommendations-after-live-baseline.md` — S118 recommendations
- `minimal-operational-baseline.md` — S117 baseline
- `structural-gains-tradeoffs-and-open-debts.md` — S96–S99 equivalent
- `platform-gains-tradeoffs-and-open-debts.md` — S101–S105 equivalent
