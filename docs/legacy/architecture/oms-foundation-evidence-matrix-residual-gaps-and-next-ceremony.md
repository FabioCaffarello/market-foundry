# OMS Foundation — Evidence Matrix, Residual Gaps & Next Ceremony

> Companion document to the [Evidence Gate](oms-foundation-evidence-gate.md).
> Contains the detailed evidence matrix, residual gap analysis, and next ceremony recommendation.

---

## 1. Evidence Matrix

### 1.1 Charter & Scope Freeze Evidence (S382)

| Item | Evidence | Status |
|------|----------|--------|
| Wave scope frozen with 17 capabilities, 9 questions, 14 non-goals | Charter document produced | **VERIFIED** |
| 9 governing questions defined with target stages | Capabilities doc enumerates OMS-Q1 through OMS-Q9 | **VERIFIED** |
| Risk register with 5 risks and mitigations | Charter report section | **VERIFIED** |
| Gap closure targets defined (G1 price realism, G10 partial fills) | Charter enumerates pre-existing gaps with closure assignments | **VERIFIED** |
| Stage execution order defined (S383→S384→S385→S386→S387→S388) | Charter report | **VERIFIED** |
| 14 non-goals explicitly frozen | NG-1 through NG-14 documented with rationale | **VERIFIED** |

### 1.2 Canonical Order Model Evidence (S383)

| Item | Evidence | Status |
|------|----------|--------|
| ExecutionIntent fully documented (16 fields) | Architecture doc: canonical order model | **VERIFIED** |
| Seven-state lifecycle mapped with three tiers | Architecture doc: Initial → In-Flight → Terminal | **VERIFIED** |
| 10 valid transitions cataloged | `ValidTransition()` logic + architecture doc | **VERIFIED** |
| 39 invalid transitions cataloged | Architecture doc: exhaustive 7×7 matrix | **VERIFIED** |
| Three execution mode paths identified | dry_run, paper, venue_live mode-specific pipelines | **VERIFIED** |
| 49 invariants cataloged across 8 categories | ST, TERM, FR, IFC, QM, SM, SAFE, CORR | **VERIFIED** |
| G1 resolution path defined (NATS KV price lookup) | Architecture doc: PriceSource interface design | **VERIFIED** |
| Only 8/49 invariants covered pre-S384 (16%) | Gap analysis: 41 invariants lacking domain tests | **VERIFIED** |

### 1.3 Lifecycle Invariant Coverage & Price Realism Evidence (S384)

| Item | Evidence | Status |
|------|----------|--------|
| 49/49 transition pairs tested (10 valid, 39 invalid) | `s384_lifecycle_invariants_test.go`: exhaustive 7×7 matrix | **AUTOMATED** |
| Terminal state absorption (3 states, no outgoing) | `IsTerminal()` tests + transition matrix | **AUTOMATED** |
| Fill record invariants (FR-1 through FR-9) | Fill presence, shape, Simulated flag, timestamp ordering | **AUTOMATED** |
| Intent-fill consistency (7 invariants) | Quantity sum, side/symbol/source preservation | **AUTOMATED** |
| Quantity monotonicity (QM-1 through QM-3) | Forward-only, bounds, terminal equality | **AUTOMATED** |
| Status monotonicity (tier ordering) | No backward transitions, no self-transitions | **AUTOMATED** |
| Safety/validation completeness (7 required fields) | Each required field tested when zeroed | **AUTOMATED** |
| Correlation chain stability (4 invariants) | CorrelationID, CausationID, PartitionKey, DeduplicationKey | **AUTOMATED** |
| PriceSource interface (`ports/price.go`) | Contract: best-effort, fallback to "0", concurrent-safe | **VERIFIED** |
| DryRunSubmitter + PaperVenueAdapter price integration | `s384_price_realism_test.go`: 10 tests | **AUTOMATED** |
| Backward compatibility (no PriceSource = same behavior) | Test: code without `WithPriceSource()` unchanged | **AUTOMATED** |
| Invariant coverage 16% → 100% | 41 new invariant tests; all 8 categories covered | **AUTOMATED** |

### 1.4 Write-Path Integration Evidence (S385)

