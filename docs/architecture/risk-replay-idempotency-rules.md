# Risk Replay & Idempotency Rules

> Stage S65 — Approved 2026-03-18
> Status: **Active invariant documentation**

---

## 1. Overview

Risk replay safety is provided by three independent, layered mechanisms. Each layer is self-sufficient — any single layer can prevent duplicate or stale data from corrupting the read model.

```
Layer 1: JetStream Deduplication (publish-side)
Layer 2: Durable Consumer + Ack (consume-side)
Layer 3: Monotonicity Guard (projection-side)
```

---

## 2. Layer 1: JetStream Deduplication (Publish-Side)

**Where**: `RiskPublisher.PublishRisk()` in derive binary.

**Mechanism**: Each published message includes a `Nats-Msg-Id` header derived from `RiskAssessment.DeduplicationKey()`:

```
risk:{type}:{source}:{symbol}:{timeframe}:{unix_timestamp}
```

Example: `risk:position_exposure:binancef:btcusdt:60:1710777600`

**Behavior**: JetStream maintains a deduplication window (default 2 minutes). If the same `Msg-Id` is published twice within the window, the second publish is silently accepted but not stored in the stream.

**What this prevents**: Duplicate events from evaluator retries or actor restarts within the derive binary.

**Limitation**: The dedup window is time-bounded. Events replayed after the window closes will appear as new events in the stream (handled by Layer 3).

---

## 3. Layer 2: Durable Consumer + Explicit Ack (Consume-Side)

**Where**: `RiskConsumer` in store binary.

**Mechanism**: The store binary uses a durable consumer (`store-risk-position-exposure`) with explicit ack policy:

| Setting | Value | Purpose |
|---------|-------|---------|
| `Durable` | `store-risk-position-exposure` | Survives consumer restarts |
| `AckPolicy` | `AckExplicit` | Message not removed until explicitly acked |
| `AckWait` | 30s | Redelivers if not acked within 30s |
| `MaxDeliver` | 5 | Stops redelivery after 5 attempts |

**Ack protocol**:
1. Message decoded successfully and handler invoked → `Ack()`
2. Message decode fails with `InvalidArgument` → `Term()` (poison message, stop redelivery)
3. Message decode fails with transient error → `Nak()` (trigger immediate redelivery)

**Redelivery detection**: When `NumDelivered > 1`, the consumer logs a warning with subject, delivery count, and stream sequence number. This makes redelivery visible without requiring external monitoring.

**What this prevents**: Lost events due to crashes between receive and ack.

**What this does NOT prevent**: Duplicate processing — a crash after handler execution but before ack will cause redelivery. This is why Layer 3 exists.

---

## 4. Layer 3: Monotonicity Guard (Projection-Side)

**Where**: `RiskKVStore.Put()` in store binary.

**Mechanism**: Before writing to KV, the store reads the existing entry and compares timestamps:

```
existing.Timestamp > new.Timestamp → PutSkippedStale (reject)
existing.Timestamp == new.Timestamp → PutSkippedDuplicate (reject)
existing.Timestamp < new.Timestamp → PutWritten (accept)
key not found → PutWritten (first write)
```

**What this prevents**:
- Stale data overwriting newer data during replay
- Duplicate writes from consumer redelivery
- Out-of-order processing from concurrent consumers (not current architecture, but safe if added)

**This is the definitive safety layer.** Even if Layers 1 and 2 fail completely, the monotonicity guard ensures the KV bucket always holds the newest assessment.

---

## 5. Replay Scenarios

### Scenario A: Store Binary Restarts

1. Durable consumer resumes from last acked position.
2. No events are reprocessed (JetStream tracks ack state per durable consumer).
3. **Result**: No duplicate processing.

### Scenario B: Store Binary Crashes Mid-Processing

1. Event was received but not acked.
2. After `AckWait` (30s), JetStream redelivers the event.
3. Consumer detects `NumDelivered > 1` and logs warning.
4. If the event was already materialized before crash, monotonicity guard skips it (`PutSkippedDuplicate`).
5. If the event was not materialized, it is materialized normally.
6. **Result**: At-least-once delivery, exactly-once materialization.

### Scenario C: Derive Binary Republishes Same Event

1. If within JetStream dedup window: silently deduplicated at stream level.
2. If after dedup window: event appears in stream, delivered to consumer.
3. Monotonicity guard skips it (`PutSkippedDuplicate` — same timestamp).
4. **Result**: No duplicate materialization.

### Scenario D: Manual Stream Replay (Ops Intervention)

1. Operator creates a new ephemeral consumer starting from stream position 0.
2. All historical events are redelivered to projection actor.
3. Monotonicity guard rejects every event older than the current KV entry.
4. Only events newer than the current entry (if any) are materialized.
5. **Result**: Safe. Latest-only semantics preserved.

### Scenario E: KV Bucket Purged (Data Loss Recovery)

1. KV bucket is empty — all keys return `ErrKeyNotFound`.
2. Replay from stream: every event is the "first write" for its key.
3. Multiple events for the same key: monotonicity guard ensures only the latest survives.
4. **Result**: Full recovery to latest state, assuming stream retention covers the data.

---

## 6. Invariants

| ID | Invariant | Enforced By |
|----|-----------|-------------|
| RI-1 | A risk event is published at most once per `DeduplicationKey` within the dedup window | JetStream Msg-ID dedup |
| RI-2 | A risk event is acked exactly once per durable consumer | Durable consumer + explicit ack |
| RI-3 | A risk event may be delivered more than once but materialized at most once | Monotonicity guard |
| RI-4 | The KV bucket always holds the assessment with the newest timestamp per partition key | Monotonicity guard |
| RI-5 | `received == materialized + skipped_stale + skipped_dedup + skipped_non_final + rejected + errors` | Stats invariant check at shutdown |
| RI-6 | Only `final == true` assessments are materialized | Final flag gate |
| RI-7 | Only domain-valid assessments are materialized | Validation gate |
| RI-8 | Only domain-valid assessments are served via query | Post-read validation in KV Get |

---

## 7. What Is NOT Guaranteed

| Non-guarantee | Why | Mitigation |
|---------------|-----|-----------|
| Exactly-once processing | At-least-once is inherent in durable consumers | Monotonicity guard makes reprocessing safe |
| Total ordering across partition keys | JetStream is ordered per subject, not globally | Each partition key has independent monotonicity |
| Zero-latency consistency | KV update is asynchronous relative to event publish | Acceptable for risk — latest-only semantics tolerate brief staleness |
| Recovery beyond stream retention | RISK_EVENTS has 72h retention | For longer history, use a dedicated archival sink (future) |
