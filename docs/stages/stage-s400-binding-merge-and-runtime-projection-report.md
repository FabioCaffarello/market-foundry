# S400 — Binding Merge and Multi-Segment Runtime Projection

> Stage report | 2026-03-22 | Phase 42: Unified Segment Runtime Foundation Wave

## Objective

Eliminate the sequential seed limitation and single-adapter runtime model.
After S400, Spot and Futures coexist in the same runtime without separate seeds
and without artificial "first enabled segment" selection.

## What Changed

### Source-Segment Mapping (`internal/shared/settings/schema.go`)

Added canonical mapping between ingest source prefixes and market segments:

- `SourceForSegment(seg)` — returns source prefix (e.g., futures -> "binancef")
- `SegmentForSource(src)` — returns segment (e.g., "binances" -> spot)

Static map: `{futures: "binancef", spot: "binances"}`.

### SegmentRouter (`internal/application/execution/segment_router.go`)

New type implementing `VenuePort` and `VenueQueryPort`. Routes `SubmitOrder`
calls by matching `intent.Source` -> segment -> registered adapter.

- `Register(seg, adapter)` — adds submit adapter for segment
- `RegisterQuery(seg, query)` — adds query adapter for segment
- Fail-closed: unknown source or unregistered segment returns Problem

### Runtime Build (`cmd/execute/run.go`)

`buildVenueAdapterFromSegments` rewritten to:

1. Build an adapter for EACH enabled segment (not just the first)
2. Register each in a SegmentRouter
3. Return the router as the composite VenuePort

For single-segment configs, the router contains one adapter — no behavioral
change from S399. For multi-segment configs, the router dispatches by source.

### Seed Merge (`scripts/seed-configctl.sh`)

Added `--merge` flag:

- Accepts `SOURCES` env var (default: `binancef,binances`)
- Generates bindings for each source x symbol in a single config document
- Activates one config carrying all source bindings

New Makefile targets:

| Target | Description |
|--------|-------------|
| `make seed-unified` | Merged Spot+Futures single-symbol |
| `make seed-unified-multi` | Merged Spot+Futures multi-symbol |

### Config and Compose

- `deploy/configs/execute-unified.jsonc` — both segments enabled
- `deploy/compose/docker-compose.unified.yaml` — mounts unified config, provides
  credential env vars for both segments

### Tests

- `s400_segment_router_test.go` — 7 tests: routing, isolation, rejection, counts
- `s400_source_segment_test.go` — 7 tests: mapping, round-trip, unified validation
- `s400_multi_segment_test.go` — 3 tests: structural validation, adapter distinctness

All 17 new tests pass. Full workspace test suite passes (no regressions).

## Evidence Matrix

| Criterion | Evidence | Result |
|-----------|----------|--------|
| Spot+Futures coexist in config | `execute-unified.jsonc` validates | Pass |
| Router dispatches by source | `s400_segment_router_test.go` | Pass |
| Unknown source rejected | `s400_segment_router_test.go` | Pass |
| Source-segment round-trip | `s400_source_segment_test.go` | Pass |
| Merged seed produces multi-source bindings | `seed-configctl.sh --merge` | Pass |
| Execute binary builds with router | `go build ./cmd/execute/` | Pass |
| Full test suite no regressions | `make test` | Pass |
| DryRunSubmitter wraps router | Code inspection `run.go:70-79` | Pass |
| Fail-closed validation unchanged | `s393_segment_enablement_test.go` (26 tests) | Pass |

## What Was NOT Changed

- **configctl internals:** Still one active config per scope. The merge happens
  at the seed script level, not in configctl's activation model.
- **Ingest layer:** The binding watcher and ExchangeScopeActor already support
  multi-source by design (S397). No changes needed.
- **Consumer subjects:** The execute consumer still subscribes to all paper_order
  subjects. Per-segment consumer filtering is deferred.
- **Kill switch / staleness:** Remain binary-wide, not per-segment.

## Acceptance Criteria

| Criterion | Status |
|-----------|--------|
| Spot and Futures can coexist in the same runtime | Met |
| Sequential seed is no longer a structural limitation | Met |
| Runtime projection aligned to unified config | Met |
| Stage prepares leakage hardening and compose proof (S401-S402) | Met |

## Limitations and Non-Goals

1. **No per-segment dry-run** — `dry_run` applies uniformly.
2. **No per-segment kill switch** — binary-wide.
3. **No multi-exchange** — source mapping is Binance-only.
4. **No per-segment consumer isolation** — single consumer, router dispatches.
5. **QueryOrder is sequential** — tries each query port; acceptable for rare path.

## Artifacts

| Type | Path |
|------|------|
| SegmentRouter | `internal/application/execution/segment_router.go` |
| Source-Segment mapping | `internal/shared/settings/schema.go` (lines 283-308) |
| Runtime build | `cmd/execute/run.go` (buildVenueAdapterFromSegments) |
| Seed merge | `scripts/seed-configctl.sh` (--merge flag) |
| Unified config | `deploy/configs/execute-unified.jsonc` |
| Unified compose | `deploy/compose/docker-compose.unified.yaml` |
| Tests (router) | `internal/application/execution/s400_segment_router_test.go` |
| Tests (settings) | `internal/shared/settings/s400_source_segment_test.go` |
| Tests (structural) | `internal/actors/scopes/execute/s400_multi_segment_test.go` |
| Arch doc (merge) | `docs/architecture/binding-merge-and-multi-segment-runtime-projection.md` |
| Arch doc (invariants) | `docs/architecture/multi-segment-runtime-projection-seeds-invariants-and-limitations.md` |

## Next Steps

- **S401:** Leakage hardening — verify that intents cannot cross segment boundaries
  at any layer (ingest, derive, execute, store).
- **S402:** Compose proof with unified config — boot both segments, seed merged
  bindings, verify end-to-end data flow with segment isolation.
