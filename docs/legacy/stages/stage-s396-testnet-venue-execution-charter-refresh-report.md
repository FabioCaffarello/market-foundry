# S396 -- Testnet Venue Execution Proof Wave Charter Refresh Report

**Stage:** S396
**Type:** Charter refresh (wave reopening)
**Date:** 2026-03-22
**Wave:** Testnet Venue Execution Proof (refreshed, Spot-first)
**Predecessor:** S395 (Binance Segmentation Evidence Gate -- PASSED)
**Original charter:** S389

---

## 1. Executive Summary

Stage S396 formally reopens the Testnet Venue Execution Proof Wave with a
refreshed charter calibrated to the segmented Binance architecture delivered
by the S390--S395 Segmentation Foundation Wave.

The original S389 charter assumed a single venue adapter (Binance Futures
testnet) and defined 12 governing questions, 10 capability targets, and 22
non-goals. Before execution could begin, an architectural gap required an
interleaving wave (S390--S395) that delivered:

- Canonical `MarketSegment` dimension in the venue model.
- `BinanceSpotTestnetAdapter` implementing `VenuePort` + `VenueQueryPort`.
- Config-driven segment enablement with fail-closed semantics.
- Compose-level segment isolation (SUBSTANTIAL).
- Zero regressions on prior wave capabilities.

The segmentation wave consumed stage numbers S390--S395. This refresh:

1. Adopts a **Spot-first** execution strategy -- proving the lifecycle against
   Binance Spot testnet before Futures.
2. Resequences execution to stages **S397--S401**.
3. Adds two infrastructure stages (Spot ingest seed, dual-instance compose)
   that the original charter did not anticipate.
4. Adds 6 new non-goals (NG-23--NG-28) reflecting post-segmentation context.
5. Preserves all 12 governing questions and 10 capability targets unchanged.

The wave is now formally reopened with frozen scope.

---

## 2. State Assessment

### 2.1 What the Segmentation Wave Delivered

| Capability | Classification | Stage |
|---|---|---|
| Canonical venue model with segment dimension | FULL | S391 |
| Binance Spot testnet adapter | FULL | S394 |
| Config-driven segment enablement | FULL | S393 |
| Compose-level segment isolation | SUBSTANTIAL | S394 |
| Mainnet extensibility proof (structural) | FULL | S395 |

### 2.2 Residual Gaps Inherited from S395

| Gap | Severity | S396 disposition |
|---|---|---|
| G1: Concurrent multi-instance compose not proven | Low | Close in S398 |
| G2: Per-segment control gate not implemented | Low | Non-goal (NG-25) |
| G3: Spot ingest not seeded | Medium | Close in S397 |
| G4: Activation surface not queryable by segment | Low | Non-goal (NG-27) |
| G5: Shared core extraction not implemented | Low | Non-goal (NG-26) |

### 2.3 Residual Gaps Inherited from S388 (OMS Foundation)

| Gap | Severity | S396 disposition |
|---|---|---|
| RG-1: ClickHouse rejection writer not wired | Low | Close in S400 |
| RG-3: Fee realism in dry-run/paper | Low | Non-goal (NG-12) |
| RG-4: `sent` status never exercised E2E | Low | Non-goal |
| RG-5: `cancelled` via adapter not tested E2E | Low | Non-goal (NG-7) |
| RG-6: RejectionProjectionActor lacks unit tests | Low | Close opportunistically |
| RG-7: No OMS-specific compose smoke script | Low | Close in S398/S400 |

### 2.4 Infrastructure Readiness (Revised)

| Component | Ready | Source |
|---|---|---|
| BinanceSpotTestnetAdapter | Yes | S394 |
| BinanceFuturesTestnetAdapter | Yes | Pre-segmentation |
| DryRunSubmitter (wraps both adapters) | Yes | S379 |
| PaperVenueAdapter | Yes | Foundation |
| Segmented config validation | Yes | S393 |
| Compose overrides (Spot/Futures) | Yes | S394 |
| Kill switch + staleness guard | Yes | S344 |
| Activation surface (3D mode) | Yes | S337--S346 |
| EXECUTION_REJECTION_EVENTS stream | Yes | S386 |
| EXECUTION_VENUE_REJECTION_LATEST KV | Yes | S386 |
| Spot ingest bindings (`binances`) | **No** | S397 will close |
| Dual-instance compose | **No** | S398 will close |

---

## 3. Charter Refresh Decisions

### 3.1 Spot-First Strategy

All 12 governing questions are answered against the Binance Spot testnet first.
Rationale:

- Spot adapter is freshly built (S394) and needs real-world validation.
- Spot REST API is simpler (fewer margin/leverage edge cases).
- Spot `fills[]` array exercises the multi-fill aggregation path.
- Proving Spot validates segmentation under real venue load.
- Futures proof becomes an additive follow-on, not a prerequisite.

### 3.2 Stage Renumbering

S390--S395 were consumed by the segmentation wave. The refreshed wave:

| Old (S389 plan) | New (S396 refresh) | Description |
|---|---|---|
| S390 | S399 | Acceptance/fill/rejection lifecycle proof |
| S391 | S399 | Rejection lifecycle proof (merged into S399) |
| S392 | S400 | Partial fill + persistence + E2E (merged into S400) |
| S393 | S400 | OMS read-path (merged into S400) |
| S394 | S398 (infra) + S400 (full) | Compose E2E (split across stages) |
| S395 | S401 | Evidence gate |
| -- | S397 (NEW) | Spot ingest seed |
| -- | S398 (NEW) | Dual-instance compose proof |

The original 6 stages (S390--S395) are compressed to 4 (S397--S400) plus gate
(S401). This is possible because:

