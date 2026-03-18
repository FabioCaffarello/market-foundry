# Governance Hygiene Status

> Living document. Tracks the alignment between governance artifacts (docs, CLI, registries) and the actual runtime/architecture state.
> Last updated: S34 (2026-03-17).

---

## Governance Dimensions

| Dimension | Description | Status |
|-----------|-------------|--------|
| **Actor ownership** | `actor-ownership.md` reflects actual supervision trees | Aligned (S33) |
| **Stream ownership** | `stream-ownership-matrix.md` lists all streams, durables, KV buckets, queries | Aligned (S33) |
| **Family catalog** | `stream-family-catalog.md` readiness status matches implementation | Aligned (S33) |
| **Projection matrix** | `projection-family-matrix.md` lists all active projections | Aligned (S33) |
| **Stream families** | `stream-families.md` evidence section reflects all 3 types | Aligned (S33) |
| **raccoon-cli topology** | Durable names, subject prefixes match runtime | Aligned (S33) |
| **raccoon-cli bindings** | Expected durables, query subjects match runtime | Aligned (S33) |
| **raccoon-cli drift** | Architecture doc inventory covers all key docs | Aligned (S33) |
| **Signal entry guard** | Premature SIGNAL_EVENTS/PROJECTION_EVENTS entry blocked by CLI | Enforced (S33) |
| **Config-driven activation** | Derive/store family activation via `pipeline.families` config | Aligned (S34) — both derive and store filter by config |

---

## Inventory: Active Evidence Families

| Family | Derive Actor | Store Consumer | Store Projection | KV Bucket(s) | Query Subject(s) | HTTP Endpoint(s) |
|--------|-------------|---------------|-----------------|-------------|-----------------|-----------------|
| candle | SamplerActor | store-candle | CandleProjectionActor | CANDLE_LATEST, CANDLE_HISTORY | candle.latest, candle.history | /evidence/candles/latest, /evidence/candles/history |
| tradeburst | TradeBurstSamplerActor | store-trade-burst | TradeBurstProjectionActor | TRADE_BURST_LATEST | tradeburst.latest | /evidence/tradeburst/latest |
| volume | VolumeSamplerActor | store-volume | VolumeProjectionActor | VOLUME_LATEST | volume.latest | /evidence/volume/latest |

---

## Inventory: Planned Families (Not Yet Implemented)

| Family | Status | Blocked By | Earliest Stage |
|--------|--------|------------|---------------|
| signal | Planned | Architecture approval, dedicated design doc | S36+ |
| projection | Planned | Gateway caching need | TBD |
| stats | Planned | No blocker (evidence type, follows pattern) | S35+ |

---

## Signal Entry Prerequisites

Signal domain entry is explicitly blocked until all of the following are met:

| # | Prerequisite | Status | Evidence |
|---|-------------|--------|----------|
| 1 | 3+ evidence types proven end-to-end | MET | candle, tradeburst, volume |
| 2 | FamilyProcessor pattern validated | MET | 3 families, SourceScopeActor untouched |
| 3 | ProjectionPipeline pattern validated | MET | 3 pipelines, StoreSupervisor untouched |
| 4 | Actor ownership docs current | MET (S33) | Updated with all 3 evidence pipelines |
| 5 | raccoon-cli topology rules current | MET (S33) | Durables, subjects, signal guard added |
| 6 | Config-driven activation proven | MET (S34) | `pipeline.families` controls derive and store; binding watchers control source/symbol activation |
| 7 | Architecture approval for signal domain | NOT MET | Requires dedicated design doc |

**Enforcement:** raccoon-cli now includes premature-entry guards in both `topology.rs` (pipeline-continuity check) and `drift_detect.rs` (premature-domain-entry check). Any code referencing `SIGNAL_EVENTS` or `PROJECTION_EVENTS` will trigger CI errors.

---

## Remaining Governance Gaps

| # | Gap | Severity | Impact |
|---|-----|----------|--------|
| 1 | Binding deactivation incomplete | MEDIUM | Cleared events logged only; requires process restart to remove bindings |
| 2 | QueryResponderActor scales manually | LOW | Each new evidence type requires manual additions to 5 sections |
| 3 | No projection lag metric | LOW | Operations cannot detect store falling behind derive |
| 4 | Single exchange adapter (binancef) | LOW | Multi-source not tested; not blocking for signal |
| 5 | Trade burst/volume lack history buckets | LOW | Intentional; add when analytical need justifies it |

---

## Governance Hygiene Cadence

- **Every new evidence type:** Update all 5 governance docs + raccoon-cli constants.
- **Every new stream family:** Full Family Addition Checklist (stream-families.md) + governance docs.
- **Every 3 stages:** Review this status document for accuracy.
- **Before any new domain entry:** Verify all prerequisites in the relevant readiness review.