| Item | Evidence | Status |
|------|----------|--------|
| dry_run mode: buy/sell/none transitions | `s385_write_path_by_mode_test.go` | **AUTOMATED** |
| paper mode: buy/sell/none transitions | `s385_write_path_by_mode_test.go` | **AUTOMATED** |
| venue_live mode: buy/sell transitions | `s385_write_path_by_mode_test.go` with httptest | **AUTOMATED** |
| venue_live rejection path (submitted→rejected) | `s385_write_path_by_mode_test.go` | **AUTOMATED** |
| venue_live partial fills | `s385_write_path_by_mode_test.go` | **AUTOMATED** |
| Cross-mode Simulated flag consistency | 7 cross-mode invariant tests | **AUTOMATED** |
| VenueOrderID prefix conventions (dryrun-, paper-, numeric) | Test assertions per mode | **AUTOMATED** |
| Correlation chain preserved across all modes | `TestCorrelationChainPreservation` variants | **AUTOMATED** |
| No-action intent semantics (SideNone) | Venue never contacted; accepted without fill | **AUTOMATED** |
| Terminal state absorption cross-mode | No transition out of filled/rejected | **AUTOMATED** |
| FilledQuantity = Quantity on full fill | Quantity equality assertion on terminal fill | **AUTOMATED** |
| All transitions validated against `ValidTransition()` | State machine alignment per assertion | **AUTOMATED** |

### 1.5 Rejection Event Path Evidence (S386)

| Item | Evidence | Status |
|------|----------|--------|
| `VenueOrderRejectedEvent` domain event | `events.go`: rejection code, reason, venue details | **VERIFIED** |
| Event interface compliance | `s386_rejection_event_test.go`: implements events.Event | **AUTOMATED** |
| Correlation chain preserved in rejection | Test: CorrelationID + CausationID survive | **AUTOMATED** |
| Lifecycle alignment (submitted→rejected valid, rejected terminal) | Test: `ValidTransition` + `IsTerminal` assertions | **AUTOMATED** |
| `EXECUTION_REJECTION_EVENTS` stream created at startup | `publisher.go`: EnsureStream in StartPublisher | **VERIFIED** |
| Subject pattern: `execution.rejection.venue_market_order.{source}.{symbol}.{timeframe}` | Registry code + architecture doc | **VERIFIED** |
| `Publisher.PublishRejection()` with dedup key | Publisher code with test coverage | **AUTOMATED** |
| `VenueAdapterActor.publishRejection()` on all failure paths | Actor code: non-retryable + exhausted retryable | **VERIFIED** |
| Two consumer specs (store + writer) | Registry: `StoreVenueMarketOrderRejectionConsumer`, `WriterVenueMarketOrderRejectionConsumer` | **VERIFIED** |
| Gate-blocked intents do NOT produce rejection events (by design) | Architecture doc: gate blocks precede venue submission | **VERIFIED** |
| 19 tests across domain/actor/registry layers | `s386_rejection_event_test.go` (7) + `s386_rejection_event_path_test.go` (5) + `s386_rejection_registry_test.go` (7) | **AUTOMATED** |

### 1.6 Lifecycle Persistence & PriceSource Wiring Evidence (S387)

| Item | Evidence | Status |
|------|----------|--------|
| `CandleKVPriceSource` reads CANDLE_LATEST KV | `price_source.go`: Close field extraction | **VERIFIED** |
| Price fallback semantics (nil store, error, cold start → "0") | `s387_price_source_test.go`: 4–5 tests | **AUTOMATED** |
| Production wiring in `cmd/execute/run.go` | Line 47: NATS KV connection → PriceSource → DryRunSubmitter | **VERIFIED** |
| `RejectionProjectionActor` materializes to KV | `rejection_projection_actor.go`: monotonicity guard, stats | **VERIFIED** |
| `EXECUTION_VENUE_REJECTION_LATEST` KV bucket | Store supervisor pipeline declaration | **VERIFIED** |
| Composite status query includes rejection | `ExecutionStatusReply.Rejection` field (additive, nil when absent) | **VERIFIED** |
| `DeriveEffectivePropagation()` with 3-way comparison | `s387_lifecycle_persistence_test.go`: 7 derivation cases | **AUTOMATED** |
| Read-path returns: Intent, Result, Rejection, Gate, Propagation | Query responder code + test assertions | **AUTOMATED** |
| PriceSource wiring validation | `s387_lifecycle_persistence_test.go`: 3 wiring tests | **AUTOMATED** |
| Backward compatible (Rejection field optional) | Nil default; no breaking change to callers | **VERIFIED** |
| All 5 binaries compile cleanly | Build verification | **VERIFIED** |
| 16 new tests (12 persistence + 4 price source) | Two test files, all pass | **AUTOMATED** |

### 1.7 Regression Evidence

| Scope | Packages | Result |
|-------|----------|--------|
| All Go workspace modules | ~50 packages | **ALL PASS** |
| New tests added by wave | ~100 tests across 8 files | **ALL PASS** |
| Pre-existing tests | All prior packages | **ZERO REGRESSIONS** |
| Binary compilation | 5 binaries (execute, store, gateway, derive, writer) | **ALL COMPILE** |

---

## 2. Residual Gaps

