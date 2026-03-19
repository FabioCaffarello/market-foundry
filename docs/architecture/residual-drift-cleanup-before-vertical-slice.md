# Residual Drift Cleanup Before Vertical Slice

## Purpose

Document the residual drift identified and removed during S107, preparing a clean foundation for the next vertical slice.

## Drift Categories

### 1. Compose Infrastructure Drift

| Item | Before | After | Rationale |
|------|--------|-------|-----------|
| ClickHouse container name | `market-raccoon-clickhouse` | `market-foundry-clickhouse` | Legacy naming from raccoon era |
| ClickHouse network | `market-raccoon-network` | `market-foundry-network` | Must join `market-foundry-network` to reach other services |
| configctl healthcheck | Process grep (`/usr/local/bin/service`) | HTTP readiness probe (`/readyz`) | Aligns with all other services; catches actual readiness |
| ClickHouse env_file | `../envs/local.env` (missing dir) | Same path + `local.env.example` template added | Prevents compose failure on fresh clone |

### 2. Test Fixture Naming Drift

| File | Before | After | Rationale |
|------|--------|-------|-----------|
| `configctl_gateway_test.go` | `"server.http"` (7 instances) | `"gateway.http"` | Matches production source identifier after server→gateway rename |
| `envelope_test.go` | `"quality.created"` / `"validator"` | `"config.created"` / `"configctl"` | Removes quality-service era naming |
| `usecases_test.go` | `"Core Quality Config"` | `"Core Market Config"` | Domain-appropriate label |

### 3. Tooling Identity Drift

| File | Before | After | Rationale |
|------|--------|-------|-----------|
| `lsp/client.rs` | `"name": "quality-service"` | `"name": "market-foundry"` | LSP workspace name must match project |
| `smoke/api.rs` | `quality-service API calls` comment | `market-foundry API calls` | Comment alignment |
| `smoke/scenarios.rs` | `quality-service cluster` comment | `market-foundry cluster` | Comment alignment |

### 4. Dead Code Removal

| Module | Size | Status |
|--------|------|--------|
| `results_inspect/` | ~32KB | Removed — deprecated quality-service command, no callers |
| `trace_pack/` | ~52KB | Removed — deprecated quality-service command, no callers |
| `ScenarioSmoke` command + tests | ~5KB | Removed from CLI and integration tests |
| `ResultsInspect` command + tests | ~3KB | Removed from CLI and integration tests |
| `TracePack` command | ~2KB | Removed from CLI |

**Total dead code removed:** ~94KB across Rust source and tests.

**Preserved:** `smoke/` module and `RuntimeSmoke` command — actively used by quality-gate deep profile.

### 5. Documentation Drift

| File | Before | After | Rationale |
|------|--------|-------|-----------|
| `AGENTS.md` | "Pre-absorption phase" | "Post first-slice phase" | Matches actual repository status |

## What Was Not Changed

- **smoke module internals**: Still contains some quality-service references in runtime code paths (error messages, API client). These are functional code used by the quality-gate deep profile and will be cleaned when the smoke system is redesigned.
- **Architecture docs referencing marketmonkey**: These are forward-looking planning docs and remain valid.
- **Topology compose test fixtures**: The `quality-service/consumer:dev` image reference in test data is a parse-validation fixture, not an operational reference.
