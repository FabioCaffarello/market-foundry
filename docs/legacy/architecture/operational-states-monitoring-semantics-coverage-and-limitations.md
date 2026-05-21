# Operational States: Monitoring Semantics, Coverage, and Limitations

**Stage**: S486
**Date**: 2026-03-26
**Status**: Active

---

## Purpose

This document maps which operational states are now explicitly monitorable, which remain implicit or dispersed, and what limitations apply. It serves as the source of truth for what the S486 monitoring layer covers and what it intentionally excludes.

---

## Coverage Matrix

### States Exposed via `GET /monitoring/state`

| State | Source | Coverage | Notes |
|---|---|---|---|
| Current session ID and status | Session KV via store gateway | Direct | Most recent session (open, closed, or halted) |
| Session operator | Session KV | Direct | Who started the session |
| Session config (venue_type, dry_run, segments) | Session KV | Direct | Value snapshot from session start |
| Session duration | Derived from started_at/closed_at | Computed | Only for terminal sessions |
| Per-segment counters | Session KV (populated at close) | Direct | Processed, filled, rejected, errors per segment |
| Gate status (active/halted) | Execution control KV | Direct | Current kill-switch state |
| Gate halt reason | Execution control KV | Direct | Why execution was halted |
| Surface availability | Gateway composition | Static | Which endpoint families are wired |

### States Queryable via Existing Surfaces (Not Duplicated)

| State | Surface | Why not duplicated |
|---|---|---|
| Runtime phase (starting/warming/active/idle/stalled/degraded) | `/statusz` on each binary | Per-binary concern; consolidating across binaries requires multi-host monitoring outside scope |
| Activation surface (adapter/gate/credentials/effective mode) | `GET /execution/control` | Full detail already queryable; monitoring surface only needs gate status |
| Execution lifecycle entries | `GET /execution/lifecycle/list` | Operational list already exists with filters |
| PO verification results | `GET /session/:id/verify` | Computed on demand; not a persistent state to monitor |
| Effectiveness summary | `GET /analytical/composite/decision/effectiveness/summary` | ClickHouse-backed; different latency profile |
| Pairing summary | `GET /analytical/composite/pairing` | ClickHouse-backed; different latency profile |
| Decision lineage | `GET /analytical/composite/chain` | Per-chain detail; not operational state |
| Consistency findings | Part of DecisionReviewBundle | Per-chain detail; not operational state |

### States That Remain Implicit

| State | Current location | Why implicit | Future path |
|---|---|---|---|
| Cross-binary pipeline health | Individual `/statusz` per binary | No central aggregator. Each binary reports independently. | S487+ could add batch triage surfaces |
| ClickHouse write lag | Not measured | Writer publishes, no feedback loop on CH commit latency | Would require writer→gateway signaling |
| NATS consumer lag (operational) | Prometheus `marketfoundry_consumer_lag_messages` | Metric exists but not surfaced in monitoring endpoint | Could be added if operator need emerges |
| Position tracking / open P&L | Not implemented | Non-goal of current system | Requires portfolio-level model |
| Multi-session effectiveness trends | Not aggregated | Effectiveness is per-session or per-chain | S487 batch review may expose this |

---

## Semantics

### Session Summary Semantics

- The monitoring endpoint returns the **most recent session** from the session list (newest-first ordering).
- When a session is `open`, counters may be **stale** — they are only finalized at session close.
- `duration` is only present for terminal sessions (closed or halted).
- If no sessions have ever been created, `session` is null.

### Gate Summary Semantics

- Gate status is read from the execution control KV store at query time.
- It reflects the **current** gate state, not the gate state at session start (that is captured in `session.activation` on the full session entity).
- `updated_at` reflects when the gate was last changed, not when it was read.

### Surface Availability Semantics

- Captured **once** at gateway startup, not re-probed at query time.
- A surface marked `true` means the gateway was able to create a connection to the underlying gateway or data source at composition time.
- It does **not** guarantee that the surface is currently responsive — a NATS connection drop after startup would not update this value.
- A surface marked `false` means endpoints in that family return 503 or are not registered.

---

## Limitations

1. **Static surface availability**: Surface availability does not update after gateway startup. A runtime NATS disconnect would not be reflected until gateway restart.

2. **Session counters lag**: Per-segment counters are populated at session close. For open sessions, counters may be zero or stale.

3. **Single session only**: The monitoring endpoint returns only the latest session, not a running history. Use `/session/list` for full history.

4. **No cross-binary aggregation**: Each binary (derive, execute, store, writer, gateway) has its own health server. The monitoring endpoint on the gateway does not aggregate health from other binaries.

5. **No ClickHouse health**: The monitoring endpoint does not probe ClickHouse. Analytical surface availability is determined at startup; runtime ClickHouse failures surface as 503s on individual analytical endpoints.

6. **No alerting integration**: This is a pull-based monitoring surface. It does not push alerts, set thresholds, or integrate with external alerting systems.

7. **No effectiveness/pairing in monitoring snapshot**: To keep the monitoring endpoint lightweight and low-latency (KV-only reads), effectiveness and pairing summaries are not included. These require ClickHouse queries with different latency profiles.

---

## Design Decisions

| Decision | Rationale |
|---|---|
| Single endpoint, not multiple | Operators should need one call for operational state, not three |
| KV-only reads (no ClickHouse) | Monitoring must be fast and available even when analytical surfaces are down |
| Static surface availability | Avoids health-check storms on every monitoring query |
| No effectiveness/pairing | Different latency profile; existing dedicated endpoints serve this need |
| Graceful degradation over errors | Monitoring should never fail — partial data is better than 503 |
