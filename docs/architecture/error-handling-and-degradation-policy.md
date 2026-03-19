# Error Handling and Degradation Policy

> Canonical reference for fail behavior, error surfacing, and degradation posture across the Market Foundry monorepo.

## 1. Design Principles

1. **Explicit over implicit.** Every runtime must declare its posture toward failure at each boundary — startup, message processing, and external dependency interaction. Silent swallowing of errors is a policy violation.
2. **Fail-fast at startup, degrade at runtime.** If a runtime cannot establish its critical dependencies during initialization, it must exit immediately. Once running, transient failures should be absorbed without crashing the process.
3. **Errors are data.** Every error path must (a) log at the appropriate level, (b) track via `healthz.Tracker`, and (c) propagate the canonical `*problem.Problem` type across architectural boundaries.
4. **Local autonomy within shared conventions.** Runtimes own their domain-specific error decisions. Cross-cutting conventions apply to boundary types, readiness contracts, and diagnostic surfaces.

## 2. Canonical Error Type: `*problem.Problem`

All functions crossing architectural boundaries (port → adapter, application → port, handler → use-case) return `*problem.Problem`, never raw `error`.

### Error Code Taxonomy

| Prefix | Code | Semantics | Typical Response |
|--------|------|-----------|-----------------|
| `VAL_` | `ValidationFailed` | Generic input validation failure | Reject input, log WARN |
| `VAL_` | `InvalidArgument` | Invalid argument shape/value | Reject input, TERM if consumer |
| `SYS_` | `NotFound` | Requested resource not found | Return empty/404 |
| `SYS_` | `Conflict` | State conflict (e.g., duplicate) | Skip or merge |
| `SYS_` | `Internal` | Unexpected system error | Log ERROR, RecordError |
| `SYS_` | `Unavailable` | Temporary network/system unavailability | NAK if consumer, degrade if optional |
| `CFG_` | (settings pkg) | Config load/parse/validation failures | Fail-fast at startup |

### Retryable Flag

`Problem.Retryable` distinguishes transient failures from permanent ones:
- **Retryable = true**: Caller should retry (e.g., NATS timeout, transient network).
- **Retryable = false** (default): Caller should not retry (e.g., validation failure, structural error).

Consumers use this to decide between `msg.Nak()` (retry) and `msg.Term()` (permanent removal).

## 3. Startup Failure Policy

All runtimes follow the same startup contract:

| Phase | Failure | Action |
|-------|---------|--------|
| Config load/parse | Any error | `os.Exit(1)` with structured log to stderr |
| Schema validation | Any error | `os.Exit(1)` with validation issues |
| Actor engine creation | Any error | Log ERROR, `os.Exit(1)` |
| Critical dependency (NATS, publisher) | Connection failure | Log ERROR, `Poison(PID)` → process exits |
| Optional dependency (control KV, config query) | Connection failure | Log WARN, continue with gate disabled |

**Invariant:** No runtime starts serving traffic until all critical dependencies are confirmed. The `/readyz` endpoint gates traffic admission.

## 4. Runtime Error Handling Contracts

### 4.1 Publisher Actors (ingest, derive)

| Outcome | Action |
|---------|--------|
| Publish success | `tracker.RecordEvent()` |
| Publish failure | `tracker.RecordError()` + log ERROR with problem code, source context |
| Startup failure | `Poison(PID)` — fatal, supervisor-level |

Publishers do **not** retry failed publishes. The rationale: NATS JetStream provides at-least-once delivery guarantees at the consumer level. A publish failure means the event is dropped at the producer, which is acceptable for derived data that will be re-derived on the next input cycle.

### 4.2 Consumer Message Processing

| Decode result | Classification | Action |
|---------------|---------------|--------|
| `InvalidArgument` | Permanent (malformed) | `msg.Term()` — remove from queue |
| Other error code | Transient | `msg.Nak()` — return for redelivery |
| Success | — | Process + `msg.Ack()` |

Consumers track redelivery count. When `NumDelivered >= MaxDeliver`, the message is logged at ERROR level before terminal exhaustion by the JetStream server.