### 2.1 Gaps Inherited and Closed

| ID | Gap (from prior waves) | Status | Closed by |
|----|------------------------|--------|-----------|
| G1 | Dry-run fills use Price="0" | **CLOSED** | S384: PriceSource interface; S387: CandleKVPriceSource wired in production |
| G10 | Single fill shape (no partials) | **CLOSED** | S385: partial fills demonstrated in venue_live mode |

### 2.2 Gaps Identified Within Wave and Closed

| ID | Gap | Status | Closed by |
|----|-----|--------|-----------|
| S385-G1 | Rejections return Problem, not event (observability gap) | **CLOSED** | S386: VenueOrderRejectedEvent published on all failure paths |
| S386-G1 | No rejection KV projection | **CLOSED** | S387: RejectionProjectionActor materializes to EXECUTION_VENUE_REJECTION_LATEST |
| S386-G2 | Read-path does not include rejection state | **CLOSED** | S387: ExecutionStatusReply gains Rejection field; DeriveEffectivePropagation considers rejection |

### 2.3 Acknowledged Residual Gaps (Not Blocking)

| ID | Gap | Severity | Why Not Blocking | Recommendation |
|----|-----|----------|-----------------|----------------|
| RG-1 | ClickHouse rejection writer not wired | LOW | Consumer spec and stream exist; JetStream provides 72h retention as interim persistence; writer actor is mechanical wiring | Wire early in next wave as housekeeping |
| RG-2 | Domain-level quantity enforcement deferred | LOW | `FilledQuantity ≤ Quantity` proven by tests but not enforced in `Validate()`; behavioral invariant is producer convention | Evaluate adding to `Validate()` when defensive depth required |
| RG-3 | Fee realism in dry-run/paper (Fee="0") | LOW | Price realism addressed; fee realism is informational, not safety-critical; no downstream consumer depends on fee accuracy | Address when P&L computation requires accurate fees |
| RG-4 | Status `sent` never exercised end-to-end | LOW | Valid in state machine but no adapter produces it (no async acknowledgment protocol); reserved for future expansion | Exercise when async WebSocket fills implemented |
| RG-5 | Status `cancelled` via adapter not tested E2E | MEDIUM | `mapBinanceStatus()` maps it correctly; no cancel-order API call implemented; state machine handles it | Address when order cancellation capability added |
| RG-6 | No RejectionProjectionActor direct unit tests | LOW | Tested indirectly through integration; monotonicity guard and stats tracking verified by code inspection | Add unit tests as hardening |
| RG-7 | No OMS-specific compose smoke script | LOW | E2E proof composed from prior wave smoke + S385–S387 integration tests; no gap in coverage, only in automation packaging | Package as `make smoke-oms-lifecycle` in future stage |
| RG-8 | OMS-Q9 sustained stability not re-proven with OMS path | LOW | Prior wave's 5+ minute stability intact; OMS additions are additive with zero regressions; re-proof is low-value | Include in next wave's E2E compose smoke |
| RG-9 | `knownExecutionFamilies` does not list rejection family | LOW | Rejection family exists at adapter/domain layer; settings schema validates execution families, not rejection streams | Add for consistency when settings schema evolves |
| RG-10 | Rejection projection best-effort (store unavailability → incomplete query) | LOW | Startup warning logged; query returns nil rejection (conservative, not misleading); NATS KV is highly available in compose | Monitor rejection store health in production |

### 2.4 Risk Register Final State

| Risk | Initial Severity | Final Status |
|------|-----------------|-------------|
| Invariant coverage insufficient to prove lifecycle | HIGH | **RESOLVED** — 16% → 100%; all 8 categories exhaustively tested |
| PriceSource adds external dependency | MEDIUM | **RESOLVED** — NATS KV is internal infrastructure; fallback to "0" on failure; no external API |
| Rejection events add stream/consumer complexity | LOW | **RESOLVED** — Single stream, single consumer, monotonicity guard; pattern follows fill events exactly |
| Write-path integration tests fragile (httptest) | LOW | **ACCEPTED** — httptest provides deterministic venue simulation; real Binance calls reserved for smoke |
| Read-path consistency across three surfaces | MEDIUM | **PARTIALLY RESOLVED** — KV + HTTP consistent; ClickHouse rejection writer deferred (RG-1) |

---

## 3. Wave Quantitative Summary

