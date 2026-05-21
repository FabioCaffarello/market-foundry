# Production Hardening and Mainnet Readiness Audit -- Capabilities, Questions, and Non-Goals

## Capabilities to Prove

| ID | Capability | Block | Success Criteria |
|---|---|---|---|
| C-1 | Canonical fee model defined | S428 | A single FillRecord field carries a normalized fee value interpretable without source-branching logic. |
| C-2 | Spot commission normalized | S428 | Spot fills populate the canonical fee field from per-leg commission aggregation. |
| C-3 | Futures fee normalized | S428 | Futures fills populate the canonical fee field from the best available venue data (cumQuote or equivalent). |
| C-4 | Raw fee preservation | S428 | Venue-specific raw fee fields remain in metadata JSON for auditability. Both ClickHouse and KV paths preserve raw values. |
| C-5 | Cross-segment fee query parity | S428 | A single read-path query returns fee data with consistent semantics for Spot and Futures fills. |
| C-6 | Per-segment health signal | S429 | Each active segment exposes connectivity status, last-event-age, and error indicators via a queryable NATS subject. |
| C-7 | Health isolation between segments | S429 | A degraded Futures segment does not cause Spot health to report degraded, and vice versa. |
| C-8 | Health fail-closed semantics | S429 | If health signals are absent or stale for a segment, the segment is reported as unhealthy rather than silently healthy. |
| C-9 | Mainnet readiness checklist evaluated | S430 | Every item in the mainnet readiness checklist has an explicit PASS, FAIL, or DEFERRED verdict with evidence reference. |
| C-10 | KV history strategy decided | S430 | RG-3 has a formal KEEP (latest-only) or CHANGE decision with rationale and, if CHANGE, a migration plan. |
| C-11 | Credential separation assessment | S430 | The audit evaluates whether the adapter layer cleanly separates testnet and mainnet credentials with no cross-contamination risk. |
| C-12 | Capital control assessment | S430 | The audit evaluates whether position-limit or notional-limit enforcement exists or is needed before mainnet. |
| C-13 | Evidence gate verdict | S431 | The wave has a scored evidence matrix with per-capability verdict and an overall wave classification. |

## Governing Questions

These questions define the boundaries and priorities of the wave. Every stage must contribute to answering at least one.

### Fee Normalization (S428)

1. **Can a single canonical fee field represent both Spot per-leg commission and Futures cumQuote without information loss?**
   If not, what is the minimal field set that achieves consistent consumer semantics?

2. **Does the normalized fee model require the `/fapi/v1/userTrades` endpoint to be useful, or is cumQuote sufficient as a proxy for production analytics?**
   If `/userTrades` is required, the wave recommends it for a future stage but does not implement it.

3. **Can the ClickHouse schema accept the normalized field without a breaking migration?**
   The answer determines whether the change is additive (ALTER TABLE ADD COLUMN) or requires a migration ceremony.

### Health and Readiness Signals (S429)

4. **What is the minimal set of signals that allows an operator to determine segment health without reading logs?**
   The answer defines the health query response schema.

5. **Should segment health be push-based (continuous publication) or pull-based (request/reply)?**
   The answer determines the architectural pattern. The charter recommends pull-based to avoid hot-path overhead.

6. **Does the existing actor health tracker infrastructure support per-segment scoping, or does it need extension?**
   The answer determines the implementation complexity of S429.

### Mainnet Readiness Audit (S430)

7. **What is the complete checklist of items that must be evaluated before a mainnet enablement decision can be made?**
   The answer defines the audit scope. The charter provides an initial checklist; S430 may refine it.

8. **Is latest-only KV semantics sufficient for production operations, or does operational use require historical state access?**
   The answer closes or escalates RG-3.

9. **Does the current adapter architecture allow mainnet credentials to be configured without code changes?**
   The answer determines whether credential separation is a config concern or requires adapter refactoring.

10. **What capital controls (if any) must exist before the first mainnet order can be submitted?**
    The answer determines whether capital controls are a prerequisite for mainnet or a separate concern.

## Non-Goals

### NG-1: Mainnet Enablement

This wave audits readiness; it does not activate mainnet execution paths. No mainnet credentials will be configured. No mainnet orders will be submitted. The kill switch (EXECUTION_CONTROL) remains testnet-scoped. The existing NG-1 enforcement from prior waves remains in full effect.

**Why**: Mainnet activation requires a separate authorization ceremony with explicit risk acceptance. This wave produces the information needed for that ceremony.

### NG-2: Multi-Exchange Support

The system remains Binance-only. No adapter work for other exchanges (Bybit, OKX, Coinbase, etc.) is authorized.

**Why**: Multi-exchange expansion is an architectural wave, not a hardening wave. The current Binance-only scope provides sufficient surface for production readiness evaluation.

### NG-3: OMS Expansion

No limit orders, order amendments, order cancellation API, or advanced order types. The system remains market-order-only.

**Why**: OMS expansion requires lifecycle model changes that are out of scope for a hardening wave. The seven-state lifecycle model is frozen.

### NG-4: Dashboard or UI Development

Operational signals are exposed via NATS query subjects. No web UI, Grafana dashboards, or visualization layer is in scope.

**Why**: Visualization is a consumer concern. This wave ensures the data is queryable; how it is displayed is a separate decision.

### NG-5: Config or Compose Surface Re-Expansion

The S416--S418 simplification reduced configs from 6 to 3 and compose overlays from 7 to 3. This wave must not re-expand these surfaces.

**Why**: Entropy reduction was a hard-won deliverable. Re-expansion without strong justification would regress operational clarity.

### NG-6: `/fapi/v1/userTrades` Integration

If fee normalization determines that true Futures commission (as opposed to cumQuote proxy) is needed, the wave documents the recommendation but does not implement the endpoint integration.

**Why**: The `/userTrades` endpoint requires a separate HTTP client path, rate-limit management, and reconciliation logic. This is implementation work beyond the scope of a normalization stage.

### NG-7: Documentation Governance Ceremony

The 97 untracked docs (RG-16) remain deferred. No documentation taxonomy, format standardization, or bulk governance is authorized.

**Why**: Documentation governance has no runtime impact and requires a dedicated ceremony with different success criteria.

### NG-8: Large Structural Refactoring

No package reorganization, binary consolidation, or architectural redesign. Changes are surgical and scoped to the four blocks defined in the charter.

**Why**: The current architecture has proven stable across 11 consecutive wave passes. Refactoring risk is not justified for a hardening wave.

### NG-9: Pagination, Advanced Filtering, or Analytical Query Expansion

Read-path queries remain as delivered in S413. No pagination, time-range filtering on KV queries, or analytical aggregation endpoints.

**Why**: Current operational queries are sufficient for the bounded cardinality of active trading state. Analytical needs are served by ClickHouse direct queries.

### NG-10: Parallel Spot+Futures Live Execution Proof

Simultaneous live execution across both segments in a single compose run is not in scope. Each segment has been individually proven; parallel proof is deferred.

**Why**: Parallel execution proof requires coordinated testnet funding and concurrent position management. It is a valid future proof but not a production hardening prerequisite.