### 4.3 Projection Actors (store)

| Outcome | Action |
|---------|--------|
| Validation rejection | `stats.rejected++`, log WARN, skip |
| Write error (Put/PutHistory) | `stats.errors++`, `tracker.RecordError()`, log ERROR, skip |
| Stale/duplicate skip | `stats.skipped*++`, log DEBUG, skip |
| Successful materialization | `stats.materialized++`, `tracker.RecordEvent()`, log INFO |

Projection actors maintain local `atomic.Int64` stats for fine-grained outcome tracking and log a summary on shutdown. Some projections (strategy, risk) verify a stats invariant: `received == sum(all outcomes)`.

### 4.4 Venue Adapter (execute)

The venue adapter implements a multi-gate pipeline:

| Gate | Failure | Action |
|------|---------|--------|
| Kill switch check | KV unavailable | Degrade: disable gate, log WARN |
| Kill switch check | Halted | Skip intent, counter `skipped_halt` |
| Staleness guard | Intent too old | Skip intent, counter `skipped_stale` |
| Venue submit | Any error | `tracker.RecordError()`, log ERROR, skip |
| Fill publish | Any error | `tracker.RecordError()`, log ERROR, skip |
| Success | — | `tracker.RecordEvent()`, counter `filled` |

### 4.5 Application Clients

All application clients (configctl, evidence, signal, decision, strategy, risk, execution) follow the same pattern:

1. Nil gateway check → `problem.Unavailable`
2. Input validation → `problem.InvalidArgument`
3. Delegate to gateway → propagate `*problem.Problem`

### 4.6 HTTP Handlers

| Condition | HTTP Status | Problem Code |
|-----------|-------------|--------------|
| Handler nil / use-case unavailable | 503 | `Unavailable` |
| Missing/invalid query parameter | 400 | `InvalidArgument` |
| Use-case returns problem | Mapped from code | Propagated |
| Success | 200 | — |

## 5. Diagnostic Surface Alignment

### Error Tracking Invariant

**Every error path that logs at ERROR level must also call `tracker.RecordError()`.** This ensures that `/statusz` and `/diagz` accurately reflect the operational state of each component.

Conversely, WARN-level events (validation rejections, stale skips, idle warnings) are tracked via domain-specific counters but do **not** increment `errorCount`. This preserves the semantic distinction: `errorCount` reflects failures that affect data flow; counters reflect operational outcomes that are expected under normal operation.

### Readiness vs. Degradation

| Endpoint | What it reflects |
|----------|-----------------|
| `/healthz` | Process is alive (always 200) |
| `/readyz` | All critical dependencies reachable (NATS, etc.) |
| `/statusz` | Per-component activity: events, errors, idle duration, counters |
| `/diagz` | Combined readiness + tracker summary |

Readiness checks gate traffic admission. They do **not** reflect degraded-but-functional states (e.g., optional control KV unavailable). Degradation is visible through `/statusz` counters and log-level signals.

## 6. Log Level Discipline

| Level | Semantics | Error tracking |
|-------|-----------|---------------|
| **ERROR** | Data loss risk, failed writes, failed publishes, startup failures | `tracker.RecordError()` |
| **WARN** | Degraded state, validation rejection, idle component, unknown messages | Domain-specific counters |
| **INFO** | Lifecycle events, successful materializations, stats summaries | `tracker.RecordEvent()` |
| **DEBUG** | Skipped events (stale, duplicate), detailed trace | — |

## 7. What Is NOT in Scope

This policy explicitly does **not** cover:
- **Retry/backoff strategies.** No retry loops exist in the current architecture. NATS JetStream redelivery is the retry mechanism for consumers. Producers do not retry.
- **Circuit breakers.** Not implemented. The control gate (kill switch) in execute is a manual safety mechanism, not an automatic circuit breaker.
- **Distributed tracing.** Correlation IDs exist in domain events but are not propagated to log lines or span contexts.
- **Alerting rules.** Idle warnings and error counts are logged/exposed via HTTP but not routed to notification channels.

These are documented as open debts for future stages.
