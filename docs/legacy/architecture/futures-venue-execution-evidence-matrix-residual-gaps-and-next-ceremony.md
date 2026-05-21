# Futures Venue Execution Proof Wave -- Evidence Matrix, Residual Gaps, and Next Ceremony (Post-Simplification)

> Wave: Futures Venue Execution Proof, Post-Simplification (Phase 47, S421--S426)
> Gate Stage: S426
> Date: 2026-03-23
> Companion: [futures-venue-execution-evidence-gate.md](futures-venue-execution-evidence-gate.md)

---

## Evidence Matrix

### Capability Evidence

| ID | Capability | Stage | Tests | Evidence Type | Grade |
|---|---|---|---|---|---|
| FV-C1 | Real Futures venue acceptance lifecycle | S422 | 19 | ValidTransition chain, BUY/SELL, multi-cycle, segment routing | **FULL** |
| FV-C2 | Real Futures fill record fidelity | S422 | 19 | avgPrice, executedQty, cumQuote, updateTime, Simulated=false | **FULL** |
| FV-C3 | Real Futures rejection lifecycle | S423 | 19 | ValidTransition chain, 6 error scenarios, terminality, QueryOrder | **FULL** |
| FV-C4 | Real Futures rejection event fidelity | S423 | 19 | AuditTrail construction, venue detail preservation, event field verification | **FULL** |
| FV-C5 | Real Futures partial fill lifecycle | S423 | 19 | Structural: ValidTransition chain, quantity monotonicity (4 ratios); no live observation | **STRUCTURAL** |
| FV-C6 | Lifecycle invariant fidelity under real Futures data | S422+S423 | 38 | Rejection terminality (no further transitions), fill monotonicity, correlation preservation | **FULL** |
| FV-C7 | Persistence consistency (KV/HTTP/ClickHouse) | S424 | 16 | Rejection detail extraction, composite status, KV round-trip, timestamp priority | **FULL** |
| FV-C8 | Read-path auditability and segment parity | S424 | 16 | 10/10 parity dimensions, propagation logic identical, partition key isolation | **FULL** |
| FV-C9 | Compose E2E with Futures on canonical surface | S425 | 10 | Full pipeline, ValidTransition, multi-cycle, dry-run safety, controls verified | **FULL** |
| FV-C10 | Segment isolation under dual-segment config | S422--S425 | 64 | Fail-closed routing, AllowedSources gate, Spot adapter untouched, partition isolation | **FULL** |

**Summary: 9/10 FULL, 1/10 STRUCTURAL.**

### Governing Question Evidence

| ID | Question | Stage | Evidence | Grade |
|---|---|---|---|---|
| FV-Q1 | Lifecycle transitions on real Futures acceptance/fill | S422 | 7 tests: dominant path, BUY, SELL, multi-cycle, router, API path, response type | **ANSWERED** |
| FV-Q2 | Fill record fidelity (price, qty, fee, timestamp) | S422 | 2 tests: avgPrice extraction, updateTime extraction; Simulated=false | **ANSWERED** |
| FV-Q3 | Rejection lifecycle (HTTP errors + venue statuses) | S423 | 3 tests: dominant path, HTTP 200 REJECTED, HTTP 200 EXPIRED | **ANSWERED** |
| FV-Q4 | Rejection event audit trail | S423 | 1 test with 6 sub-scenarios: margin, balance, LOT_SIZE, auth, rate limit, venue internal | **ANSWERED** |
| FV-Q5 | Partial fill live observation | S423 | Structural proof: ValidTransition chain for accepted->partially_filled->filled; testnet fills instantly (same as Spot) | **STRUCTURAL** |
| FV-Q6 | Quantity monotonicity for partial fills | S423 | 1 test with 4 sub-cases (25%, 50%, 75%, 99%) | **ANSWERED** |
| FV-Q7 | KV/HTTP/ClickHouse terminal state agreement | S424 | Composite status tests, rejection detail extraction, KV round-trip | **ANSWERED** |
| FV-Q8 | ClickHouse rejection writer transparency | S424 | Fee semantics audit trail, rejection detail survives storage | **ANSWERED** |
| FV-Q9 | Compose E2E pipeline (ingest -> read-path) | S425 | 10 integration tests + 16-phase smoke script | **ANSWERED** |
| FV-Q10 | Sustained multi-cycle behavior | S424+S425 | S425 test 9: 5 sequential orders, unique IDs, per-order correlation | **ANSWERED** |
| FV-Q11 | Correlation chain integrity (ingest to outcome) | S422 | 3 tests: chain preservation, intent fields, ClientOrderID determinism | **ANSWERED** |
| FV-Q12 | Post-200 reconciliation (correct API path) | S422 | 2 tests: QueryOrder recovers fill, uses /fapi/v1/order | **ANSWERED** |

