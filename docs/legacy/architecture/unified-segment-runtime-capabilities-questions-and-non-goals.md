# Unified Segment Runtime Foundation -- Capabilities, Questions, and Non-Goals

**Wave:** Unified Segment Runtime Foundation
**Charter stage:** S398
**Date:** 2026-03-22
**Companion:** [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md)

---

## 1. Capability Targets

| ID | Capability | Description | Target block |
|---|---|---|---|
| C1 | Unified config model | Single config file expresses multiple market segments with per-segment type, source, and enablement | B1 (S399) |
| C2 | Multi-segment validation | Config validation rejects incomplete, contradictory, or disabled-only segment declarations at startup | B1 (S399) |
| C3 | Backward-compatible migration | Legacy single-segment configs (with `venue.type`) boot correctly with deprecation warning | B1 (S399) |
| C4 | Merged binding seed | Single seed activation produces NATS bindings for all enabled segments | B2 (S400) |
| C5 | Multi-adapter runtime projection | Execute binary boots one adapter instance per enabled segment from a single config | B2 (S400) |
| C6 | Source-based intent routing | `VenueAdapterRouter` dispatches intents to segment adapter matching `intent.Source` | B2 (S400) |
| C7 | Fail-closed unknown source rejection | Intents with source values not matching any enabled segment are rejected, not dropped | B2, B3 (S400, S401) |
| C8 | Cross-segment leakage prevention | NATS consumer subject filtering and source validation prevent Spot/Futures data mixing | B3 (S401) |
| C9 | Single-compose coexistence | Both segments run concurrently in one compose stack with one binary and one config | B4 (S402) |
| C10 | Global dry_run preservation | `dry_run=true` wraps ALL segment adapters uniformly; no per-segment override | B1, B4 (S399, S402) |

---

## 2. Governing Questions

### B1 -- Unified Config Model (S399)

| ID | Question | Evidence required |
|---|---|---|
| USR-Q1 | Can a single config file express enablement for multiple market segments simultaneously? | Config example with both `spot` and `futures` segments enabled; parse test passing |
| USR-Q2 | Does config validation reject contradictory or incomplete multi-segment declarations at startup? | Validation test matrix: missing source, unknown type, disabled-only segments, duplicate source values |
| USR-Q3 | Does the backward-compatible migration path accept legacy single-segment configs without breakage? | Test: legacy config with `venue.type` boots correctly; deprecation warning emitted in logs |
| USR-Q11 | Does the unified runtime preserve the global dry_run=true fail-closed invariant? | Validation: `dry_run` absent/null -> true for all segments; no per-segment override path exists |

### B2 -- Binding Merge and Multi-Segment Runtime Projection (S400)

| ID | Question | Evidence required |
|---|---|---|
| USR-Q4 | Can a single seed activation produce bindings for all enabled segments? | Seed script test: single `make seed` produces both `binancef.*` and `binances.*` bindings in configctl |
| USR-Q5 | Does the execute binary boot multiple adapter instances from a single config? | Startup log showing both adapters initialized; health endpoint listing both segments |
| USR-Q6 | Does intent routing dispatch to the correct segment adapter based on source? | Unit test: Spot-sourced intent reaches Spot adapter; Futures-sourced intent reaches Futures adapter |
| USR-Q7 | Is an intent with an unknown or disabled source rejected fail-closed? | Unit test: intent with `source=unknown` triggers rejection event, not silent drop |

### B3 -- Segment-Safe Routing and Leakage Hardening (S401)

| ID | Question | Evidence required |
|---|---|---|
| USR-Q8 | Can a Spot intent never reach the Futures adapter, and vice versa? | Cross-segment invariant test: N intents per segment, zero cross-delivery |
| USR-Q9 | Does NATS consumer subject filtering prevent cross-segment message delivery? | Subject audit + consumer config test: each adapter subscribes only to its source subjects |

### B4 -- Single-Compose Coexistence Proof (S402)

| ID | Question | Evidence required |
|---|---|---|
| USR-Q10 | Can both segments run concurrently in a single compose stack with a single binary? | Smoke script: unified compose boot, both segments process dry-run intents, fills land on correct subjects |
| USR-Q12 | Are per-segment compose overrides still valid for single-segment operation? | Smoke: `docker-compose.spot.yaml` still boots Spot-only; `docker-compose.futures.yaml` still boots Futures-only |

---

## 3. Non-Goals (Frozen Exclusions)

### 3.1 Compose and Config Scope

