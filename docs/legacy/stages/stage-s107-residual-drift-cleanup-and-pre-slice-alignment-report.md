# Stage S107 — Residual Drift Cleanup and Pre-Slice Alignment

**Status:** Complete
**Objective:** Remove cheap residual drift and align the repository for the next vertical slice.

## Executive Summary

S107 performed a targeted cleanup of residual naming, infrastructure, and tooling drift inherited from the quality-service era and the server→gateway rename. No new features were introduced. No architectural reorganization was performed. The repository is now cleaner, more consistent, and better prepared for the next vertical slice.

## Drift Removed

### Compose Infrastructure (3 fixes)
- **ClickHouse container name**: `market-raccoon-clickhouse` → `market-foundry-clickhouse`
- **ClickHouse network**: `market-raccoon-network` → `market-foundry-network` (was disconnected from service mesh)
- **configctl healthcheck**: Process grep → HTTP readiness probe (aligns with all other services)

### Missing Infrastructure (1 fix)
- **ClickHouse env template**: Created `deploy/envs/local.env.example` — compose referenced `../envs/local.env` but the directory didn't exist

### Test Fixture Naming (3 files, 10 edits)
- `configctl_gateway_test.go`: `"server.http"` → `"gateway.http"` (7 instances)
- `envelope_test.go`: `"quality.created"` / `"validator"` → `"config.created"` / `"configctl"`
- `usecases_test.go`: `"Core Quality Config"` → `"Core Market Config"`

### Tooling Identity (3 fixes)
- LSP client workspace name: `"quality-service"` → `"market-foundry"`
- Smoke module doc comments: updated to `market-foundry`

### Dead Code Removal (~94KB)
- Deleted `tools/raccoon-cli/src/results_inspect/` (~32KB) — deprecated, no callers
- Deleted `tools/raccoon-cli/src/trace_pack/` (~52KB) — deprecated, no callers
- Removed `ScenarioSmoke`, `ResultsInspect`, `TracePack` CLI commands and handler code
- Removed corresponding integration tests (~170 lines)
- **Preserved**: `smoke/` module and `RuntimeSmoke` command (used by quality-gate deep profile)

### Documentation (1 fix)
- `AGENTS.md` status: `"Pre-absorption phase"` → `"Post first-slice phase"`

## Files Changed

| File | Change Type |
|------|-------------|
| `deploy/compose/docker-compose.yaml` | Fix container name, network, healthcheck |
| `deploy/envs/local.env.example` | New — env template for ClickHouse |
| `internal/adapters/nats/configctl_gateway_test.go` | Fix `server.http` → `gateway.http` |
| `internal/shared/envelope/envelope_test.go` | Fix quality-service era naming |
| `internal/application/configctl/usecases_test.go` | Fix test fixture label |
| `tools/raccoon-cli/src/main.rs` | Remove deprecated commands and modules |
| `tools/raccoon-cli/src/lsp/client.rs` | Fix workspace name |
| `tools/raccoon-cli/src/smoke/api.rs` | Fix doc comment |
| `tools/raccoon-cli/src/smoke/scenarios.rs` | Fix doc comment |
| `tools/raccoon-cli/tests/cli_integration.rs` | Remove deprecated command tests |
| `AGENTS.md` | Update status |
| `docs/architecture/residual-drift-cleanup-before-vertical-slice.md` | New |
| `docs/architecture/pre-slice-repository-alignment.md` | New |

## Structural Gains

1. **Compose consistency**: All services now use the same naming scheme and healthcheck pattern. ClickHouse joins the correct network.
2. **Test fidelity**: Test fixtures reflect actual production identifiers, reducing confusion during debugging.
3. **Tooling hygiene**: ~94KB of dead code removed from raccoon-cli. CLI surface reduced to active commands only.
4. **Identity alignment**: No active code path references `quality-service` or `server.http`.

## Limits Maintained

- No new features introduced
- No architectural reorganization
- No cosmetic-only changes (every edit has structural rationale)
- No wide horizontal refactoring
- smoke module preserved (functional dependency from quality-gate)

## Preparation for S108

The repository is now ready for the next phase. Recommended focus areas:

1. **Vertical slice design**: Define the scope and contracts for the next end-to-end slice
2. **Smoke module evolution**: Decide whether to keep the runtime-smoke gate step or replace it with `make smoke` integration
3. **Raccoon-cli warning cleanup**: Optional — suppress unused-code warnings accumulated from removed callers