**Summary: 11/12 ANSWERED, 1/12 STRUCTURAL.**

### Artifact Inventory

| Category | Count | Details |
|---|---|---|
| Test files (new) | 4 | s422 (1), s423 (1), s424 (1), s425 (1) |
| Total tests (new) | 64 | 19 + 19 + 16 + 10 (top-level functions; sub-tests raise effective count to 84+) |
| Architecture docs | 10 | Charter + capabilities/non-goals + 2 per execution stage |
| Stage reports | 5 | S421--S425 |
| Smoke scripts | 2 | smoke-e2e-unified-futures.sh (updated), smoke-futures-venue-live.sh |
| Config files created | 0 | Canonical surface reused |
| Compose overlays created | 0 | Canonical surface reused |
| Production code changes | 0 | Zero across all 4 execution stages |

### Evidence Strength by Block

| Block | Stage | Tests | Prod Changes | Evidence Strength |
|---|---|---|---|---|
| Charter | S421 | 0 | 0 | Design framework, surface contract |
| Connectivity and fill | S422 | 19 | 0 | Strong: ValidTransition chain, multi-cycle, segment routing |
| Rejection and partial fill | S423 | 19 | 0 | Strong (rejection), Structural (partial fill) |
| Read-path and parity | S424 | 16 | 0 | Strong: 10/10 parity, KV round-trip, composite status |
| Compose E2E | S425 | 10 | 0 | Strong: full pipeline, controls, multi-cycle |

---

## Residual Gaps

### Carried Forward (Unchanged from S420 Phase 46 Gate)

| Gap | Severity | Status | Rationale |
|---|---|---|---|
| RG-2: Partial fill live observation | Low | CARRIED | Market orders fill instantly on both Spot and Futures testnet; structural proof sufficient; adapter parsing validated |
| RG-3: Latest-only KV semantics | Low | BY DESIGN | Each partition key holds only most recent intent; ClickHouse retains full history via event streams |
| RG-4: Segment-scoped list queries | Low | DEFERRED | LifecycleList enumerates all keys; consumers can filter by Source field; not blocking for wave scope |
| RG-6: Rejection code in JSON, not column | Low | CARRIED | Queryable via JSONExtractString; promote if analytics demand |
| RG-7: No dedicated rejection endpoint | Low | CARRIED | General endpoint with filter sufficient |
| RG-8: Synthetic endurance (cycle-based) | Low | CARRIED | Compose smoke phases mitigate; S425 multi-cycle proof adds coverage |
| RG-9: No time-based drift detection | Low | CARRIED | Actor health tracker and compose phases mitigate |
| RG-10: No pagination on lifecycle list | Low | CARRIED | Bounded cardinality; add if >500 keys |
| RG-11: Lifecycle list eventually consistent | Low | CARRIED | <1s lag acceptable for diagnostics |

### Carried from Phase 45 (Reassessed)

| Gap | Severity | Status | Reassessment |
|---|---|---|---|
| RG-12: cumQuote as Futures fee proxy | Low | CARRIED | Confirmed in S422/S424: cumQuote is total notional, not commission; true commission requires `/fapi/v1/userTrades` |
| RG-13: Fee semantic divergence (Spot vs Futures) | Medium | CARRIED | S424 proves this is venue-specific, not architectural; consumers must interpret by source field; **elevated to Medium** as most senior residual gap for production readiness |
| RG-14: No parallel Spot+Futures live execution proof | Low | CARRIED | S425 proves structural coexistence; simultaneous live execution deferred |
| RG-15: Single symbol scope at compose level | Low | CARRIED | All evidence for BTCUSDT only; multi-symbol structurally supported |

