# Unified Segment Runtime Foundation Wave -- Charter and Scope Freeze

**Wave:** Unified Segment Runtime Foundation
**Charter stage:** S398
**Date:** 2026-03-22
**Predecessor wave:** Binance Spot/Futures Segmentation Foundation (S390--S395, CLOSED)
**Predecessor stage:** S397 (Spot ingest binding seed -- complete)
**Authority:** This document freezes wave scope. Changes require a new stage.

---

## 1. Strategic Context

The Binance Segmentation Foundation Wave (S390--S395) proved that Spot and
Futures can coexist as architecturally separate segments with independent
adapters, credentials, NATS source values, and config validation. It chose a
**multi-binary per segment** approach: one execute binary per segment, one
compose override per segment, one config file per segment.

That approach was correct for proving segment isolation. It is **incorrect as
a permanent architecture**. The multi-binary split creates four structural
debts that compound as the platform evolves:

| Debt | Origin | Impact |
|---|---|---|
| D1: Sequential seed semantics | `seed-configctl.sh` activates one binding set at a time | Spot and Futures cannot be seeded concurrently; operator must run two seed commands |
| D2: One config active per binary | `execute-spot.jsonc` vs `execute-futures.jsonc` | Doubled maintenance surface; pipeline families, log, HTTP, NATS duplicated verbatim |
| D3: Compose override per segment | `docker-compose.spot.yaml` vs `docker-compose.futures.yaml` | Operator chooses one overlay; no single-command stack for both segments |
| D4: Source-aware routing distributed across actors | `websocket_actor.go` switches on `Source` string | No centralized segment routing; leakage risk grows with each new consumer |

S397 closed the Spot ingest binding gap and added `make seed-spot` / `make
seed-spot-multi`, but it left all four debts open with low-severity
classification. This wave promotes them to first-class problems and resolves
them with a unified runtime model.

---

## 2. Problem Redefinition

### 2.1 Current State (Multi-Binary per Segment)

```
execute-spot.jsonc ──> docker-compose.spot.yaml ──> execute binary (Spot only)
execute-futures.jsonc ──> docker-compose.futures.yaml ──> execute binary (Futures only)

make seed-spot        (sequential, one binding set)
make seed-futures     (sequential, one binding set)
```

Each segment runs in complete isolation. This is safe but operationally
expensive and structurally redundant.

### 2.2 Target State (Unified Segment Runtime)

```
execute.jsonc ──> docker-compose.yaml ──> execute binary (Spot + Futures)
  venue.segments.spot_enabled: true
  venue.segments.futures_enabled: true

make seed  (merged, both binding sets in single activation)
```

A single execute binary boots with both segments enabled. The config declares
which segments are active. The runtime projects segment-specific adapter
instances, credential sets, and NATS source routing from that single config.
Compose needs no per-segment overlay. Seed merges both binding sets.

### 2.3 Architectural Approach: Segment Array in Single Binary

The current `VenueConfig` selects **one** adapter via `venue.type`. The
unified model replaces this with a segment array:

```jsonc
{
  "venue": {
    "dry_run": true,
    "staleness_max_age": "120s",
    "submit_timeout": "10s",
    "segments": {
      "spot": {
        "enabled": true,
        "type": "binance_spot_testnet",
        "source": "binances"
      },
      "futures": {
        "enabled": true,
        "type": "binance_futures_testnet",
        "source": "binancef"
      }
    }
  }
}
```

Each enabled segment spawns its own adapter instance at startup. The execute
binary routes intents to the correct adapter based on the intent's source
field. The decorator pipeline (DryRunSubmitter, Post200Reconciler,
RetrySubmitter) wraps each adapter independently.

**Key invariant:** `dry_run` remains top-level and global. If `dry_run=true`,
ALL segment adapters are wrapped by DryRunSubmitter. There is no per-segment
dry-run toggle in this wave.

---

## 3. Wave Blocks (Ordered)

| Block | Stage | Title | Deliverable |
|---|---|---|---|
| B1 | S399 | Unified config model and segment enablement | Schema refactor, validation, migration path |
| B2 | S400 | Binding merge and multi-segment runtime projection | Merged seed, multi-adapter boot, intent routing |
| B3 | S401 | Segment-safe routing and leakage hardening | Consumer subject filtering, source-scoped dispatch, invariant tests |
| B4 | S402 | Single-compose coexistence proof | Compose E2E smoke with both segments active, concurrent health |
| B5 | S403 | Evidence gate: Unified Segment Runtime Foundation | Matrix evaluation, residual gap registry, wave close |

### Block Details

#### B1 -- Unified Config Model and Segment Enablement (S399)

Refactor `VenueConfig` to support a segment-indexed model where each segment
declares its own `type`, `source`, and `enabled` flag. The current
`SegmentConfig` (boolean-only) evolves into a richer structure.

Deliverables:
- New `VenueSegmentEntry` type with `enabled`, `type`, `source` fields.
- `VenueConfig.Segments` becomes `map[MarketSegment]VenueSegmentEntry`.
- Validation: at least one segment enabled when `venue.type` is removed;
  `dry_run` remains top-level; each segment's `type` must be in
  `knownVenueTypes`; `source` must match segment convention.
