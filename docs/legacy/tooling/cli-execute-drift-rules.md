# CLI Execute Drift Rules

> Stage S83 — Drift detection rules for the execute binary and its governance.

## Overview

These rules extend the execution domain drift rules (ED-1 through ED-6) with execute-binary-specific checks introduced in S83. The execute binary was added to `APP_BINARIES` and its artifacts are now fully tracked.

## Changes from S83

### Constants Updated

| Constant | Before (S80) | After (S83) |
|----------|-------------|-------------|
| `APP_BINARIES` | 5 (configctl, gateway, ingest, derive, store) | 6 (+execute) |
| `CANONICAL_STREAMS` | 8 | 9 (+EXECUTION_FILL_EVENTS) |
| `EXECUTION_DOCS` | 7 | 11 (+execute-runtime-and-activation-model, +execute-governance-and-activation-model, +execution-family-separation-after-paper-step, +venue-routing-and-ownership-split) |
| `EXECUTION_EXPECTED_SUBJECTS` | 2 | 6 (+fill, +status, +control.get, +control.set) |
| `EXECUTION_EXPECTED_DURABLES` | 1 | 3 (+execute-venue-market-order-intake, +store-execution-venue-market-order-fill) |
| `EXECUTION_EXPECTED_BUCKETS` | 1 | 3 (+EXECUTION_VENUE_MARKET_ORDER_LATEST, +EXECUTION_CONTROL) |
| `EXECUTION_ADAPTER_FILES` | 5 | 7 (+execution_control_gateway.go, +execution_control_kv_store.go) |
| `EXECUTION_DOMAIN_FILES` | 6 | 13 (+control.go, +paper_venue_adapter.go, +staleness_guard.go, +control_contracts.go, +get_execution_status.go, +get_execution_control.go, +venue.go) |

### Dead Code Removed

- `check_execution_premature_implementation` — removed. This function was the S70 pre-implementation guard. It blocked execution artifacts before S75. Since execution is now fully implemented (S71-S82), this check was dead code producing false positives.

### Checks Enhanced

- `check_execution_domain_drift` — now includes execute scope actors (execute_supervisor.go, venue_adapter_actor.go) in addition to derive/store actors
- `check_execution_config_drift` — now verifies execute.jsonc exists and declares execution_families, plus checks for venue config presence

## Active Rule Catalog

### ED-1: execution-docs-drift
**Phase**: Active (S70, updated S83)
**Severity**: Error
**What**: Verifies all 11 execution architecture documents exist.
**Files checked**:
- `execution-domain-design.md`
- `execution-stream-families.md`
- `execution-activation-and-ownership.md`
- `execution-query-surface-guidelines.md`
- `execution-readiness-review.md`
- `execution-entry-prerequisites.md`
- `execution-risks-and-blockers.md`
- `execute-runtime-and-activation-model.md` (added S83)
- `execute-governance-and-activation-model.md` (added S83)

### ED-2: execution-premature-implementation
**Phase**: REMOVED (S83)
**Reason**: Execution is fully implemented. The premature guard is no longer needed.

### ED-3: execution-adapter-drift
**Phase**: Active (S71, updated S83)
**Severity**: Error
**What**: Verifies 7 NATS adapter files exist for the execution domain.
**Files**: execution_registry.go, execution_publisher.go, execution_consumer.go, execution_gateway.go, execution_kv_store.go, execution_control_gateway.go, execution_control_kv_store.go

### ED-4: execution-domain-drift
**Phase**: Active (S71, updated S83)
**Severity**: Error
**What**: Verifies 13 domain/application files, 6 actor files (derive+store+execute), and 2 HTTP files exist.
**Actor scopes**: derive (evaluator, publisher), store (consumer, projection), execute (supervisor, venue_adapter)

### ED-5: execution-config-drift
**Phase**: Active (S71, updated S83)
**Severity**: Error/Warning
**What**: Verifies symmetric execution_families across derive.jsonc, store.jsonc, and execute.jsonc. Also checks venue config presence in execute.jsonc.

### ED-6: execution-contracts-drift
**Phase**: Active (S71, updated S83)
**Severity**: Error
**What**: Verifies 6 subjects, 3 durable consumers, and 3 KV bucket names exist in Go source.

## Binary Drift

With `execute` added to `APP_BINARIES`, the existing binary drift checks now also verify:
- `cmd/execute/` directory exists
- Compose service `execute` exists (or warning if absent)

## Venue Config Drift

New in S83: `check_execution_config_drift` now emits:
- `execution-venue-config-present` (info) — venue config found in execute.jsonc
- `execution-venue-config-absent` (warning) — venue config missing, falls back to paper_simulator