### New This Wave

| Gap | Severity | Mitigation |
|---|---|---|
| RG-16: 97 untracked docs | Low | Deferred from S420; requires separate governance ceremony; no runtime impact |
| RG-17: Smoke script naming inconsistency | Low | All references canonical; naming is cosmetic |
| RG-18: Doc suitability not assessed | Low | Deferred from S420; content correctness verified, format standardization deferred |

### Gap Disposition Summary

| Severity | Count | Classification |
|---|---|---|
| High | 0 | -- |
| Medium | 1 | RG-13 (fee semantic divergence) |
| Low | 15 | 9 carried from prior gates, 3 carried from Phase 45, 3 new |

**No blocking gaps. 1 Medium gap (RG-13) is the highest-priority item for the next ceremony.**

---

## Segment Parity Matrix (Consolidated)

| Dimension | Spot Evidence | Futures Evidence (Phase 47) | Parity Status |
|---|---|---|---|
| Adapter implementation | BinanceSpotTestnetAdapter (S405) | BinanceFuturesTestnetAdapter (S422) | FULL |
| API endpoint | `/api/v3/order` | `/fapi/v1/order` | Different by design |
| Lifecycle transitions (ValidTransition) | Proven (S405) | Proven (S422) | FULL |
| Fill record fidelity (Simulated=false) | fills[] array (S405) | avgPrice + cumQuote (S422) | FULL (format-normalized) |
| Rejection lifecycle (6+ error scenarios) | Proven (S406) | Proven (S423, 6 scenarios) | FULL |
| Rejection event audit trail | Proven (S406) | Proven (S423) | FULL |
| Partial fill (structural) | Structural (S406) | Structural (S423) | SAME GAP (RG-2) |
| Quantity monotonicity | Proven (S406) | Proven (S423, 4 ratios) | FULL |
| Rejected state terminality | Proven (S406) | Proven (S423, exhaustive) | FULL |
| Read-path queryability (4 lifecycle states) | Proven (S407) | Proven (S424) | FULL |
| Rejection detail extraction | Proven (S407) | Proven (S424, all 6 scenarios) | FULL |
| Rejection metadata KV round-trip | Proven (S407) | Proven (S424) | FULL |
| Composite status derivation | Proven (S407) | Proven (S424, fill+rejection+mixed) | FULL |
| Correlation chain preservation | Proven (S405-S408) | Proven (S422-S425, all 4 states) | FULL |
| Partition key isolation | binances.*.* (S407) | binancef.*.* (S424) | FULL |
| Mixed-segment lifecycle aggregation | Proven (S407) | Proven (S424) | FULL |
| Compose E2E pipeline (tests + smoke) | 8 tests + 16 phases (S408) | 10 tests + 16 phases (S425) | FULL |
| Segment isolation (fail-closed routing) | Proven (S408) | Proven (S425) | FULL |
| DryRunSubmitter safety | Wraps SegmentRouter (S408) | Wraps SegmentRouter (S425) | FULL |
| AllowedSources gate | binances permitted (S408) | binancef permitted (S425) | FULL |

**Parity verdict: FULL PARITY on 19/20 dimensions. 1 SAME GAP (RG-2: partial fill live observation).**

Venue-format differences (fills[] vs avgPrice, commission vs cumQuote, transactTime vs updateTime) are API-level differences correctly normalized by the respective adapters.

---

## Risk Register Closure

| Risk (from S421 charter, Section 7) | Outcome |
|---|---|
| G-4 fee divergence affects fill fidelity | MANAGED: S424 confirms cumQuote is venue-specific, not architectural; audit trail preserved |
| G-1 no parallel dual-segment live proof | MANAGED: structural coexistence proven in S425; not elevated |
| RG-2 partial fill elevated for Futures | MANAGED: structural proof complete (S423); same disposition as Spot |

All residual gaps from S421 charter Section 7 addressed. Zero escalations.

---

## Controls Verified Across Wave