- Backward compatibility: `venue.type` (single value) accepted as shorthand
  for single-segment enablement during migration. Emit deprecation warning.
- Config migration documentation.

#### B2 -- Binding Merge and Multi-Segment Runtime Projection (S400)

Extend seed and runtime boot to project multiple segments from a single
config activation.

Deliverables:
- `seed-configctl.sh` reads all enabled segments and seeds bindings for each
  source in a single activation pass.
- Execute binary `Run()` iterates enabled segments, builds adapter per
  segment, wraps each in decorator pipeline.
- Intent routing: `VenueAdapterRouter` dispatches `ExecutionIntent` to the
  adapter matching `intent.Source`.
- Unknown source -> fail-closed rejection (no silent drop).
- Unit tests for multi-segment boot and routing.

#### B3 -- Segment-Safe Routing and Leakage Hardening (S401)

Harden the runtime to prevent cross-segment data flow.

Deliverables:
- NATS consumer subject filtering: each segment adapter subscribes only to
  subjects matching its source value.
- Source validation at intent ingress: reject intents with source values
  that don't match any enabled segment.
- Invariant tests proving:
  - Spot intent never reaches Futures adapter.
  - Futures intent never reaches Spot adapter.
  - Unknown-source intent is rejected, not dropped.
- Leakage audit of all NATS subject patterns used by execute binary.

#### B4 -- Single-Compose Coexistence Proof (S402)

Prove that both segments run correctly in a single compose stack with a
single config and single execute binary.

Deliverables:
- Unified `execute.jsonc` with both segments enabled.
- `docker-compose.yaml` runs execute with unified config (no overlay needed).
- Smoke script proving:
  1. Execute boots with both segments' adapters active.
  2. Spot dry-run intent flows through Spot adapter pipeline.
  3. Futures dry-run intent flows through Futures adapter pipeline.
  4. Fill/rejection events land on correct source-scoped NATS subjects.
  5. No cross-segment leakage detected.
  6. Health endpoint reports both segments.
- Per-segment compose overrides (`docker-compose.spot.yaml`,
  `docker-compose.futures.yaml`) remain valid for single-segment operation
  but are no longer required for dual-segment operation.

#### B5 -- Evidence Gate (S403)

Evaluate all governing questions against collected evidence. Close wave or
register residual gaps.

Deliverables:
- Evidence matrix with classification per capability and question.
- Residual gap registry.
- Wave verdict (PASS / PASS WITH GAPS / FAIL).
- Next ceremony recommendation.

---

## 4. Governing Questions

| ID | Question | Target block |
|---|---|---|
| USR-Q1 | Can a single config file express enablement for multiple market segments simultaneously? | B1 |
| USR-Q2 | Does config validation reject contradictory or incomplete multi-segment declarations at startup? | B1 |
| USR-Q3 | Does the backward-compatible migration path accept legacy single-segment configs without breakage? | B1 |
| USR-Q4 | Can a single seed activation produce bindings for all enabled segments? | B2 |
| USR-Q5 | Does the execute binary boot multiple adapter instances from a single config? | B2 |
| USR-Q6 | Does intent routing dispatch to the correct segment adapter based on source? | B2 |
| USR-Q7 | Is an intent with an unknown or disabled source rejected fail-closed? | B2, B3 |
| USR-Q8 | Can a Spot intent never reach the Futures adapter, and vice versa? | B3 |
| USR-Q9 | Does NATS consumer subject filtering prevent cross-segment message delivery? | B3 |
| USR-Q10 | Can both segments run concurrently in a single compose stack with a single binary? | B4 |
| USR-Q11 | Does the unified runtime preserve the global dry_run=true fail-closed invariant? | B1, B4 |
| USR-Q12 | Are per-segment compose overrides still valid for single-segment operation? | B4 |

---

## 5. What Enters (Scope Boundary)

1. Config schema evolution from single `venue.type` to segment-indexed model.
2. Config validation for multi-segment declarations.
3. Backward compatibility shim for legacy single-segment configs.
4. Merged seed activation for multiple sources.
5. Multi-adapter boot in single execute binary.
6. Intent-to-adapter routing by source field.
7. NATS consumer subject filtering per segment.
8. Cross-segment leakage invariant tests.
9. Single-compose E2E smoke with both segments active.
10. Evidence gate evaluation.

---

## 6. What Does NOT Enter (Scope Freeze)

See companion document:
[`unified-segment-runtime-capabilities-questions-and-non-goals.md`](unified-segment-runtime-capabilities-questions-and-non-goals.md)

Summary of frozen exclusions:

| ID | Exclusion | Rationale |
|---|---|---|
| NG-1 | Separate compose per segment as permanent model | This wave eliminates that need |
| NG-2 | Separate config file per segment as permanent model | Unified config replaces per-segment files |
| NG-3 | Multi-exchange support (beyond Binance) | Separate wave when justified |
| NG-4 | Full OMS (lifecycle, cancel, amend) | OMS wave concern |
| NG-5 | Portfolio risk management | Separate domain |
| NG-6 | Mainnet execution | Testnet only in this wave |
| NG-7 | Per-segment dry_run toggle | `dry_run` remains global and top-level |
| NG-8 | Multi-symbol routing within a single segment | Single symbol per segment instance |
| NG-9 | Ingest binary unification | Ingest segment routing is already source-aware (S397); no structural change needed |
| NG-10 | ClickHouse schema changes | Observability wave concern |
| NG-11 | WebSocket fill streaming | Separate execution concern |
| NG-12 | Advanced order types (limit, stop-loss, OCO) | Separate execution concern |
| NG-13 | Platform-wide actor topology redesign | Only execute-side routing changes |
| NG-14 | Credential rotation or vault integration | Env var model unchanged |
| NG-15 | Real trading activation | Dry-run only throughout wave |

---

## 7. Dependencies and Preconditions

| Dependency | Status | Source |
|---|---|---|
| Binance segmentation foundation (S390--S395) | CLOSED (PASS) | S395 evidence gate |
| Spot ingest binding seed (S397) | Complete | S397 report |
| Source-aware WebSocket routing | Complete | S397 (websocket_actor.go) |
| SegmentConfig in settings schema | Complete | S393 (schema.go) |
| DryRunSubmitter fail-closed semantics | Stable | S379 |
| Decorator pipeline (DryRun -> Post200 -> Retry) | Stable | S379, S344 |
| Multi-binary orchestration (S370--S375) | Proven | Compose patterns reusable |
| Activation surface (S337--S346) | Stable | Extend, do not replace |

---

## 8. Risk Registry

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Config schema migration breaks existing compose workflows | Medium | Medium | B1 backward-compat shim; deprecation warning, not hard break |
| Multi-adapter boot increases startup complexity | Low | Medium | Fail-closed: if any segment fails to initialize, binary exits |
| Source-based routing adds single point of failure | Low | High | B3 invariant tests; unknown source -> rejection, not drop |
| Merged seed creates ordering dependency | Low | Low | B2 seed is idempotent per source |
| Scope creep into per-segment dry_run | Medium | High | NG-7 frozen; reject at review |

---

## 9. Success Criteria

The wave is complete when:

1. A single `execute.jsonc` expresses both Spot and Futures segments.
2. Config validation rejects incomplete or contradictory multi-segment configs.
3. Legacy single-segment configs still boot correctly (backward compat).
4. `make seed` produces bindings for all enabled segments in one pass.
5. Execute binary boots with multiple adapter instances from unified config.
6. Intent routing dispatches correctly by source with fail-closed rejection.
7. NATS consumer filtering prevents cross-segment message delivery.
8. Single compose stack runs both segments concurrently without overlay.
9. Global `dry_run=true` wraps all segment adapters (fail-closed preserved).
10. All 12 governing questions answered at FULL or SUBSTANTIAL.
11. Evidence gate (S403) passes.

---

## 10. Relationship to Prior Waves

### Binance Segmentation Foundation (S390--S395)

That wave proved segment isolation via multi-binary split. This wave unifies
the split into a single-binary, single-config runtime while preserving the
isolation guarantees.

### Testnet Venue Execution Proof (S396--S401 original plan)

The S396 charter planned S398 as "dual-instance compose proof." This wave
**replaces** that plan with a deeper unification. The original S398--S401
stages (venue execution proof, OMS read-path, evidence gate) are renumbered
to follow after the unified runtime is established:

| S396 plan | This wave | Post-wave |
|---|---|---|
| S397: Spot ingest seed | S397 (done) | -- |
| S398: Dual-instance compose proof | **S398: This charter** | -- |
| S399: Acceptance/fill/rejection proof | -- | S404+ |
| S400: OMS read-path + E2E | -- | S405+ |
| S401: Evidence gate | -- | S406+ |

The venue execution governing questions (TV-Q1--TV-Q12) remain valid and
will be answered in the resumed wave on the unified runtime.

---

## 11. References

| Reference | Link |
|---|---|
| Companion: capabilities, questions, non-goals | [`unified-segment-runtime-capabilities-questions-and-non-goals.md`](unified-segment-runtime-capabilities-questions-and-non-goals.md) |
| S397 report (predecessor) | [`../stages/stage-s397-spot-ingest-binding-seed-report.md`](../stages/stage-s397-spot-ingest-binding-seed-report.md) |
| S395 evidence gate | [`../stages/stage-s395-binance-segmentation-evidence-gate-report.md`](../stages/stage-s395-binance-segmentation-evidence-gate-report.md) |
| S396 charter refresh | [`../stages/stage-s396-testnet-venue-execution-charter-refresh-report.md`](../stages/stage-s396-testnet-venue-execution-charter-refresh-report.md) |
| Segmentation wave charter | [`binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md`](binance-spot-futures-segmentation-wave-charter-and-scope-freeze.md) |
| Stage INDEX | [`../stages/INDEX.md`](../stages/INDEX.md) |
