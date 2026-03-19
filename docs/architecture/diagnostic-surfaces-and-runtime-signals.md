# Diagnostic Surfaces and Runtime Signals

> Catalog of HTTP diagnostic endpoints, runtime lifecycle signals, and debugging workflows.

---

## HTTP Diagnostic Surface Catalog

All diagnostic endpoints return JSON with `Content-Type: application/json`. All use `GET` method only.

### /healthz -- Liveness

**Purpose:** Confirm the process is alive and the HTTP stack is functional.

**Request:**
```
GET /healthz
```

**Response (200 OK):**
```json
{"status": "ok"}
```

**Status codes:**
- `200` -- process is alive.
- No other status is returned. If the process is dead, the connection fails.

**Use case:** Kubernetes `livenessProbe`. If this endpoint stops responding, the orchestrator restarts the container. No application logic is checked -- pure process liveness.

---

### /readyz -- Readiness

**Purpose:** Confirm that all external dependencies are reachable and the runtime can serve traffic.

**Request:**
```
GET /readyz
```

**Response (200 OK -- all checks pass):**
```json
{"status": "ready"}
```

**Response (503 Service Unavailable -- a check failed):**
```json
{
  "status": "not_ready",
  "check": "nats",
  "error": "nats: no servers available for connection"
}
```

**Status codes:**
- `200` -- all readiness checks passed.
- `503` -- at least one check failed. The first failing check is reported.

**Checks by runtime:**

| Runtime | Checks |
|---------|--------|
| gateway | (none -- gateway readiness is HTTP server availability) |
| store | NATS connectivity |
| derive | NATS connectivity |
| ingest | NATS connectivity |
| execute | NATS connectivity |
| configctl | NATS connectivity |

**Use case:** Kubernetes `readinessProbe`. Prevents routing traffic to instances that cannot fulfill requests (e.g., NATS not yet connected).

---

### /statusz -- Activity Status

**Purpose:** Report per-component activity, runtime metadata, and idle warnings.

**Request:**
```
GET /statusz
```

**Response (200 OK):**
```json
{
  "status": "ok",
  "runtime": "store",
  "started_at": "2026-03-19T14:00:00Z",
  "uptime_seconds": 3600,
  "trackers": [
    {
      "name": "evidence-projection",
      "last_event_at": "2026-03-19T14:59:30Z",
      "event_count": 42850,
      "error_count": 3,
      "idle_seconds": 30,
      "idle_warning": false,
      "counters": {
        "filled": 42000,
        "skipped_stale": 850
      }
    },
    {
      "name": "evidence-consumer",
      "last_event_at": "2026-03-19T14:59:31Z",
      "event_count": 42850,
      "error_count": 0,
      "idle_seconds": 29,
      "idle_warning": false
    }
  ]
}
```

**Status codes:**
- `200` -- always. The status field reflects tracker state, not endpoint health.

**Field semantics:**

| Field | Meaning |
|-------|---------|
| `runtime` | Binary name of this runtime instance |
| `started_at` | RFC3339 timestamp of runtime start |
| `uptime_seconds` | Seconds since runtime started |
| `trackers[].name` | Component name (follows `<family>-<role>` convention) |
| `trackers[].last_event_at` | RFC3339 timestamp of most recent event (omitted if no events) |
| `trackers[].event_count` | Total events processed |
| `trackers[].error_count` | Total errors recorded (omitted if zero) |
| `trackers[].idle_seconds` | Seconds since last event (omitted if no events) |
| `trackers[].idle_warning` | True if idle duration exceeds threshold (default 2 minutes) |
| `trackers[].counters` | Custom named counters (omitted if none) |

**Use case:** Operator dashboarding, pipeline stall detection, post-incident inspection.

---

### /diagz -- Diagnostic Summary

**Purpose:** Single-request overview combining readiness state and tracker health.

**Request:**
```
GET /diagz
```

**Response (200 OK):**
```json
{
  "runtime": "store",
  "uptime_seconds": 3600,
  "readiness": {
    "status": "ready",
    "checks": [
      {"name": "nats", "status": "pass"}
    ]
  },
  "trackers": {
    "active": 4,
    "idle": 1,
    "total_events": 171400,
    "total_errors": 3
  }
}
```

**Response when a readiness check fails:**
```json
{
  "runtime": "store",
  "uptime_seconds": 3600,
  "readiness": {
    "status": "not_ready",
    "checks": [
      {"name": "nats", "status": "fail", "error": "connection refused"}
    ]
  },
  "trackers": {
    "active": 0,
    "idle": 4,
    "total_events": 171400,
    "total_errors": 3
  }
}
```

**Status codes:**
- `200` -- always. This is a diagnostic report, not a health gate.

**Use case:** Incident triage. One `curl` to `/diagz` answers: "Is this runtime connected to its dependencies? Is it processing events? Are there errors?" Without needing to parse multiple endpoints.

---

## Runtime Lifecycle Signals

### Startup Log

Each runtime emits a single INFO log as its first meaningful action after logger construction:

| Runtime | Log message | Extra fields |
|---------|-------------|--------------|
| gateway | `"gateway starting"` | `addr` |
| store | `"store starting"` | (none) |
| derive | `"derive starting"` | (none) |
| ingest | `"ingest starting"` | (none) |
| execute | `"execute starting"` | (none) |
| configctl | `"configctl starting"` | (none) |

All startup logs carry the `runtime` field automatically via the logger's default attributes.

### Shutdown Signal Log

When `WaitTillShutdown` receives SIGTERM or SIGINT, it logs the signal type before beginning actor poison:

```
level=INFO msg="received shutdown signal" runtime=store signal=SIGTERM
```

This enables post-mortem distinction between operator-initiated shutdowns (SIGTERM from orchestrator), interactive interrupts (SIGINT from terminal), and crashes (no shutdown log at all).

### Shutdown Complete Log

After all actors have been poisoned and stopped:

```
level=INFO msg="shutdown complete" runtime=store
```

If this log is absent, the shutdown did not complete cleanly within the poison timeout (10 seconds).

---

## Health Tracker Signals

### Event Recording

Actors call `tracker.RecordEvent()` after successfully processing an event. This updates the last-event timestamp and increments the event counter. These values are visible via `/statusz`.

### Error Recording

Actors call `tracker.RecordError()` when event processing fails. This updates the last-event timestamp (keeping the tracker "alive") and increments the error counter separately. Errors do not suppress the event count -- they are tracked in parallel.

### Custom Counters

Actors can record domain-specific metrics via `tracker.Counter("name").Add(1)`. Common counters include:

| Counter | Meaning |
|---------|---------|
| `filled` | Projection rows successfully written |
| `skipped_stale` | Events skipped because newer data exists |
| `ack_failed` | NATS acknowledgments that failed |

Custom counters appear in `/statusz` under the tracker's `counters` map.

---

## Idle Monitoring

### Heartbeat Loop

The health server runs a background goroutine that checks all trackers every 30 seconds. For each tracker that has recorded at least one event (event_count > 0 or error_count > 0):

- If idle duration exceeds the threshold (default 2 minutes), emit a WARN log.
- If idle duration is within threshold, do nothing (silent heartbeat).

### Idle Warning Log Format

```
level=WARN msg="component idle" component=healthz tracker=evidence-projection idle_seconds=180 last_event=2026-03-19T14:55:00Z event_count=42850 error_count=3
```

### Why 30s/2min

- **30-second heartbeat**: frequent enough to detect stalls within a reasonable window, infrequent enough to produce negligible overhead.
- **2-minute idle threshold**: market data pipelines process events continuously. Two minutes of silence is abnormal and warrants operator attention. The threshold is configurable via `WithIdleThreshold` for runtimes with different activity patterns.

### Trackers With No Events

Trackers that have never recorded an event are not reported as idle. This handles the startup period where consumers have been created but no messages have arrived yet.

---

## Actor Lifecycle Signals

### Started/Stopped Pattern

Actors log their lifecycle transitions at INFO level:

```
level=INFO msg="evidence projection started" runtime=store actor=evidence-projection
level=INFO msg="evidence projection stopped" runtime=store actor=evidence-projection
```

These are emitted from the actor's `Started()` and lifecycle hooks in the Hollywood framework.

### Unknown Message Warnings

When an actor receives a message type it does not handle, it logs a WARN:

```
level=WARN msg="unknown message" runtime=store actor=evidence-projection type=*actor.DeadLetterEvent
```

This catches integration errors (wrong message routed to wrong actor) without crashing.

---

## Supervisor Startup Summaries

Each supervisor logs a summary of what it activated during spawn. These are INFO-level logs emitted once during Phase 4.

### Store Supervisor

```
level=INFO msg="store supervisor started" runtime=store families="evidence,signal,decision" streams=3 consumers=3
```

Fields: which domain families are enabled, how many JetStream streams and consumers were created.

### Derive Supervisor

```
level=INFO msg="derive supervisor started" runtime=derive families="signal,decision,strategy,risk" sources=4 timeframes="1m,5m,15m,1h"
```

Fields: which derivation families are active, how many source scopes, which timeframes.

### Ingest Supervisor

```
level=INFO msg="ingest supervisor started" runtime=ingest sources=2 venues="binance,kraken"
```

Fields: how many ingestion sources, which venues are active.

### Execute Supervisor

```
level=INFO msg="execute supervisor started" runtime=execute venue=binance mode=paper
```

Fields: which venue adapter, execution mode (paper/live).

---

## Debugging Workflows

### Pipeline stall (no events flowing)

1. Check `/statusz` on the suspect runtime. Look for `idle_warning: true` on trackers.
2. Check `/readyz` -- if NATS is not ready, events cannot flow.
3. Check `/diagz` for a single-request overview of readiness + tracker state.
4. Check logs filtered by `runtime=<name>` for WARN-level idle messages.
5. Check upstream runtime's `/statusz` to determine if the stall originates further up the pipeline.

### Startup failure

1. Check logs filtered by `runtime=<name> level=ERROR`. Startup errors log the failing phase.
2. If no startup log exists (no `"<runtime> starting"` message), the binary failed before logger construction (config parsing, flag validation).
3. Check the supervisor summary log. If absent, actor spawn failed.

### Connectivity issues

1. `curl /readyz` -- reports which dependency check failed and the error message.
2. `curl /diagz` -- shows readiness status alongside tracker activity.
3. Logs filtered by `component=nats` or `component=healthz` will show connection errors and idle warnings.

### High error rate

1. Check `/statusz` for trackers with elevated `error_count`.
2. Custom counters (e.g., `ack_failed`) provide more granularity.
3. Logs filtered by `runtime=<name> level=ERROR` will contain the error details with the `error` field.

### Distinguishing shutdown causes

1. `"received shutdown signal" signal=SIGTERM` -- orchestrator-initiated (deployment, scaling).
2. `"received shutdown signal" signal=SIGINT` -- operator-initiated (Ctrl+C, manual stop).
3. No shutdown signal log -- process crashed or was killed with SIGKILL (no graceful shutdown).
4. `"shutdown complete"` present -- clean shutdown. Absent -- poison timeout exceeded.
