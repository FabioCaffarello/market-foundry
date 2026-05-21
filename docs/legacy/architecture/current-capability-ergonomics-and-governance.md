# Current Capability Ergonomics and Governance

> **Stage:** S141 — Ergonomics and Governance Consolidation
> **Scope:** Document and codify the ergonomic surface and governance rules for the current capabilities.

---

## 1. Purpose

This document defines the ergonomic standards and governance rules for operating the current Market Foundry capabilities. It is the authoritative reference for how configuration, tooling, diagnostics, and operational governance work today — after the S141 consolidation pass.

---

## 2. Configuration Ergonomics

### 2.1. Config File Layout

All service configs live in `deploy/configs/<service>.jsonc`. The format is JSONC (JSON with comments), parsed by the settings package with strict unknown-field rejection.

**Canonical reference:** `deploy/configs/CONFIG-REFERENCE.md` documents every field, its type, default, valid range, and constraints.

### 2.2. Validation Chain

Configuration is validated at multiple levels:

| Level | When | What | Where |
|-------|------|------|-------|
| **Schema** | Service startup | Field types, required fields, unknown field rejection | `internal/shared/settings/Load()` |
| **Range** | Service startup | Duration ranges, timeframe bounds, family name lookup | `internal/shared/settings/schema.go` |
| **Dependency** | Service startup | Cross-layer family dependencies (signal→evidence, etc.) | `PipelineConfig.ValidatePipeline()` |
| **Duplicate** | Service startup | No duplicate families or timeframes | `rejectDuplicates()`, `ValidateTimeframes()` |
| **Venue** | Service startup | Venue type, staleness bounds, timeout bounds | `VenueConfig.Validate()` |

Invalid config causes startup failure with structured error output (RFC 7807 problem format).

### 2.3. Family Registry

The canonical family registry is in `internal/shared/settings/schema.go`. The exported API:

- `KnownFamilies(domain)` — returns all registered family names for a domain
- `IsKnownFamily(domain, family)` — checks if a family is registered
- `DependencyGraph()` — returns the full cross-layer dependency map

Adding a new family requires:
1. Add to the appropriate `known*Families` map in `schema.go`
2. Add dependency entry if it has upstream requirements
3. Update config reference (`CONFIG-REFERENCE.md`)
4. Update relevant JSONC configs
5. Run `make verify` to confirm tests pass

### 2.4. Config Consistency Rule

Store must project every family that derive produces. The config for `store.jsonc` should be a superset of `derive.jsonc` family lists. Execute declares its full dependency chain for validation. This is enforced by `ValidatePipeline()` at startup.

---

## 3. Script Ergonomics

### 3.1. Shared Library

All scripts source `scripts/utils/lib.sh` for:

- **Color output** — `pass()`, `fail()`, `info()`, `warn()`, `phase()`
- **Error tracking** — `ERRORS` counter, `record_fail()`
- **Service constants** — `SVC_PORTS` map, `ALL_SERVICES`, `PIPELINE_SERVICES`
- **Timeouts** — `HEALTH_WAIT_MAX`, `CANDLE_WAIT_MAX` (environment-overridable)
- **JSON helpers** — `json_field`, `json_nested`, `json_has_key`

### 3.2. Makefile Targets

The Makefile is the primary entry point for all operations:

| Category | Targets | Purpose |
|----------|---------|---------|
| **Build** | `tidy`, `test`, `build`, `docker-build` | Standard Go workflow |
| **Stack** | `up`, `down`, `restart`, `logs`, `ps` | Docker Compose lifecycle |
| **Pipeline** | `live`, `live-check`, `live-multi`, `live-multi-check` | Full pipeline orchestration |
| **Smoke** | `smoke`, `smoke-multi` | E2E validation |
| **Diagnostics** | `diag` | Lightweight diagnostic snapshot |
| **Seed** | `seed`, `seed-multi` | Configctl seeding |
| **Quality** | `check`, `verify`, `check-deep` | Pre/post-change validation |
| **Raccoon** | `quality-gate`, `arch-guard`, `drift-detect`, etc. | Architecture governance |

### 3.3. Diagnostic Quick Path

