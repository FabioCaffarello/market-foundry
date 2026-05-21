# Round-Trip Pairing Evidence Matrix, Residual Gaps, and Next Ceremony

## Evidence Matrix

### Capability Evidence

| ID | Capability | Verdict | Domain Tests | Application Tests | HTTP Wired | Architecture Doc |
|----|-----------|---------|-------------|-------------------|------------|-----------------|
| C-RT1 | Canonical Round-Trip Model | FULL | 26 | — | — | canonical-round-trip-and-leg-pairing-model.md |
| C-RT2 | FIFO Leg-Matching Strategy | FULL | 12 (subset of 26) | — | — | entry-exit-legs-pairing-rules-open-closed-unresolved-semantics-and-limitations.md |
| C-RT3 | Pairing Read Model | FULL | — | 9 | `/pairing`, `/pairing/chain` | pairing-read-model-and-attribution-integration.md |
| C-RT4 | Paired Batch Effectiveness | FULL | — | 3 | existing `/effectiveness/batch` modified | pairing-read-model-and-attribution-integration.md |
| C-RT5 | Round-Trip Review Endpoint | FULL | — | 10 | `/pairing/review`, `/pairing/review/chain` | round-trip-review-and-outcome-reconciliation.md |
| C-RT6 | Outcome Reconciliation | FULL | 7 | — | via review endpoint | fills-fees-pairing-result-reconciliation-semantics-and-limitations.md |

### Governing Question Evidence

| ID | Question | Verdict | Primary Test | Supporting Evidence |
|----|----------|---------|-------------|---------------------|
| Q-RT1 | Identify and pair legs | YES | 12 FIFO matching tests | IntentToLeg with 6 direction inference tests |
| Q-RT2 | Increase resolved rate | YES | TestGetEffectivenessSummary_PairingIntegration_ReducesUnresolved | executeBatch() now runs FIFO before classify |
| Q-RT3 | Correct P&L classification | YES | TestGetPairing_Batch_PairedRoundTrip (win), TestGetPairing_Batch_LossRoundTrip (loss) | ClassifyPair gross/net P&L |
| Q-RT4 | Surface outcomes and flag unmatched | YES | 10 review tests with state/outcome/flagged filters | UnmatchedReason codes, reconciliation flags |
| Q-RT5 | No new infrastructure | YES | All tests use stubCompositeReader over existing schema | Zero new ClickHouse tables |

### Guard Rail Evidence

| # | Guard Rail | Verified By |
|---|-----------|------------|
| 1 | No OMS expansion | No position/portfolio types added; grep confirms no new position state |
| 2 | No new ClickHouse tables | No DDL in wave; CompositeReader unchanged |
| 3 | No new exchange connectivity | No adapter/venue code modified |
| 4 | No write-path changes | `cmd/writer/pipeline.go` not touched by wave stages |
| 5 | No portfolio analytics | Per-decision only; no cross-symbol aggregation |
| 6 | No real-time streaming | Batch read-path at query time |
| 7 | No domain type refactoring | New package `internal/domain/pairing/`; existing packages unchanged |
| 8 | No UI or dashboards | JSON HTTP endpoints only |
| 9 | No risk/position engine | No risk state types added |
| 10 | Additive only | All modifications are additive wiring; existing tests unbroken |

## Residual Gaps

### From This Wave

| ID | Gap | Severity | Root Cause | Impact | Mitigation |
|----|-----|----------|-----------|--------|-----------|
| G-RT1 | Futures fees structurally zero | LOW | Binance Futures API does not return fee in RESULT response (inherited from S428) | Net P&L overstates returns for futures round-trips; `fee_gap` flag raised | Reconciliation flag `fee_gap` surfaces the issue; `fee_reliable=false` prevents silent misattribution |
| G-RT2 | Paper/dry-run fills have zero cost basis | LOW | Simulated fills carry no real price data | Paired paper round-trips return `outcome_unresolved` despite having both legs | `simulated` and `cost_basis_zero` flags surface this; not a real-money concern |
| G-RT3 | FIFO only; no LIFO/HIFO | LOW | Explicit non-goal (NG-RT7) | FIFO may not match trader intent when multiple entries exist | Documented as design choice; additional strategies can be added later without schema changes |
| G-RT4 | No cross-session pairing | LOW | Explicit non-goal (NG-RT6) | Entry in session A, exit in session B remains unresolved | Session-scoped pairing is correct for current operational model |
| G-RT5 | Strategy direction defaults to long | LOW | Some chains lack explicit strategy direction metadata | Short strategies may have entry/exit mislabeled | Correct for majority of current strategies; direction override possible in future |
| G-RT6 | No causal validation in matching | LOW | Structural matching (symbol/side/time) vs causal (correlation chain) | FIFO may pair legs from different signal chains if symbol/timing coincides | Acceptable for current volume; correlation-ID post-filtering possible |
| G-RT7 | Float64 quantity precision | LOW | Standard floating-point arithmetic with epsilon=1e-12 | Very small or very large quantities could produce matching artifacts | Epsilon tolerance tested; acceptable for all current trade sizes |
| G-RT8 | Fee-asset currency conversion absent | LOW | No price feeds for cross-currency fee comparison | `fee_asset_mismatch` flag raised but not resolved to common denomination | Flag surfaces the gap; conversion requires external price data (out of scope) |

### Inherited Gaps (Unchanged)

| ID | From Wave | Gap | Status |
|----|-----------|-----|--------|
| G-SE2 | S474–S478 | No statistical significance on cohort comparisons | LOW — by design (NG-SE2) |
| G-SE4 | S474–S478 | No temporal decomposition within single query | LOW — multiple queries as workaround |
| G-SE6 | S474–S478 | No cross-symbol aggregation | LOW — by design (NG-SE1) |