| ID | Exclusion | Rationale |
|---|---|---|
| NG-1 | Separate compose per segment as permanent model | This wave unifies compose; per-segment overrides remain optional, not required |
| NG-2 | Separate config file per segment as permanent model | Unified config replaces the need for per-segment files |
| NG-7 | Per-segment dry_run toggle | `dry_run` is a global safety invariant; per-segment override adds risk without current value |

### 3.2 Platform Scope

| ID | Exclusion | Rationale |
|---|---|---|
| NG-3 | Multi-exchange support (beyond Binance) | No second exchange adapter exists; segmentation is Binance-internal |
| NG-4 | Full OMS (lifecycle, cancel, amend) | OMS Foundation Wave (S382--S388) delivered the foundation; completion is a separate wave |
| NG-5 | Portfolio risk management | Separate domain, not execution plumbing |
| NG-6 | Mainnet execution | All proofs on testnet only |
| NG-13 | Platform-wide actor topology redesign | Only execute-side adapter routing changes; ingest, store, derive untouched |
| NG-15 | Real trading activation | Dry-run throughout wave |

### 3.3 Execution Scope

| ID | Exclusion | Rationale |
|---|---|---|
| NG-8 | Multi-symbol routing within a single segment | Single symbol per segment instance; multi-symbol is a future enhancement |
| NG-11 | WebSocket fill streaming | REST-based fill capture only; WS fills are a separate concern |
| NG-12 | Advanced order types (limit, stop-loss, OCO) | Market order only in current adapters |

### 3.4 Infrastructure Scope

| ID | Exclusion | Rationale |
|---|---|---|
| NG-9 | Ingest binary unification | Ingest already has source-aware routing (S397); no structural change needed for this wave |
| NG-10 | ClickHouse schema changes | Observability enhancement, not runtime unification |
| NG-14 | Credential rotation or vault integration | Env var model is sufficient for testnet |

---

## 4. Classification Targets

The evidence gate (S403) will evaluate each capability against this scale:

| Classification | Definition |
|---|---|
| **FULL** | All evidence present, no exceptions, all invariant tests pass |
| **SUBSTANTIAL** | Primary evidence present, minor gaps that do not compromise safety |
| **PARTIAL** | Some evidence present but key questions remain open |
| **NONE** | No evidence or evidence contradicts the claim |

**Wave pass threshold:** All 10 capabilities at FULL or SUBSTANTIAL. No
capability at PARTIAL or NONE.

---

## 5. Evidence Collection Plan

| Stage | Artifacts expected |
|---|---|
| S399 | Schema types, validation tests, config examples, migration tests |
| S400 | Router implementation, multi-boot tests, seed script, intent routing tests |
| S401 | Subject audit, consumer filter tests, cross-segment invariant tests |
| S402 | Unified compose file, smoke script, concurrent health proof, leakage absence proof |
| S403 | Evidence matrix, gap registry, verdict document |

---

## 6. Relationship to Prior Non-Goal Sets

### Segmentation Wave Non-Goals (S390, 13 items)

All 13 frozen exclusions from the segmentation wave remain respected. This
wave does not reopen any of them. Key preserved exclusions:

- NG-SEG-1: Mainnet execution (testnet only) -- preserved as NG-6.
- NG-SEG-2: Multi-exchange -- preserved as NG-3.
- NG-SEG-7: Multi-symbol routing -- preserved as NG-8.
- NG-SEG-13: Platform-wide redesign -- preserved as NG-13.

### Testnet Venue Execution Non-Goals (S396, 28 items)

The 28 non-goals from the refreshed testnet venue wave are not affected.
This wave inserts before the venue execution stages (which are renumbered
to S404+). Those non-goals will be re-evaluated when the venue execution
wave resumes.

---

## 7. References

| Reference | Link |
|---|---|
| Wave charter | [`unified-segment-runtime-wave-charter-and-scope-freeze.md`](unified-segment-runtime-wave-charter-and-scope-freeze.md) |
| Segmentation wave non-goals | [`binance-segmentation-capabilities-questions-and-non-goals.md`](binance-segmentation-capabilities-questions-and-non-goals.md) |
| Testnet venue execution non-goals | [`testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md`](testnet-venue-execution-spot-first-capabilities-questions-and-non-goals.md) |
| S395 evidence gate | [`../stages/stage-s395-binance-segmentation-evidence-gate-report.md`](../stages/stage-s395-binance-segmentation-evidence-gate-report.md) |
| S397 report | [`../stages/stage-s397-spot-ingest-binding-seed-report.md`](../stages/stage-s397-spot-ingest-binding-seed-report.md) |
