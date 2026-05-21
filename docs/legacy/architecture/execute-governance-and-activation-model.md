# Execute Governance and Activation Model

**Stage**: S83
**Status**: Active

## Purpose

This document defines the governance model for the `execute` binary and its activation gates. It supersedes the partial governance notes in `execute-runtime-and-activation-model.md` (S80) and consolidates all control points into a single authority document.

## Binary Identity

The `execute` binary is the 6th application service in market-foundry:

| Binary | Port | Role |
|--------|------|------|
| configctl | — | Configuration management |
| ingest | — | Observation ingestion |
| derive | :8083 | Evidence → signal → decision → strategy → risk → execution evaluation |
| store | :8081 | Projection materialization and query surface |
| gateway | :8082 | HTTP API gateway |
| execute | :8084 | Venue order submission and fill publishing |

## Governance Layers

### Layer 1: Config-Driven Venue Selection

The venue adapter is selected via `venue.type` in `execute.jsonc`. This replaces the hardcoded adapter from S80.

**Allowed values:**

| Venue Type | Status | Description |
|-----------|--------|-------------|
| `paper_simulator` | ACTIVE | Simulated fills, no exchange contact |
| `venue_market_order` | BLOCKED | Requires full activation gate ceremony |

The schema (`settings.VenueConfig`) validates venue type at config load time. Unknown types are rejected before the binary starts.

**Adding a new venue type requires:**
1. Addition to `knownVenueTypes` in `schema.go`
2. Implementation of `ports.VenuePort` interface
3. Registration in `buildVenueAdapter()` in `cmd/execute/run.go`
4. Full activation gate ceremony (see below)
5. Update to raccoon-cli drift rules

### Layer 2: Pipeline Family Gating

Execution families are opt-in via `pipeline.execution_families`. The config validation enforces:
- Only known families are accepted (`paper_order`, `venue_market_order`)
- Cross-layer dependencies are satisfied (execution → risk → strategy → ...)
- Config symmetry between derive, store, and execute is verified by raccoon-cli

### Layer 3: Runtime Kill Switch

The `EXECUTION_CONTROL` KV bucket stores a global gate:
- `GateActive` — execution proceeds normally
- `GateHalted` — all execution intents are blocked

**Authority model:**
- **Write**: gateway (via `PUT /execution/control`)
- **Read**: execute (VenueAdapterActor checks before each submission)
- **Default**: fail-open (missing gate = active)

### Layer 4: Temporal Guard

The staleness guard rejects intents older than `DefaultStalenessMaxAge` (120s). This prevents stale intents from executing after recovery or restart.

### Layer 5: Domain Validation

Each execution intent is validated through `ExecutionIntent.Validate()` before submission. This catches structural issues without relying on upstream guarantees.

## Activation Gate Ceremony

Before any new venue type can be activated, the following gates must pass:

### Structural Gates (pre-deployment)
- G-S1: Venue adapter implements `VenuePort` interface
- G-S2: Venue type registered in `knownVenueTypes` and `buildVenueAdapter()`
- G-S3: Config file updated with new venue type
- G-S4: Raccoon-cli drift rules updated
- G-S5: Architecture doc updated with new venue type

### Behavioral Gates (pre-activation)
- G-B1: Kill switch halts the new adapter
- G-B2: Staleness guard rejects stale intents
- G-B3: Domain validation rejects invalid intents
- G-B4: Trace propagation (correlation_id, causation_id) flows through
- G-B5: Fill events are published to EXECUTION_FILL_EVENTS

### Operational Gates (pre-production)
- G-O1: Health checks report venue adapter status
- G-O2: Metrics track processed/filled/skipped/errors
- G-O3: Graceful shutdown closes connections cleanly
- G-O4: End-to-end smoke test with multi-symbol pipeline

## What Remains Blocked

| Capability | Gate | Reason |
|-----------|------|--------|
| Real exchange connectivity | Full ceremony | No venue adapter implementation exists |
| Multi-venue routing | G-S1..G-S5 | Only single venue per binary is supported |
| Dynamic venue switching | Not planned | Config is static; restart required for changes |
| Venue credential management | Not planned | No credential infrastructure exists |

## Raccoon-CLI Enforcement

The `drift_detect.rs` analyzer enforces:
- `execute` in `APP_BINARIES` — binary directory must exist
- `EXECUTION_FILL_EVENTS` in `CANONICAL_STREAMS` — fill stream is canonical
- Execute scope actors verified in `check_execution_domain_drift`
- Venue config presence checked in `check_execution_config_drift`
- All 7 adapter files, 13 domain files verified
- Config symmetry across derive/store/execute enforced
