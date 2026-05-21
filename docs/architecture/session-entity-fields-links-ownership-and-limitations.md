# Session Entity: Fields, Links, Ownership, and Limitations

S460 | 2026-03-24

---

## 1. Field Reference

### Core Fields

| Field | Type | Required | Set By | When |
|-------|------|----------|--------|------|
| `session_id` | string | Yes | execute binary | Session open |
| `operator` | string | No | WithOperator option | Session open |
| `status` | SessionStatus | Yes | execute binary | Open/close/halt |
| `halt_reason` | string | If halted | execute binary | Session halt |
| `started_at` | time.Time | Yes | execute binary | Session open |
| `closed_at` | *time.Time | If terminal | execute binary | Session close/halt |

### Config Snapshot

| Field | Type | Description |
|-------|------|-------------|
| `config.venue_type` | string | Venue adapter type at session start |
| `config.dry_run` | bool | Whether dry-run mode was active |
| `config.segments` | []string | Enabled segment sources (e.g. ["binances", "binancef"]) |
| `config.config_file` | string | Path to config file used (optional) |

### Activation Snapshot

| Field | Type | Description |
|-------|------|-------------|
| `activation.adapter` | AdapterState | paper or venue |
| `activation.credentials` | CredentialState | present or absent |
| `activation.gate_status` | GateStatus | active or halted |
| `activation.effective` | EffectiveMode | Computed mode (paper, venue_halted, venue_live, venue_degraded) |

### Segment Counters (at close)

| Field | Type | Description |
|-------|------|-------------|
| `segment_counters[].segment` | string | Segment name (spot, futures) |
| `segment_counters[].processed` | int64 | Total intents processed |
| `segment_counters[].filled` | int64 | Total fills received |
| `segment_counters[].rejected` | int64 | Total rejections received |
| `segment_counters[].errors` | int64 | Total errors encountered |

### Artifacts

| Field | Type | Description |
|-------|------|-------------|
| `artifacts` | map[string]string | Named references to external artifacts (e.g. config_file, compose_file) |

---

## 2. Links to Other Entities

### Link: Session to Orders

- **Mechanism**: Timestamp window (`started_at` to `closed_at`)
- **Query pattern**: ClickHouse `WHERE occurred_at BETWEEN session.started_at AND session.closed_at`
- **No direct foreign key**: Session does not embed order IDs; the link is temporal
- **Rationale**: Orders are high-volume; embedding them would violate the bounded-size session record constraint

### Link: Session to Activation Surface

- **Mechanism**: `activation` snapshot captured at session open
- **Current state**: Queryable via `GET /execution/activation` for live surface
- **Session record**: Contains point-in-time snapshot, not a live reference

### Link: Session to Control Gate

- **Mechanism**: `activation.gate_status` snapshot at open; `halt_reason` at close
- **Live state**: Control gate changes during a session are visible via the existing gate endpoint, not tracked per-event in the session record

### Link: Session to Segment Health

- **Mechanism**: `segment_counters` captured at session close from venue-adapter tracker
- **Granularity**: Per-segment totals, not per-event detail

### Link: Session to Config

- **Mechanism**: `config` snapshot captured at open; `artifacts.config_file` as pointer
- **Note**: This is a value copy, not a live reference; if config changes mid-session (requires restart), a new session is created

---

## 3. Ownership

| Component | Role |
|-----------|------|
| **execute binary** | Creates and closes sessions; owns session lifecycle |
| **store binary** | Persists sessions in NATS KV; serves queries |
| **gateway binary** | Forwards session queries to store; exposes HTTP |
| **operator** | Identified via `WithOperator` option at binary startup |

### Authority Chain

```
execute binary (creates session)
    |
    v
NATS KV EXECUTION_SESSION (persistent store)
    |
    v
store binary (serves queries via request-reply)
    |
    v
gateway binary (exposes HTTP: /session/:id, /session/list)
```

---

## 4. Validation Rules

| Rule | Enforcement |
|------|-------------|
| `session_id` must not be empty | `Session.Validate()` |
| `status` must be open, closed, or halted | `Session.Validate()` |
| `started_at` must not be zero | `Session.Validate()` |
| `closed_at` must be set when terminal | `Session.Validate()` |
| `halt_reason` must be set when halted | `Session.Validate()` |
| `config.venue_type` must not be empty | `Session.Validate()` |
| Validation runs before KV write | `SessionKVStore.Put()` |

---

## 5. Limitations

| Limitation | Severity | Mitigation Path |
|------------|----------|-----------------|
| No ClickHouse persistence for sessions | LOW | KV is sufficient for bounded session count; CH can be added in a future stage if retention becomes a concern |
| No real-time session event stream | LOW | Session record is written at open and close only; mid-session state changes (gate toggle) are tracked via existing control gate events, not session events |
| Operator field is optional | LOW | Binary startup may not always have operator context; operational ceremonies set it via WithOperator |
| Session-to-order link is temporal, not referential | LOW | Temporal join is reliable because sessions are bounded time windows; a direct FK would bloat the session record |
| No multi-binary session correlation | MEDIUM | Only the execute binary participates in session lifecycle; derive/store/writer sessions are not tracked. This is intentional — session is an execution concept |
| Session KV has no TTL/automatic cleanup | LOW | Sessions are small records; manual cleanup or TTL can be added when volume warrants it |
| No session amendment after close | LOW | A closed session is immutable; if metadata needs correction, a new annotation mechanism would be needed |
