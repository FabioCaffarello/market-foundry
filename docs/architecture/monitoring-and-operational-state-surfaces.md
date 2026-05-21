# Monitoring and Operational State Surfaces

**Stage**: S486
**Date**: 2026-03-26
**Status**: Active

---

## Purpose

This document defines the minimal monitoring and operational state surfaces introduced in S486. These surfaces close the gap where an operator needed 3+ separate HTTP calls to understand "what is the system doing right now?"

The goal is **operational legibility**, not observability platform or dashboard infrastructure.

---

## Surface: `GET /monitoring/state`

**What it answers**: What is the system doing right now?

**Response shape**:

```json
{
  "observed_at": "2026-03-26T12:00:00Z",
  "session": {
    "session_id": "session_20260326_120000",
    "operator": "op1",
    "status": "open",
    "started_at": "2026-03-26T12:00:00Z",
    "config": {
      "venue_type": "binance",
      "dry_run": true,
      "segments": ["spot"]
    },
    "counters": [
      {"segment": "spot", "processed": 100, "filled": 10, "rejected": 2, "errors": 1}
    ]
  },
  "gate": {
    "status": "active",
    "updated_at": "2026-03-26T11:59:00Z"
  },
  "surfaces": {
    "evidence": true,
    "signal": true,
    "decision": true,
    "strategy": true,
    "risk": true,
    "execution": true,
    "session": true,
    "analytical": true,
    "activation": true
  }
}
```

### Fields

| Field | Source | Semantics |
|---|---|---|
| `session` | Latest from `/session/list` (NATS KV via store) | Most recent session (open or terminal). Nil when no sessions exist. |
| `gate` | `/execution/control` (NATS KV) | Current kill-switch state. Nil when execution control gateway is unavailable. |
| `surfaces` | Static snapshot from gateway composition | Which endpoint families were wired at startup. Does not probe runtime availability. |

### Graceful Degradation

- If the session gateway is unavailable, `session` is null — no error.
- If the execution control gateway is unavailable, `gate` is null — no error.
- The endpoint itself always returns 200 with whatever data is available.
- `surfaces` is always populated (captured at composition time, not at query time).

---

## Relationship to Existing Surfaces

| Existing surface | What it provides | How /monitoring/state differs |
|---|---|---|
| `/statusz` (health server) | Runtime phase, trackers, idle warnings | Per-binary health. No session or gate context. |
| `/session/list` | Full session list with all fields | /monitoring/state returns only the latest session with lightweight summary fields. |
| `/execution/control` | Full gate + activation surface | /monitoring/state returns only gate status/reason. |
| `/readyz` | Binary readiness | Pass/fail, no operational context. |

The monitoring endpoint **does not replace** any existing surface. It aggregates a subset for operational convenience.

---

## What This Surface Does NOT Cover

- Real-time streaming or push notifications.
- Alerting rules or thresholds.
- Effectiveness or pairing summaries (use `/analytical/composite/decision/effectiveness/summary` and `/analytical/composite/pairing`).
- Full audit detail (use `/session/:id/audit`).
- ClickHouse health or query performance.
- Multi-binary pipeline health (each binary has its own `/statusz`).

---

## Architecture Invariants

1. The monitoring endpoint is **read-only** — it never modifies state.
2. Surface availability is captured **at composition time** — it does not probe dependencies at query time.
3. Session summary is a **projection**, not a copy — it carries only monitoring-relevant fields.
4. All fields degrade independently — a failure in one source does not block others.