| Metric | Value |
|--------|-------|
| Stages executed | 6 (S382–S387) |
| Governing questions | 9 (6 fully answered, 3 substantially answered) |
| Capabilities proven | 17 (13 FULL, 4 SUBSTANTIAL) |
| Invariant categories | 8 (ST, TERM, FR, IFC, QM, SM, SAFE, CORR) |
| Invariant coverage | 49/49 transition pairs (100%) |
| Go tests added | ~100 across 8 test files |
| Architecture docs produced | 7 |
| Stage reports produced | 6 |
| Production files modified/added | 9 |
| New domain events | 1 (VenueOrderRejectedEvent) |
| New NATS streams | 1 (EXECUTION_REJECTION_EVENTS) |
| New KV buckets | 1 (EXECUTION_VENUE_REJECTION_LATEST) |
| New interfaces | 1 (PriceSource) |
| Pre-existing gaps closed | 2 (G1, G10) |
| Intra-wave gaps closed | 3 (S385-G1, S386-G1, S386-G2) |
| Residual gaps | 10 (1 MEDIUM, 9 LOW — none blocking) |
| Regressions | 0 |
| Non-goals violated | 0 |

---

## 4. Next Ceremony Recommendation

### 4.1 What the Evidence Says

The OMS Foundation Wave has closed the critical gap between exchange listening/dry-run (S376–S381) and a fully auditable order lifecycle. The system now:

- Models every order lifecycle state (7 states, 10 transitions, 3 terminal absorbing states)
- Enforces lifecycle invariants exhaustively (100% coverage across 8 categories)
- Fills dry-run and paper orders with realistic market prices (CANDLE_LATEST KV)
- Proves write-path correctness across all three execution modes
- Publishes and persists rejection events for complete observability
- Provides a composite read-path with intent + fill + rejection + gate + propagation
- Maintains zero regressions across the entire codebase

The remaining gaps cluster around three themes:
1. **Persistence completeness** (ClickHouse rejection writer, lifecycle history) — mechanical wiring
2. **Operational hardening** (compose smoke for OMS, sustained stability re-proof) — automation packaging
3. **Execution capability expansion** (cancellation, async fills, multi-venue) — the strategic next step

### 4.2 Recommended Next Macro-Front

| Option | Description | Architectural Urgency |
|--------|-------------|----------------------|
| **A (Recommended)** | Testnet Venue Execution Proof | **HIGH** |
| B | Operational Hardening & Observability | MEDIUM |
| C | Multi-Exchange Expansion | LOW |

**Option A (Recommended): Testnet Venue Execution Proof**

The natural vertical continuation. Three waves have built the pipeline from multi-binary orchestration → live exchange listening → dry-run execution → OMS lifecycle foundation. The next frontier is exercising the `venue_live` path with a real testnet venue end-to-end, proving that the lifecycle operates correctly with actual exchange responses.

Rationale:
- The activation surface model (12 config combinations) has been proven, but only the `paper` and `dry_run` paths are exercised in compose. `venue_live` is proven only at integration-test level with httptest.
- The rejection event path exists but has only been tested with simulated rejections. Real venue rejections (rate limits, insufficient balance, invalid parameters) are untested.
- The ClickHouse writer gap (RG-1) is naturally closed when the venue execution path requires auditable persistence.
- The three-dimensional activation surface provides the safety envelope: testnet credentials + gate active + dry_run=false is required for venue_live.

**Option B: Operational Hardening & Observability**

Address RG-7 (OMS compose smoke), RG-8 (sustained stability), latency measurement, backpressure, and monitoring dashboards. Valuable for production readiness but less architecturally urgent — the system operates correctly within current operational bounds.

**Option C: Multi-Exchange Expansion**

Add exchange adapters beyond Binance Futures. Architecturally mechanical (activation surface is parametric), but premature before the single-venue execution lifecycle is fully proven with real exchange responses.

### 4.3 Ceremony Format

The next ceremony should follow the established wave pattern:

- **Type:** Wave Charter and Scope Freeze
- **Contents:** Frozen scope, non-goals, governing questions, capability targets, execution stage order
- **Prerequisite:** This evidence gate (S388) passed
- **Authority:** This gate does NOT open the next wave. The repository owner decides the direction and timing.

### 4.4 Promoted Documents

The following documents from this wave should be considered long-term architectural reference:

| Document | Reason |
|----------|--------|
| `canonical-order-model-and-lifecycle-state-machine.md` | Canonical reference for ExecutionIntent and seven-state lifecycle |
| `order-lifecycle-invariant-coverage-matrix-and-price-realism-findings.md` | Exhaustive invariant matrix — must be updated if state machine evolves |
| `write-path-integration-tests-by-execution-mode.md` | Write-path contract per mode — must be updated if new modes added |
| `rejection-event-path-and-write-path-observability.md` | Rejection event contract and observability model |
| `lifecycle-persistence-read-path-alignment-and-pricesource-wiring.md` | Composite read-path contract and PriceSource wiring |
| `oms-foundation-evidence-gate.md` | Wave closure record |
