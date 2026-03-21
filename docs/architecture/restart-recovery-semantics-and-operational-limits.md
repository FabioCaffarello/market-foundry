# Restart Recovery Semantics and Operational Limits

**Stage:** S280
**Date:** 2026-03-21
**Status:** Documented

## Purpose

This document describes the actual recovery semantics of the market-foundry system after component restarts, including honest statements about what is and is not guaranteed. It is intended as an operational reference, not a marketing document.

## Delivery Guarantees

### Stream → Consumer Path (NATS JetStream)

| Property | Guarantee | Mechanism |
|----------|-----------|-----------|
| Delivery | At-least-once | Durable consumer + explicit ACK |
| Ordering | Per-subject FIFO | JetStream stream ordering |
| Dedup | Within ~2 minute window | MsgID-based dedup at stream level |
| Persistence | File-backed | JetStream file storage on `nats_data` volume |
| Resume position | Last ACKed message | Durable consumer state stored by NATS server |

**What this means in practice:**
- If the writer consumer crashes after receiving a message but before ACKing it, the message will be redelivered on restart.
- If the writer consumer crashes after ACKing but before the inserter flushes to ClickHouse, the message is lost from the analytical projection.
- If the same event is published twice with the same MsgID within ~2 minutes, the duplicate is silently dropped.

### Consumer → ClickHouse Path (Writer Inserter)

| Property | Guarantee | Mechanism |
|----------|-----------|-----------|
| Batching | Up to 1000 rows or 5 seconds | In-memory buffer |
| Retry | 3 attempts with exponential backoff | 100ms → 200ms → 400ms |
| Overflow | FIFO eviction above 10000 rows | Oldest rows dropped |
| Graceful shutdown | Drain buffer before exit | `actor.Stopped` handler |
| Crash shutdown | Buffer lost | No WAL or disk buffer |

**What this means in practice:**
- Under normal shutdown (SIGTERM with 15s grace), the buffer is drained before exit.
- Under abnormal shutdown (SIGKILL, OOM), buffer content (up to 1000 rows) is lost.
- ClickHouse INSERT is not idempotent — if a batch is inserted and then the ACK to NATS fails, the batch may be re-inserted on redelivery, creating duplicate rows.

### KV → Reader Path

| Property | Guarantee | Mechanism |
|----------|-----------|-----------|
| Persistence | File-backed | NATS KV with `FileStorage` |
| Monotonicity | Enforced on write | Timestamp comparison guard |
| Fail-open | Control gate defaults to active | `DefaultControlGate()` on error |
| Consistency | Eventual (poll-based) | Components poll KV; no push notification |

## Recovery Behavior by Component

### Writer Restart

```
Before restart:
  Stream: [msg1 ACK] [msg2 ACK] [msg3 ACK] [msg4 delivered, unACKed] [msg5 pending]
  Buffer: [msg4_row]  (not yet flushed)

After restart:
  Stream position resumes at msg4 (first unACKed)
  Buffer: empty (lost)
  msg4 redelivered → new buffer → flush → ClickHouse
  msg5 delivered → buffer → flush → ClickHouse

Gap: msg4 was in the old buffer. If msg4 was ACKed before crash,
     its row is permanently lost from ClickHouse.
     If msg4 was NOT ACKed, it is redelivered (no loss, possible duplicate).
```

**Supervisor recovery:** If individual consumer startup fails, the supervisor retries with exponential backoff (2s → 4s → 8s → 16s → 30s cap), up to 5 attempts per family. After exhaustion, the family is marked "degraded" and other families continue.

### Execute Restart

```
Before restart:
  Safety gate: reading from EXECUTION_CONTROL KV

After restart:
  New connection to NATS
  New KV handle for EXECUTION_CONTROL
  Safety gate re-reads current gate status
  Resume processing intents with correct halt/active state
```

**No state loss:** Execute carries no in-process state beyond the current intent being evaluated. A restart between intents is seamless.

### Store Restart

```
Before restart:
  KV handles: EXECUTION_PAPER_ORDER_LATEST, EXECUTION_CONTROL
  In-flight request/reply operations

After restart:
  New NATS connection
  New KV handles (same buckets, same data — file-backed)
  Resume serving request/reply from gateway
```

**Gap:** Any in-flight NATS request/reply operations at restart time will timeout on the gateway side. The gateway should retry or return an error to the HTTP caller.

### Gateway Restart

