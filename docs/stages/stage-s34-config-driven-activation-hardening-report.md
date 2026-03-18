# Stage S34 — Config-Driven Activation Hardening

> **Status:** Complete
> **Date:** 2026-03-17
> **Objective:** Harden config-driven activation so the next domain layer can enter without hardcodes or implicit activation.
> **Scope:** Activation model hardening — no new domain, no signal implementation.

---

## Executive Summary

S34 introduced `pipeline.families` — a config-driven family activation mechanism for both derive and store. Previously, all evidence families were hardcoded to always spawn at startup. Now each binary filters its available families against the configured list, enabling selective activation, staged rollout, and a clear path for new domains to enter without implicit always-on behavior. The system remains fully backward compatible.

---

## Activation Points Hardened

### 1. Store supervisor — from always-on to config-filtered

**Before:** `store_supervisor.go` registered 3 `ProjectionPipeline` entries and spawned all unconditionally in `start()`. Health trackers in `cmd/store/run.go` were hardcoded for all 3 families.

**After:** All pipelines are defined in `allPipelines`, then filtered by `cfg.Pipeline.IsFamilyEnabled()`. Only matching pipelines are spawned. Health trackers are built dynamically — only trackers for enabled families are created. If no families match, the binary fails with a clear error.

**Files:** `internal/actors/scopes/store/store_supervisor.go`, `cmd/store/run.go`

### 2. Derive supervisor — from always-on to config-filtered

**Before:** `derive_supervisor.go` registered 3 `FamilyProcessor` entries and passed all to source scopes. Every source/symbol activation spawned samplers for all 3 families.

**After:** All processors are defined in `allProcessors`, then filtered by `cfg.Pipeline.IsFamilyEnabled()`. Only matching processors propagate to source scopes. If derive is configured with `"families": ["candle"]`, a symbol activation only spawns candle samplers — no tradeburst or volume samplers.

**File:** `internal/actors/scopes/derive/derive_supervisor.go`

### 3. Settings schema — new `pipeline.families` field

**Before:** `PipelineConfig` had only `Timeframes []int`.

**After:** Added `Families []string` with helper methods:
- `IsFamilyEnabled(family) bool` — true if in list or list empty
- `EnabledFamilies() []string` — returns list or nil

Fully backward compatible — empty or absent field means all families enabled.

**Files:** `internal/shared/settings/schema.go`, `internal/shared/settings/settings_test.go`

### 4. Deploy configs — families declared explicitly

**Before:** `deploy/configs/store.jsonc` had no pipeline section. `deploy/configs/derive.jsonc` had only timeframes.

**After:** Both configs now declare `pipeline.families: ["candle", "tradeburst", "volume"]`. This makes the activation explicit and auditable.

**Files:** `deploy/configs/store.jsonc`, `deploy/configs/derive.jsonc`

### 5. Startup logging — activation mode visible

Both supervisors now log their activation mode at startup:
- `"activation": "config-driven"` — when `pipeline.families` is set
- `"activation": "all (no pipeline.families configured)"` — when field is absent

This makes it immediately visible in logs whether config-driven activation is active.

---

## Files Changed

### Go source (4 files)
| File | Change |
|------|--------|
| `internal/shared/settings/schema.go` | Added `Families []string` to `PipelineConfig`, `IsFamilyEnabled()`, `EnabledFamilies()` |
| `internal/shared/settings/settings_test.go` | Added 4 tests for family config methods |
| `internal/actors/scopes/store/store_supervisor.go` | Pipeline filtering by config, fail-safe on empty, activation mode logging |
| `internal/actors/scopes/derive/derive_supervisor.go` | Processor filtering by config, fail-safe on empty, activation mode logging |

### Entry points (1 file)
| File | Change |
|------|--------|
| `cmd/store/run.go` | Dynamic tracker creation based on enabled families |

### Config (2 files)
| File | Change |
|------|--------|
| `deploy/configs/store.jsonc` | Added `pipeline.families` |
| `deploy/configs/derive.jsonc` | Added `pipeline.families` |

### Docs (3 files)
| File | Change |
|------|--------|
| `docs/architecture/config-driven-activation-hardening.md` | New — canonical activation model doc |
| `docs/architecture/governance-hygiene-status.md` | Updated config-driven activation status to MET |
| `docs/stages/stage-s34-config-driven-activation-hardening-report.md` | This report |

---

## Test Results

| Suite | Result |
|-------|--------|
| `internal/shared/settings` | All pass (8 tests including 4 new) |
| `internal/application/derive` | All pass |
| `internal/application/ingest` | All pass |
| `internal/domain/evidence` | All pass |
| `internal/adapters/nats` | All pass |
| `go build ./cmd/store` | OK |
| `go build ./cmd/derive` | OK |
| `go build ./cmd/ingest` | OK |

---

## Activation Model Summary

| Layer | Control | Mechanism | Dynamic? |
|-------|---------|-----------|----------|
| **Family** | Which evidence types run | `pipeline.families` config | No (restart required) |
| **Binding** | Which sources/symbols run | Configctl BindingWatcherActor | Yes (live events) |
| **Timeframe** | Sampling granularity | `pipeline.timeframes` config | No (restart required) |

### Two-layer interaction example

Config: `families: ["candle"]`, `timeframes: [60, 300]`
Binding: `binancef.btcusdt` activated via configctl

Result:
- Derive spawns SourceScopeActor for `binancef`
- Only candle samplers created: `sampler-btcusdt-60s`, `sampler-btcusdt-300s`
- No tradeburst or volume samplers (not in families)
- Store spawns only candle pipeline (candle-projection + candle-consumer)
- No tradeburst or volume projections

---

## Remaining Limitations

| # | Limitation | Severity | Notes |
|---|-----------|----------|-------|
| 1 | Binding deactivation incomplete | MEDIUM | Cleared events logged only; requires process restart |
| 2 | Family changes require restart | LOW | By design; config-driven, not runtime-dynamic |
| 3 | QueryResponderActor not family-filtered | LOW | Still opens KV stores for all 3 types; improvement for later |
| 4 | No config validation for unknown family names | LOW | Unknown families silently skipped; could add a warning |

---

## Signal Entry Readiness (Updated)

| Prerequisite | Status |
|-------------|--------|
| 3+ evidence types proven | MET |
| FamilyProcessor pattern validated | MET |
| ProjectionPipeline pattern validated | MET |
| Actor ownership docs current | MET (S33) |
| raccoon-cli topology rules current | MET (S33) |
| **Config-driven activation proven** | **MET (S34)** |
| Architecture approval for signal domain | NOT MET |

**Signal entry now blocked only by architecture approval.** The activation mechanism is proven — signal families can enter via `pipeline.families` without any activation model changes.

---

## Preparation for S35

S34 unblocks the following S35 candidates:

1. **evidence.stats (new evidence family)** — All prerequisites met. Add sampler/projection, register in allProcessors/allPipelines, add to config families.
2. **Signal architecture design doc** — The last remaining prerequisite for signal entry. Would define boundaries, activation, and contracts.
3. **Binding deactivation** — Wire scope→binding tracking for clean runtime deactivation without restart.

---

## Guard Rails Compliance

| Guard Rail | Complied? |
|-----------|----------|
| No signal implementation | Yes |
| No multiple new domains | Yes — only activation hardening |
| No generic activation framework | Yes — simple config filter, no abstraction layers |
| No hardcodes reintroduced | Yes — existing hardcodes replaced with config-driven filtering |
| Remaining limits documented | Yes — 4 limitations listed above |
