# Evidence-Driven Refactors After Vertical Slice 01

> Stage S111 — Targeted refactors justified by S110 friction capture.
> Date: 2026-03-19

---

## Selection Criteria

Each refactor entered S111 only if it met all three conditions:

1. **Evidence-based** — directly traced to a friction finding from S110 operational validation.
2. **High value/cost ratio** — reduces recurring maintenance cost or eliminates a class of silent bugs.
3. **Bounded scope** — the change is self-contained, does not trigger horizontal rewrites, and preserves existing behavior.

---

## Refactor R1: Signal Publisher Missing correlation_id (F06 partial)

**S110 Finding:** F06 (P2) — Signal publisher actor omits `correlation_id` from error logs while all other domain publishers include it.

**Evidence:** Direct code comparison of 4 publisher actors in `internal/actors/scopes/derive/`. The `decision_publisher_actor.go`, `risk_publisher_actor.go`, and `strategy_publisher_actor.go` all log `correlation_id` on publish failure. The `signal_publisher_actor.go` does not.

**Change:** Added `"correlation_id", msg.Event.Metadata.CorrelationID` to the error log in `signal_publisher_actor.go`.

**Value:** Restores observability parity across all domain publishers. Without this field, tracing a failed signal publish back to its originating candle event requires searching by type/source/symbol/timeframe instead of a single correlation ID.

**Files changed:** 1
- `internal/actors/scopes/derive/signal_publisher_actor.go`

---

## Refactor R2: Projection Actor Stats Normalization (F05)