| Control | Mechanism | Evidence Stage |
|---|---|---|
| Dry-run safety | DryRunSubmitter wraps entire SegmentRouter | S425 |
| Kill switch | EXECUTION_CONTROL KV gate | Inherited (S319, S380) |
| Staleness guard | 120s max age | Inherited (S317) |
| Source guard | AllowedSources in VenueAdapterActor | S425 |
| NATS consumer filter | Segment-filtered subscriptions | Inherited (S401) |
| Fail-closed routing | SegmentRouter rejects unknown sources | S422, S425 |
| Request signing | HMAC-SHA256 per Binance API contract | S422 |
| Idempotency | Deterministic ClientOrderID | S422 |

---

## Wave Predecessor Verification

| Dependency | Source | Status in Phase 47 |
|---|---|---|
| Seven-state lifecycle model | S383 | Futures applies same model; ValidTransition chain proven |
| Write-path per execution mode | S385 | venue_live + dry_run both proven for Futures |
| Rejection event infrastructure | S386, S411 | Reused without changes for Futures |
| Persistence read-path | S387, S413 | Segment-agnostic; works for Futures transparently |
| Segment routing | S400-S401 | SegmentRouter dispatches Futures correctly |
| Config enablement | S393, S399 | Futures segment enabled via canonical configs |
| Unified runtime | S402-S403 | Both segments coexist on single binary |
| Spot execution baseline | S405-S409 | Intact; zero regressions |
| Production hardening | S411-S414 | Endurance, rejection persistence, queryability all hold |
| Runtime simplification | S416-S420 (Phase 46) | Canonical surface enforced; zero per-segment deviations |

All predecessor capabilities verified intact.

---

## Next Ceremony Recommendation

### Wave Closure

The Futures Venue Execution Proof Wave (Phase 47) is **CLOSED** with **PASS -- FULL DELIVERY**.

Combined with the Phase 45 proof (PASS -- SUBSTANTIAL), Phase 46 simplification (PASS -- FULL), and this Phase 47 re-proof (PASS -- FULL), the Futures segment has been proven three times with progressively stronger evidence on a progressively cleaner surface.

### Strategic Position

The Foundry has achieved:
- **11 consecutive passing gates** (S375 through S426).
- **Dual-segment lifecycle** (Spot + Futures) proven on the unified runtime.
- **Canonical surface** (3 configs, 3 compose) handling all operational modes.
- **Full segment parity** (19/20 dimensions, 1 shared gap).
- **Zero production code changes** across Phase 47 -- the architecture supports Futures natively.

### Recommended Next Direction

**Production hardening ceremony** (short wave, focused scope):

1. **Fee normalization** -- Close RG-13 (most senior residual gap). Define canonical fee model that accounts for Spot commission and Futures cumQuote differences. Requires `/fapi/v1/userTrades` for true Futures commission.

2. **Mainnet readiness audit** -- Assess what separates the current testnet-proven system from mainnet deployment. Produce explicit readiness checklist with blocking/non-blocking classification.

3. **Per-segment health check** -- Close G-5 (carried since S421). Enable segment-specific health monitoring.

4. **KV history strategy assessment** -- Evaluate whether RG-3 (latest-only KV) needs resolution before production, or if ClickHouse historical views are sufficient.

**Why this direction:** It addresses the highest-value residual gaps without opening new architectural surfaces. It builds directly on the proven lifecycle and prepares the Foundry for the eventual mainnet decision.

### Candidates explicitly deferred:

| Candidate | Why deferred |
|-----------|-------------|
| OMS expansion (limit orders, cancel) | High complexity; requires new lifecycle states; not blocked by current gaps |
| Multi-exchange | Validates architecture generality but does not close residual gaps |
| Multi-symbol proof | Low risk; structurally supported; can be proven incrementally |
| Observability maturity | Important but not blocking the critical path to production |

### Items NOT authorized by this gate:

- Opening mainnet execution (NG-1 remains enforced).
- Multi-exchange expansion (NG-2 remains enforced).
- Full OMS redesign (NG-6 remains enforced).
- Re-opening config/compose/taxonomy surfaces.
- Portfolio or risk integration (NG-11 remains enforced).
- Committing 97 untracked docs (requires separate governance ceremony).