```bash
make diag              # snapshot of all service health, phases, trackers
make live-check        # full validation of running stack (single-symbol)
make live-multi-check  # full validation of running stack (multi-symbol)
```

---

## 4. Governance Rules

### 4.1. Family Addition Governance

| Step | Action | Enforced by |
|------|--------|-------------|
| 1 | Register family name in `schema.go` | Compile time |
| 2 | Declare dependencies in `schema.go` | `ValidatePipeline()` |
| 3 | Add to relevant config files | Startup validation |
| 4 | Update `CONFIG-REFERENCE.md` | Code review |
| 5 | Pass `make verify` | CI / pre-commit |

### 4.2. Venue Addition Governance

Venue types require an activation gate ceremony:

| Step | Action |
|------|--------|
| 1 | Governance review (risk assessment for real capital) |
| 2 | Adapter implementation in `internal/adapters/` |
| 3 | Integration test with embedded venue mock |
| 4 | Update `knownVenueTypes` in `schema.go` |
| 5 | Update `execute.jsonc` config reference |
| 6 | Document ceremony outcome |

### 4.3. Timeframe Addition Governance

Adding a timeframe has no code change — just config update. But operational implications must be considered:

| Timeframe | Candle finalization | RSI warm-up (15 periods) |
|-----------|--------------------|-----------------------------|
| 60s       | 1 minute           | 15 minutes                  |
| 300s      | 5 minutes          | 1.25 hours                  |
| 900s      | 15 minutes         | 3.75 hours                  |
| 3600s     | 1 hour             | 15 hours                    |
| 14400s    | 4 hours            | 60 hours (2.5 days)         |
| 86400s    | 24 hours           | 15 days                     |

Longer timeframes increase cold-start warm-up time proportionally. This is a physics constraint, not a bug.

### 4.4. Config Change Governance

Config changes do not require code changes. The governance path:

1. Edit the relevant `deploy/configs/*.jsonc` file
2. Run `make compose-config` to validate compose syntax
3. Restart the affected service (`make restart SERVICE=<name>`)
4. Run `make diag` to verify health
5. Run `make smoke` to verify query surface

---

## 5. Diagnostic Endpoints Summary

### /statusz Phase Semantics

| Phase | Meaning | Action |
|-------|---------|--------|
| `starting` | < 30s uptime, no events yet | Wait |
| `warming` | Some trackers awaiting first event | Wait (can take minutes for longer TFs) |
| `active` | All trackers receiving events | Normal operation |
| `idle` | At least one tracker idle > 2min | Investigate — may be normal for long TFs |
| `stalled` | All trackers idle > 2min | Alert — pipeline may be disconnected |

### Idle Threshold

Default: 2 minutes. Configurable via `WithIdleThreshold()` in the health server. Long timeframes (900s, 3600s) will naturally show idle between candle windows — this is expected.

---

## 6. Query Surface Ergonomics

### URL Pattern

```
GET /<domain>/<family>/latest?source=<src>&symbol=<sym>&timeframe=<tf>
GET /<domain>/<family>/history?source=<src>&symbol=<sym>&timeframe=<tf>&limit=N
GET /<domain>/<family>/history?source=<src>&symbol=<sym>&timeframe=<tf>&since=X&until=Y
```

### Required Parameters

| Parameter   | Type   | Example      | Notes |
|-------------|--------|--------------|-------|
| `source`    | string | `binancef`   | Exchange adapter ID |
| `symbol`    | string | `btcusdt`    | Lowercase, no separator |
| `timeframe` | int    | `60`         | Seconds; must match a configured timeframe |

### Error Responses

| Condition | HTTP Status | Problem Code |
|-----------|-------------|--------------|
| Missing required param | 400 | `missing_parameter` |
| Unknown family | 404 | `not_found` |
| No data yet | 200 | Response with `null` value |

---

## 7. Limits and Accepted Constraints

These are known limitations that are accepted for the current baseline:

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| In-memory sampler state | Crash loses up to 1 window per TF | Self-heals on next window |
| RSI cold-start warm-up | 15 periods × TF duration | Algorithmic constraint |
| No interim candle snapshots | Partial candles not observable | By design |
| Global timeframe list | All symbols get same TFs | By design |
| NATS 72h retention | History limited to 3 days | ClickHouse future work |