### Gap Severity Summary

| Severity | Count | Action |
|----------|-------|--------|
| CRITICAL | 0 | — |
| HIGH | 0 | — |
| MEDIUM | 0 | — |
| LOW | 11 (8 new + 3 inherited) | Documented, flagged, acceptable trade-offs |

No gap blocks wave closure.

## Regression Verification

| Package | Pre-Wave Tests | Post-Wave Tests | Regressions |
|---------|---------------|-----------------|-------------|
| `internal/domain/pairing` | 0 (new package) | 33 | 0 |
| `internal/domain/effectiveness` | existing suite | existing suite | 0 |
| `internal/application/analyticalclient` | existing suite | existing + 22 new | 0 |
| `internal/interfaces/http/handlers` | existing suite | existing suite | 0 |
| `cmd/gateway` | compiles, routes wired | compiles, routes wired | 0 |

## Files Changed in Wave

### New Files (S480–S482)

| File | Stage | Purpose |
|------|-------|---------|
| `internal/domain/pairing/pairing.go` | S480 | Leg, RoundTrip, MatchFIFO, IntentToLeg |
| `internal/domain/pairing/pairing_test.go` | S480 | 26 domain tests |
| `internal/domain/pairing/reconciliation.go` | S482 | ReconciliationFlag, ReconcileRoundTrip |
| `internal/domain/pairing/reconciliation_test.go` | S482 | 7 reconciliation tests |
| `internal/application/analyticalclient/pairing_contracts.go` | S481 | PairingQuery, PairingReply, RoundTripView |
| `internal/application/analyticalclient/get_pairing.go` | S481 | GetPairingUseCase |
| `internal/application/analyticalclient/s481_pairing_read_model_test.go` | S481 | 12 pairing read model tests |
| `internal/application/analyticalclient/review_contracts.go` | S482 | RoundTripReviewQuery, ReviewSummary |
| `internal/application/analyticalclient/get_roundtrip_review.go` | S482 | GetRoundTripReviewUseCase |
| `internal/application/analyticalclient/s482_roundtrip_review_test.go` | S482 | 10 review tests |

### Modified Files (S481–S482)

| File | Stage | Change |
|------|-------|--------|
| `internal/application/analyticalclient/get_effectiveness.go` | S481 | executeBatch now runs FIFO → ClassifyPair before single-leg classify |
| `internal/application/analyticalclient/get_effectiveness_summary.go` | S481 | Same pairing integration for cohort aggregation |
| `internal/interfaces/http/handlers/composite.go` | S481, S482 | Added GetPairing, GetPairingSingle, GetRoundTripReview, GetRoundTripReviewSingle |
| `internal/interfaces/http/routes/analytical.go` | S481, S482 | Added route registrations and interface deps |
| `cmd/gateway/compose.go` | S481, S482 | Wired GetPairingUseCase and GetRoundTripReviewUseCase |

### Architecture Documents Produced

| Document | Stage | Purpose |
|----------|-------|---------|
| canonical-round-trip-and-leg-pairing-model.md | S480 | Core entity definitions, matching rules, integration points |
| entry-exit-legs-pairing-rules-open-closed-unresolved-semantics-and-limitations.md | S480 | Operational semantics reference |
| pairing-read-model-and-attribution-integration.md | S481 | Read model design, effectiveness wiring |
| round-trip-read-surfaces-realized-vs-unresolved-attribution-and-limitations.md | S481 | Attribution semantics |
| round-trip-review-and-outcome-reconciliation.md | S482 | Review surface, flags, reliability signals |
| fills-fees-pairing-result-reconciliation-semantics-and-limitations.md | S482 | Fill aggregation, fee semantics, reconciliation rules |
| round-trip-pairing-evidence-gate.md | S483 | This wave's evidence gate |

## Next Ceremony Recommendation

### Assessment

The Round-Trip Pairing wave closes the structural gap (G-SE1) that prevented
effective classification of execution outcomes. The measurement stack now spans:

1. **Signal → Decision**: Lineage (S469–S473)
2. **Decision → Execution**: Effectiveness classification (S474–S478)
3. **Execution → Round-Trip**: Pairing, attribution, reconciliation (S479–S483)

Each layer is read-path only, additive, and operates on existing data. No
critical or high-severity gaps remain across the three measurement waves.

### Recommended Next Direction

The next macro-front should be selected from the following candidates, in order
of strategic value:

1. **Operational automation and monitoring hardening** — The measurement stack
   is now three waves deep. Before expanding further, harden the operational
   surface: alerting on anomalous resolved rates, automated post-session
   verification using pairing data, and integration of reconciliation flags into
   operational dashboards. This consolidates rather than expands.

2. **Cross-session position continuity** — G-RT4 (no cross-session pairing) is
   the most impactful residual gap for operators running multi-session strategies.
   A short wave could extend pairing across session boundaries using existing
   correlation IDs without requiring OMS or position tracking.

3. **Futures fee recovery** — G-RT1 (futures fees structurally zero) affects P&L
   accuracy for the entire futures segment. Requires write-path changes to fetch
   fees from a separate API call, which breaks the current wave's guard rails
   but is operationally important.

### What NOT to Open Next

- Portfolio analytics or cross-symbol aggregation (premature)
- ML-based scoring or predictive effectiveness (no data foundation yet)
- Real-time streaming pairing (latency requirements undefined)
- LIFO/HIFO matching strategies (FIFO sufficient for current volume)
- Dashboard or visualization layer (operational surface not yet stabilized)

### Ceremony Closure

This document, together with the evidence gate, constitutes the formal closure
of the Round-Trip Pairing wave. The wave opens no successor directly.
The next wave charter must be opened in a separate stage.
