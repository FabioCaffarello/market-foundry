# Smoke CI Shape, PASS/FAIL Contract, And Operational Frictions

## Purpose

This document maps the shape of each smoke target for CI purposes: what it
exercises, its pass/fail contract, time budget, prerequisites, and known
operational frictions. It serves as the operational reference for anyone
diagnosing CI failures or deciding which smokes to integrate next.

## CI-Integrated Smokes

### `smoke-composed` (CI job: Smoke Composed Pipeline)

| Property | Value |
|---|---|
| Canonical target | `make smoke-composed` |
| Script | `scripts/smoke-composed-pipeline.sh` |
| Infrastructure | none (Go tests only) |
| Time budget | ~30s |
| Prerequisites | Go 1.23+ |

**PASS contract:**
- `go vet` on execution and execute actor packages succeeds
- SC-01..SC-07 supervisor composition tests pass
- VP-01..VP-09 venue path verification tests pass
- Error code classification tests pass
- Full execution package regression gate passes

**FAIL signals:**
- Non-zero exit code from any Go test phase
- `[FAIL]` markers in stdout
- Script exits with code 1 and prints `smoke_fail_summary`

**Frictions:** None significant. Pure Go tests, fully deterministic.

### `smoke-analytical` (CI job: Smoke Analytical E2E)

| Property | Value |
|---|---|
| Canonical target | `make smoke-analytical` |
| Script | `scripts/smoke-analytical-e2e.sh` |
| Infrastructure | full compose stack (ClickHouse, NATS, all services) |
| Time budget | ~4-6 min (120s readiness + 120s flush wait + test time) |
| Prerequisites | Docker, compose, Go 1.23+ |

**PASS contract:**
- ClickHouse infrastructure reachable and migrations applied
- Writer service consuming from NATS JetStream and flushing to ClickHouse
- Candle, signal, decision, strategy, risk, and execution history endpoints
  return 200 with valid response structures
- Row counts are positive for all analytical tables

**FAIL signals:**
- Non-zero exit from `ci-wait-ready.sh` (infrastructure not ready)
- Non-zero exit from `smoke-analytical-e2e.sh`
- Error-level log entries in compose logs (warning, not fatal)
- Compose logs artifact uploaded on failure

**Frictions:**
- 120s flush wait is a fixed sleep (not polled); may be too short on slow CI runners
- ClickHouse readiness polling depends on `clickhouse-client` inside the container
- Compose stack startup time varies by CI runner load
- Error log scan is advisory (warns but doesn't fail)

## CI-Integrated Non-Smoke Checks

### `repository-checks` (CI job: Repository Consistency & Quality Gate)

| Property | Value |
|---|---|
| Canonical targets | `make repo-consistency-check` + `make quality-gate-ci` |
| Infrastructure | none (needs Rust toolchain for raccoon-cli) |
| Time budget | ~60-90s (first run with build), ~10-15s (cached) |

**PASS contract:**
- All 21+ repository consistency checks pass (docs, naming, links, alignment)
- Quality gate CI profile passes (architecture boundaries, drift detection)

**FAIL signals:**
- Non-zero exit code
- Specific check failure messages with check name

**Frictions:**
- First CI run must build raccoon-cli from source (~60s)
- Cached builds depend on Cargo.lock + source hash matching

## Local-Only Smokes (Not In CI)

### Why These Remain Local

| Smoke | Reason |
|---|---|
| `smoke` (first-slice) | Requires live Binance WS; non-deterministic timing |
| `smoke-multi` | Same as first-slice, multi-symbol |
| `smoke-round-trip` | Requires live WS + compose stack; combines both frictions |
| `smoke-live-stack` | Requires live WS + compose stack + extended observation |
| `smoke-activation` | Requires live WS + compose stack + activation lifecycle |
| `smoke-operational` | Requires compose stack + process signal manipulation |
| `smoke-restart-recovery` | Requires compose stack + container restart orchestration |
| `smoke-venue-integration` | Requires Binance Futures testnet API credentials |

### Shape Summary For Future CI Consideration

| Smoke | Stack? | Live WS? | Credentials? | Time | CI Feasibility |
|---|---|---|---|---|---|
| `smoke` | yes | yes | no | ~2min | low (live WS) |
| `smoke-multi` | yes | yes | no | ~3min | low (live WS) |
| `smoke-round-trip` | yes | yes | no | ~3min | low (live WS) |
| `smoke-live-stack` | yes | yes | no | ~5min | low (live WS) |
| `smoke-activation` | yes | yes | no | ~5min | low (live WS) |
| `smoke-operational` | yes | no | no | ~2min | medium (compose only) |
| `smoke-restart-recovery` | yes | no | no | ~3min | medium (compose only) |
| `smoke-venue-integration` | no | no | yes | ~1min | low (credentials) |

**Next CI integration candidates** (medium feasibility, no live WS):
- `smoke-operational` — compose-only, tests process signals
- `smoke-restart-recovery` — compose-only, tests container restarts

These could be integrated as optional/nightly CI jobs in future stages.

## Shared Infrastructure

### `scripts/ci-wait-ready.sh`

Reusable readiness polling script used by CI and available for local use:

```bash
./scripts/ci-wait-ready.sh                    # default: 120s timeout
./scripts/ci-wait-ready.sh --timeout 180      # custom timeout
./scripts/ci-wait-ready.sh --skip-clickhouse  # gateway-only check
```

**Contract:**
- Exit 0: all polled services ready
- Exit 1: at least one service timed out
- Structured `[PASS]`/`[FAIL]` output for each service

### `scripts/utils/lib.sh`

All smoke scripts source this shared library for:
- Color-coded `[PASS]`/`[FAIL]`/`[INFO]`/`[WARN]` logging
- `record_fail` + `ERRORS` counter pattern
- `smoke_banner` / `smoke_die_with_hints` / `smoke_fail_summary`
- `http_code`, `json_field`, `json_nested`, `json_has_key` helpers
- `require_commands`, `require_positive_integer` validators

This ensures consistent pass/fail signaling across all smokes.

## Operational Friction Inventory

| Friction | Severity | Mitigation |
|---|---|---|
| 120s flush wait is a fixed sleep | medium | Could be converted to polling; acceptable for now |
| Raccoon-cli must be compiled from source | low | Cargo caching reduces to seconds after first build |
| Live WS smokes cannot run in CI | inherent | Separated into local-only targets |
| Compose startup time varies | low | Readiness polling with timeout handles this |
| Error log scan is advisory only | low | By design — errors may be transient during startup |
| No notification on CI failure | low | GitHub Actions default notifications apply |
