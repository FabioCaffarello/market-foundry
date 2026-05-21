# Verification Trigger: Events, Dedup, Fail-Closed Semantics, and Limitations

**Stage**: S490
**Status**: Implemented

## Trigger Events

### Which Events Trigger Verification

| Session Status | Triggers Verification | Rationale |
|---------------|----------------------|-----------|
| `closed` | Yes | Normal session end — verification confirms operational integrity |
| `halted` | Yes | Emergency stop — verification is especially important |
| `open` | No | Session is still active, verification would query incomplete data |

### Event Subject Routing

Events are published to `execution.session.lifecycle.{status}`:
- `execution.session.lifecycle.closed`
- `execution.session.lifecycle.halted`

The consumer subscribes to `execution.session.lifecycle.>` (wildcard) to receive all terminal status events.

## Deduplication

### JetStream Message-Level Dedup

Each event carries a deduplication key:

```
session-lifecycle:{session_id}:{status}
```

Example: `session-lifecycle:session_20260326_120000:closed`

JetStream enforces at-most-once delivery per message ID within the stream's dedup window. A session close event for the same session ID and status will not be delivered twice.

### Consumer-Level Idempotency

Even if redelivered (JetStream retry on ack failure), verification is idempotent:
- The same PO checks run against the same session data
- Results may differ slightly due to timing (e.g., more writes settling) but are never destructive
- No state mutation occurs — verification is read-only

### Closed vs. Halted for Same Session

A session that is halted has a different dedup key than a closed session. In practice, a session can only reach one terminal state (the first terminal transition is final). If both events were somehow published, verification would run twice — this is harmless due to idempotency.

## Fail-Closed Semantics

### Publisher Failure (Execute Binary)

| Failure | Behavior | Recovery |
|---------|----------|----------|
| Publisher not started | Log warning, skip publishing | Manual verification via HTTP/script |
| Publish times out | Log warning with session ID | Manual verification; session data in KV |
| Stream unavailable | Log warning | Manual verification; data is intact |

**Invariant**: Publisher failure never prevents session close. The session is always persisted to KV before publishing.

### Consumer Failure (Gateway Binary)

| Failure | Behavior | Recovery |
|---------|----------|----------|
| Consumer start fails | Log warning, gateway runs without trigger | Manual verification |
| Event decode fails | Log error, terminate message (non-recoverable) | Event is dead-lettered after max retries |
| Verification times out | Log error, ack message | Operator runs manual verification |
| Verification returns error | Log error, ack message | Operator investigates via audit endpoint |

**Invariant**: Consumer failure never blocks the gateway's HTTP server. The trigger is a background process.

### Verification Failure

| Failure | Behavior | Ack? |
|---------|----------|------|
| Session not found | Log error | Yes — retrying won't help |
| ClickHouse unavailable | Checks skip, report has skipped verdicts | Yes — report logged |
| Timeout | Log error | Yes — retrying would likely timeout again |
| All checks pass | Log success | Yes |
| Some checks fail | Log warning with failure count | Yes |

**Invariant**: Verification failure always acks the message. Retrying the same verification would produce the same result (or worse, if transient conditions resolved). Failed verifications are surfaced in logs for operator review.

## Anti-Loop Protection

### Why Loops Cannot Occur

1. **Trigger direction is one-way**: session close → verification. Verification never closes sessions.
2. **No feedback loop**: verification results are logged, not published as events.
3. **Dedup key prevents replay**: even if the same event were re-published, JetStream dedup blocks it.
4. **Consumer acks unconditionally**: after processing, the message is always acked. The only exception is decode failure, which terminates the message.

### Max Delivery Safety Net

The consumer has `MaxDeliver: 5`. If a message cannot be processed after 5 attempts (ack failures), NATS stops delivering it. This prevents infinite retry loops even in pathological scenarios.

## Settle Delay

The trigger waits 5 seconds between receiving the event and running verification. This allows:

- ClickHouse writer consumer to process and persist recent fills/rejections
- Store binary to materialize latest KV projections
- Any in-flight NATS messages to settle

This is a best-effort delay, not a guarantee. If writes are still in-flight after 5 seconds, affected PO checks will return `skip` or `warn` verdicts — not false passes.

## Limitations

### L1: No Report Persistence

Event-driven verification logs results but does not persist them to `backups/sessions/` or any external store. Operators who need persisted reports should run `make po-verify PO_FLAGS="--save"` or query the HTTP endpoint.

**Rationale**: Adding persistence would require filesystem access or a new write path. The trigger's purpose is alerting, not archival.

### L2: Gateway Must Be Running

The durable consumer lives in the gateway binary. If the gateway is down when a session closes, the event is queued in JetStream and processed when the gateway restarts. However, if the gateway is down for longer than 7 days (stream max age), the event expires.

**Mitigation**: 7-day retention is generous. In practice, the gateway runs alongside other binaries.

### L3: No Alerting Integration

Verification failures are logged but not sent to external alerting systems (Slack, PagerDuty, etc.). The trigger provides the reactive mechanism; alerting integration is a future concern.

### L4: Single Consumer Instance

The durable consumer runs in one gateway instance. In a multi-gateway deployment, only one instance processes each event (NATS queue group semantics via durable consumer). This is correct behavior, not a limitation — verification should run once per session, not once per gateway.

### L5: 5-Second Settle Delay Is Heuristic

The 5-second delay is sufficient for typical write latency but not guaranteed. Under extreme load or ClickHouse write backpressure, some checks may skip. The operator can always re-run verification manually for a definitive result.

### L6: No Partial Re-verification

If verification is triggered and some checks skip due to unavailable data, there is no mechanism to re-trigger for the skipped checks later. The operator must re-run manually.

## Coverage Summary

| Concern | Covered | Mechanism |
|---------|---------|-----------|
| Terminal session detection | Yes | SessionLifecycleEvent with status filter |
| Deduplication | Yes | JetStream message ID + idempotent verification |
| Anti-loop | Yes | One-way trigger, no feedback, unconditional ack |
| Publisher failure | Yes | Non-fatal, manual fallback |
| Consumer failure | Yes | Non-fatal, gateway continues |
| Verification failure | Yes | Logged, acked, no retry |
| Data settle timing | Partial | 5s delay heuristic |
| Report persistence | No | L1 — logs only |
| External alerting | No | L3 — future concern |