- Acceptance + rejection can be proven in a single stage (same adapter, same
  flow, different parameters).
- Partial fill + persistence + E2E compose share the same runtime and can be
  bundled.
- Two new precondition stages (S397, S398) are added to close segmentation gaps.

### 3.3 Non-Goal Expansion

6 new non-goals (NG-23--NG-28) are added to prevent scope inflation from the
post-segmentation context:

- NG-23: No parallel Futures proof.
- NG-24: No re-opening segmentation wave.
- NG-25: No per-segment control gate.
- NG-26: No shared core extraction.
- NG-27: No activation surface segment query.
- NG-28: No multi-exchange adapters.

---

## 4. Revised Stage Order

| Stage | Block | Description | Depends on |
|---|---|---|---|
| S396 | B0 | Charter refresh (this document) | S395 |
| S397 | B1 | Spot ingest binding seed and runtime projection closure | S396 |
| S398 | B2 | Dual-instance compose proof for segmented runtime | S397 |
| S399 | B3 | Spot real acceptance/fill/rejection lifecycle proof | S398 |
| S400 | B4 | Spot OMS read-path, auditability, and compose E2E proof | S399 |
| S401 | B5 | Evidence gate (final) | S400 |

---

## 5. Governing Questions (Revised Targeting)

All 12 questions preserved verbatim. Full mapping in companion document.

| Question | Revised target | Original target |
|---|---|---|
| TV-Q1: Acceptance + fill lifecycle | S399 | S390 |
| TV-Q2: Fill record fidelity | S399 | S390 |
| TV-Q3: Rejection lifecycle | S399 | S391 |
| TV-Q4: Rejection event fidelity | S399 | S391 |
| TV-Q5: Partial fill lifecycle | S400 | S392 |
| TV-Q6: Quantity monotonicity | S400 | S392 |
| TV-Q7: Persistence consistency | S400 | S393 |
| TV-Q8: ClickHouse rejection writer | S400 | S393 |
| TV-Q9: Compose E2E | S398/S400 | S394 |
| TV-Q10: Sustained operation | S400 | S394 |
| TV-Q11: Correlation chain | S399 | S390/S391 |
| TV-Q12: Post-200 reconciliation | S399 | S390 |

---

## 6. Non-Goals (28 Total)

22 original (NG-1--NG-22) preserved from S389.
6 new (NG-23--NG-28) added for post-segmentation context.

Key exclusions for this refresh:

- **NG-23:** No parallel Futures testnet proof in this wave.
- **NG-24:** Segmentation wave is closed. No venue model or config redesign.
- **NG-25:** Per-segment control gate is operational refinement, not blocking.
- **NG-28:** No multi-exchange. Segmentation is Binance-internal.

Full enumeration in
[`testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md`](../architecture/testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md).

---

## 7. Preparation Recommended for S397

Before S397 begins:

1. **Spot ingest source definition.** Define the `binances` source in NATS
   JetStream configuration. This includes the subjects for Spot market data
   (klines, ticker, depth as applicable).

2. **Seed script update.** Extend `make seed` to include Spot bindings or
   create a dedicated `make seed-spot` target.

3. **Spot testnet credentials.** Verify that Spot testnet API key and secret
   are provisioned and the account has test balance for BTCUSDT market orders.

4. **Config template.** Verify `deploy/configs/execute-spot.jsonc` is complete
   with:
   - `venue.type = "binance_spot_testnet"`
   - `venue.dry_run = true` (default; S399 will toggle to false)
   - `venue.segments.spot_enabled = true`
   - `venue.symbol` set to `BTCUSDT`

5. **Dual-compose skeleton.** Draft `docker-compose.dual.yaml` structure
   (to be completed in S398).

---

## 8. Artifacts Produced

| Artifact | Path |
|---|---|
| Charter refresh | `docs/architecture/testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md` |
| Revised capabilities, questions, and non-goals | `docs/architecture/testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md` |
| Stage report (this document) | `docs/stages/stage-s396-testnet-venue-execution-charter-refresh-report.md` |

---

## 9. Verdict

**S396 COMPLETE.** The Testnet Venue Execution Proof Wave is formally reopened
with refreshed charter, Spot-first strategy, revised stage order (S397--S401),
28 non-goals, and frozen scope. The 12 governing questions and 10 capability
targets are preserved unchanged and retargeted to the Binance Spot testnet.

---

## 10. Links

| Reference | Link |
|---|---|
| Charter refresh | [`../architecture/testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md`](../architecture/testnet-venue-execution-proof-wave-charter-refresh-segmented-spot-first.md) |
| Revised capabilities and non-goals | [`../architecture/testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md`](../architecture/testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md) |
| Original charter | [`../architecture/testnet-venue-execution-proof-wave-charter-and-scope-freeze.md`](../architecture/testnet-venue-execution-proof-wave-charter-and-scope-freeze.md) |
| Original capabilities and non-goals | [`../architecture/testnet-venue-execution-capabilities-questions-and-non-goals.md`](../architecture/testnet-venue-execution-capabilities-questions-and-non-goals.md) |
| Segmentation evidence gate | [`../architecture/binance-spot-futures-segmentation-evidence-gate.md`](../architecture/binance-spot-futures-segmentation-evidence-gate.md) |
| Segmentation evidence matrix | [`../architecture/binance-segmentation-evidence-matrix-residual-gaps-and-next-ceremony.md`](../architecture/binance-segmentation-evidence-matrix-residual-gaps-and-next-ceremony.md) |
| OMS Foundation evidence gate | [`../architecture/oms-foundation-evidence-gate.md`](../architecture/oms-foundation-evidence-gate.md) |
| Stage INDEX | [`INDEX.md`](INDEX.md) |
