# Segment Health/Readiness Signals: Semantics, Coverage, and Limitations

> S429 — Defines the semantics, coverage boundaries, and known limitations of per-segment health and readiness signals in the unified runtime.

## Signal Semantics

### Per-Segment Phase

Each segment has an independently computed phase:

| Phase | Meaning | Operator Action |
|-------|---------|-----------------|
| `disabled` | Segment not enabled in config | None — by design |
| `ready` | Segment enabled, no activity yet | Normal during startup or idle periods |
| `active` | Segment processing intents successfully | Healthy operating state |
| `degraded` | Segment has errors but no successful processing | Investigate — adapter may be failing |

**Important:** Phase is computed from cumulative counters, not from a sliding window. A segment that was active and later stopped processing will remain `active` (not transition to `idle`) until the process restarts. This is a deliberate simplicity tradeoff — idle detection per segment would require per-segment timestamp tracking, which is deferred.

### Counter Semantics

| Counter | Scope | Incremented When |
|---------|-------|------------------|
| `<seg>:processed` | Per-segment | Intent received and source maps to this segment |
| `<seg>:filled` | Per-segment | Fill event published successfully for this segment |
| `<seg>:rejected` | Per-segment | Rejection event published for this segment |
| `<seg>:errors` | Per-segment | Submit failed (after retries exhausted) for this segment |

Counters are monotonically increasing per process lifetime. They reset on binary restart.

### Relationship to Global Counters

Global counters (`processed`, `filled`, `rejected`) and per-segment counters are incremented independently. The sum of per-segment counters should equal the global counter for multi-segment deployments:

```
processed == spot:processed + futures:processed
```

This invariant is maintained by design — both are incremented in the same code path.

## Coverage Matrix

| Signal | Spot | Futures | Segment-Aware? |
|--------|------|---------|----------------|
| `/healthz` (liveness) | Global | Global | No — liveness is deployment-wide |
| `/readyz` (readiness) | Global | Global | No — NATS check is shared |
| `/statusz` segments array | Yes | Yes | Yes — per-segment phase and counters |
| `/diagz` segments array | Yes | Yes | Yes — per-segment phase and counters |
| Tracker counters | Yes | Yes | Yes — `spot:*` and `futures:*` prefixes |
| ActivationSurface | Global | Global | No — single gate, single adapter state |
| Control gate (kill switch) | Global | Global | No — halts all segments uniformly |

## What This Stage Delivers

1. **Per-segment health visibility in /statusz and /diagz** — operators can distinguish Spot vs Futures health without log analysis.
2. **Per-segment counters** — `spot:processed`, `spot:filled`, `futures:rejected`, etc., available in tracker state.
3. **SegmentHealthRegistry** — reusable primitive for segment-aware health aggregation.
4. **Phase computation per segment** — `disabled`/`ready`/`active`/`degraded` independently per segment.

## What This Stage Does NOT Deliver

1. **Per-segment readiness checks** — `/readyz` remains deployment-wide. A per-segment readiness endpoint (e.g., `/readyz?segment=spot`) is not implemented. The NATS readiness check is shared because NATS is a shared dependency.

2. **Per-segment idle detection** — The global `idle` phase considers tracker-level idle thresholds. Per-segment idle detection would require per-segment timestamps, which is deferred to avoid complexity.

3. **Per-segment ActivationSurface** — The activation surface (adapter state, gate, credentials) remains global. Per-segment gates (e.g., halt Futures while Spot continues) are not implemented.

4. **Per-segment control gate** — The kill switch halts all segments. Per-segment halt capability is a significant architectural change deferred to a future wave.

5. **Per-segment alerting** — No alerting rules, thresholds, or notification channels. This stage exposes signals; consuming them is out of scope.

6. **Ingest-side segment health** — The ingest binary's health tracking remains per-publisher, not per-exchange. Ingest segment health is deferred.

7. **Store-side segment health** — The store/writer binaries have no segment concept in their health tracking.

8. **Dashboard or UI** — No Grafana boards, no web UI. Only JSON HTTP endpoints.

## Limitations

### L1: Cumulative Counters Only

Counters are monotonic per process lifetime. There is no rate or windowed view. An operator sees "spot:filled = 1000" but not "spot fill rate = 5/min". Rate derivation requires external tooling (Prometheus scrape + rate() function).

### L2: Shared Tracker

Both Spot and Futures segment counters live on the same `venue-adapter` Tracker instance. This means:
- Global phase computation considers all segments together
- A degraded Futures segment may not affect the global `phase` if Spot is active
- Per-segment phase and global phase can diverge

### L3: No Cross-Binary Aggregation

Each binary (execute, ingest, store) has its own health server. There is no unified "system health" endpoint that aggregates across binaries. The gateway `/readyz` checks its own readiness but does not probe execute's segment health.

### L4: Phase Does Not Reflect Recency

A segment with `spot:processed=100` from 2 hours ago still shows `active`. Only the global tracker's idle detection (via `IdleSince()`) provides staleness signals, and those are not per-segment.

### L5: Single-Segment Configs

When only one segment is enabled (e.g., Spot only), the segments array contains only that segment. The other segment does not appear — it is not registered, not "disabled". This is intentional: the registry only tracks configured segments.

## Operational Guidance

### Checking Segment Health

```bash
# Full status with segments
curl -s http://localhost:8084/statusz | jq '.segments'

# Specific segment phase
curl -s http://localhost:8084/statusz | jq '.segments[] | select(.segment == "spot") | .phase'

# Check if any segment is degraded
curl -s http://localhost:8084/statusz | jq '[.segments[] | select(.phase == "degraded")] | length'
```

### Interpreting Phases

- **Both segments `ready` after startup** — Normal. Awaiting first intents.
- **One `active`, one `ready`** — The active segment is processing. The ready segment may not have received intents yet. Check if the segment's source subjects have publishers.
- **One `active`, one `degraded`** — The degraded segment has only errors. Check logs for the specific error. Common causes: credential issues, venue API errors, adapter misconfiguration.
- **Both `active`** — Healthy multi-segment operation.