**S110 Finding:** F05 (P1) — 7 projection actors follow the same flow but diverge silently in stats tracking. Only 2 of 7 (risk, strategy) had a `received` counter and `checkStatsInvariant()`. Only 3 of 7 (decision, risk, strategy) tracked `received` at all. Logger initialization style varied (some include family/bucket, others don't).

**Evidence:** Line-by-line comparison during S110 structural review. The table in Finding F05 shows exact inconsistencies per actor.

**Change:** Added `received` counter and `checkStatsInvariant()` to the 5 actors that were missing them:
- `candle_projection_actor.go` — added `received` to stats, `checkStatsInvariant()`, updated `logStats()`
- `signal_projection_actor.go` — same
- `volume_projection_actor.go` — same
- `trade_burst_projection_actor.go` — same
- `decision_projection_actor.go` — had `received` already, added `checkStatsInvariant()`

**Value:** All 7 projection actors now:
- Count every received message via `received` counter
- Validate on shutdown that `received == sum(materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors)`
- Log `received` in stats output

This means any future message-loss bug will be detected at shutdown time rather than requiring manual audit. The invariant check is the projection equivalent of a double-entry bookkeeping assertion.

**Files changed:** 5
- `internal/actors/scopes/store/candle_projection_actor.go`
- `internal/actors/scopes/store/decision_projection_actor.go`
- `internal/actors/scopes/store/signal_projection_actor.go`
- `internal/actors/scopes/store/trade_burst_projection_actor.go`
- `internal/actors/scopes/store/volume_projection_actor.go`

**Not changed (already correct):**
- `internal/actors/scopes/store/risk_projection_actor.go`
- `internal/actors/scopes/store/strategy_projection_actor.go`

---

## Refactor R3: Raccoon-CLI Dead Code Cleanup (F03)

**S110 Finding:** F03 (P2) — 26 compiler warnings for dead code across the raccoon-cli crate. Unused functions, imports, and fields mask real warnings during development.

**Evidence:** `cargo check` output showing 26 warnings. The smoke module became entirely dead after S109 removed the `ScenarioSmoke` CLI command.

**Changes:**
1. Added `#[allow(dead_code)]` to `mod smoke` — the smoke module is designed for live-infrastructure operational testing and will be rewired when the runtime E2E surface matures. Deleting it would lose working scenario implementations.
2. Removed unused re-exports from `codeintel/mod.rs` — 8 type re-exports were unused after S109 refactoring.
3. Prefixed unused variables with `_` in `contracts.rs` — `expected_command` and `expected_query` variables used in a heuristic that isn't yet active.
4. Removed dead utility functions from `coverage_map.rs` — `relevant_checks_for_path()`, `tdd_guidance()`, and `TddGuidance` struct were never called from any code path. Their tests were also removed.
5. Added targeted `#[allow(dead_code)]` to API-surface code in `codeintel/index.rs`, `lsp/client.rs`, `lsp/protocol.rs`, `lsp/types.rs`, and `runtime_bindings/configs.rs` — these are query/enrichment APIs that will be consumed by future analyzers.

**Value:** `cargo check` now produces zero warnings. Real issues will no longer be buried in noise.

**Files changed:** 8
- `tools/raccoon-cli/src/main.rs`
- `tools/raccoon-cli/src/codeintel/mod.rs`
- `tools/raccoon-cli/src/codeintel/index.rs`
- `tools/raccoon-cli/src/analyzers/contracts.rs`
- `tools/raccoon-cli/src/analyzers/coverage_map.rs`
- `tools/raccoon-cli/src/analyzers/runtime_bindings/configs.rs`
- `tools/raccoon-cli/src/lsp/client.rs`
- `tools/raccoon-cli/src/lsp/protocol.rs`
- `tools/raccoon-cli/src/lsp/types.rs`

---

## Refactor R4: Generic UseCase Factory for ConfigctlClient (F04)

**S110 Finding:** F04 (P1) — 10 configctlclient files, each ~30 lines of identical boilerplate (struct → constructor → Execute with nil check → normalize → validate → delegate). Adding a new configctl operation requires copy-pasting an entire file.

**Evidence:** Code review of `internal/application/configctlclient/` showing 10 files with identical structure. The only differences are type names and the gateway method called.

**Changes:**
1. Created `internal/shared/usecase/usecase.go` with two generic types:
   - `CommandUseCase[Cmd, Reply]` — wraps a gateway function with nil safety, `Normalize()`, and `Validate()`. For use with command/query types that implement the `Normalizable` interface.
   - `GatewayUseCase[In, Out]` — wraps a gateway function with nil safety only. For simple delegation without input processing.

2. Replaced 10 configctlclient files. Each file went from ~30 lines (struct + constructor + Execute method) to ~16 lines (type alias + constructor). The constructor names are unchanged — `compose.go` and all consumers required zero modifications.

   - 7 files use `CommandUseCase` (activate, compile, create_draft, get_config, list_active_runtime_projections, validate_config, validate_draft)
   - 3 files use `GatewayUseCase` (get_active_config, list_configs, list_active_ingestion_bindings)

**Value:**
- Eliminated ~150 lines of pure boilerplate (10 files × ~15 lines reduced).
- Adding a new configctl operation is now 5 lines instead of 30.
- The nil-check, normalize, and validate chain is defined once and tested in one place.
- Type aliases preserve backward compatibility — existing constructor names and `Execute()` signatures are unchanged.
- The generic is reusable for future client packages that follow the same pattern.

**Design decision:** The 3 "just delegate" files use `GatewayUseCase` instead of `CommandUseCase` to preserve their original behavior. Although `ListActiveIngestionBindingsQuery` implements `Normalizable`, the original code chose not to call it — the refactor preserves that choice.

**Files changed:** 11
- `internal/shared/usecase/usecase.go` — **NEW**
- `internal/application/configctlclient/activate_config.go`
- `internal/application/configctlclient/compile_config.go`
- `internal/application/configctlclient/create_draft.go`
- `internal/application/configctlclient/get_active_config.go`
- `internal/application/configctlclient/get_config.go`
- `internal/application/configctlclient/list_active_ingestion_bindings.go`
- `internal/application/configctlclient/list_active_runtime_projections.go`
- `internal/application/configctlclient/list_configs.go`
- `internal/application/configctlclient/validate_config.go`
- `internal/application/configctlclient/validate_draft.go`

**Not changed:**
- `internal/application/configctlclient/usecases_test.go` — tests pass without modification
- `cmd/gateway/compose.go` — wiring unchanged
- `internal/interfaces/http/routes/core.go` — interfaces unchanged