```
Before restart:
  HTTP server listening on :8080
  NATS connection for request/reply
  ClickHouse connection for analytical queries

After restart:
  New HTTP server (port re-bound)
  New NATS connection
  New ClickHouse connection pool
  All state is external — no loss
```

**Gap:** HTTP requests during the restart window (~5-15 seconds) will fail. Clients must retry.

## Known Limits and Gaps

### 1. Buffer Loss Window (Writer)

**Gap:** Between the last `msg.Ack()` and the next `inserter.flush()`, there is a window where events have been acknowledged to NATS but not yet written to ClickHouse. If the writer crashes in this window, those events are lost from the analytical projection.

**Bound:** Maximum `batchSize` (1000) rows or `flushInterval` (5 seconds) of data.

**Mitigation:** Durable consumer + at-least-once delivery means the events are still in the NATS stream for other consumers. Only the ClickHouse analytical copy is affected.

### 2. ClickHouse Duplicate Rows

**Gap:** If the writer successfully INSERTs a batch to ClickHouse but fails to ACK the messages (crash between INSERT and ACK), the messages will be redelivered on restart and the rows will be inserted again.

**Bound:** Maximum one batch (1000 rows) of duplicates per restart event.

**Mitigation:** ClickHouse queries can use `DISTINCT` or `argMax` patterns for deduplication. The `event_id` column provides a natural dedup key.

### 3. JetStream Dedup Window

**Gap:** JetStream MsgID deduplication has a server-configured window (default ~2 minutes). If the same event is published more than 2 minutes apart (e.g., after a long outage), it may be stored twice.

**Bound:** Only relevant if the same logical event is published again after the dedup window expires.

**Mitigation:** The KV monotonicity guard prevents stale data from being projected. For the analytical path, duplicate rows are bounded by the stream dedup window.

### 4. No Automatic Reconnect

**Gap:** If NATS becomes unavailable while a service is running, the service does not attempt to reconnect. It relies on Docker's `restart: unless-stopped` policy to restart the entire process.

**Bound:** Recovery time is Docker restart delay + service startup time (typically 5-15 seconds).

**Mitigation:** This is a deliberate simplicity choice. Client-side reconnect adds complexity and partial-state risk. Full process restart is cleaner and well-tested.

### 5. Control Gate Polling Latency

**Gap:** Components read the control gate by polling the KV bucket. There is no push notification for gate state changes. A gate change (halt → active or vice versa) may take up to one polling cycle to propagate.

**Bound:** Polling interval depends on event rate. For derive, it's checked once per event. For execute, it's checked once per intent.

**Mitigation:** Acceptable for the current use case. Real-time gate propagation would require a NATS KV watch, which is a future enhancement.

### 6. No WAL or Checkpoint for Inserter

**Gap:** The writer inserter has no write-ahead log (WAL) or checkpoint mechanism. The in-memory buffer is the only staging area between NATS consumption and ClickHouse insertion.

**Bound:** See Buffer Loss Window (limit 1).

**Mitigation:** Adding a disk-backed buffer or WAL is a future enhancement if the buffer loss window proves unacceptable in production.

## What Is NOT Covered by S280

| Scenario | Reason Not Covered |
|----------|--------------------|
| NATS server crash | Infrastructure-level; outside application scope |
| ClickHouse crash | Infrastructure-level; writer retries are bounded |
| Network partition | Requires distributed systems testing framework |
| Simultaneous multi-service crash | Combinatorial explosion; diminishing returns |
| Data corruption recovery | Requires backup/restore procedures |
| Graceful degradation under load | Performance testing, not restart recovery |
| Exactly-once ClickHouse delivery | Requires WAL or transactional INSERT |

## Operational Recommendations

1. **Monitor buffer depth:** The writer exposes `buffer_depth` gauge per family. Alert if it consistently approaches `maxPending` (10000).

2. **Monitor redelivery rate:** Consumer `redelivered` counter should be near zero in steady state. Sustained redelivery indicates ACK failures.

3. **Monitor flush failures:** `flush_failures` counter should be zero. Non-zero indicates ClickHouse connectivity or schema issues.

4. **Restart budget:** The supervisor allows 5 restarts per family per process lifetime. If degraded families appear, investigate root cause before restarting the writer process.

5. **Docker restart policy:** `restart: unless-stopped` ensures automatic recovery. Do not use `restart: always` as it prevents intentional shutdown.
