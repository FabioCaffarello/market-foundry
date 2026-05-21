# Multi-Binary Smoke: Failure Isolation Findings and Limitations

> Stage S374 — Honest accounting of failure isolation evidence, findings, and gaps.

## Findings

### F1: Binary Failures Are Isolated by NATS Decoupling

When derive restarts, execute/store/gateway remain fully operational. This is because:
- Execute reads from STRATEGY_EVENTS stream (NATS), not from derive directly
- Store reads from all event streams (NATS), not from derive or execute directly
- Gateway reads from store (NATS request/reply), not from upstream binaries

The NATS JetStream layer acts as a complete decoupling boundary.

### F2: Durable Consumers Enable Zero-Loss Resume

After any binary restart, its durable consumers resume from the last acknowledged sequence. This means:
- No events are lost during the restart window
- Events published during downtime are buffered in the stream
- On restart, the consumer drains the backlog

The durable consumer spec is deterministic: `ExecuteStrategyMeanReversionEntryConsumer()` always returns the same durable name, ensuring the JetStream server associates the reconnected consumer with the correct checkpoint.

### F3: Control Gate Survives All Restart Scenarios

The execution control gate (halt/active) is stored in a NATS KV bucket. This state:
- Survives derive restart (gate lives in NATS, not derive)
- Survives execute restart (execute re-reads gate on startup)
- Survives store restart (KV bucket is in NATS, store just projects it)
- Survives gateway restart (gateway re-reads gate from store)

### F4: Tracker Metrics Are Binary-Local

Each binary maintains its own `healthz.Tracker` instances. A restart in one binary resets only that binary's counters; other binaries' counters are unaffected. This was proven structurally:
- Creating three independent trackers and incrementing one does not affect others
- Counter continuity is maintained when the same tracker is reused across actor lifecycles

### F5: Staleness Guard Protects Against Stale Replays

After a binary restart, the consumer may replay events from the stream. The staleness guard (2-minute default) ensures that events older than the threshold are rejected, preventing stale data from reaching the venue adapter.

### F6: Gateway Degrades Gracefully During Store Restart

When store restarts, gateway's KV-backed endpoints (e.g., `/evidence/candles/latest`) may return errors or timeouts temporarily. However:
- Gateway's liveness endpoint (`/healthz`) remains responsive
- Gateway recovers automatically when store comes back
- Non-KV endpoints (analytical, via ClickHouse) are unaffected by store restart

### F7: Actor Redelivery Is Deterministic

The StrategyConsumerActor produces identical output for identical input. This means that JetStream redelivery (after MaxDeliver retries on un-ACKed messages) produces the same execution intent, enabling safe replay.

## Evidence Summary

| Claim | Test | Type |
|-------|------|------|
| Durable consumer spec stable | `TestS374_FailureIsolation_DurableConsumerSpecStable` | Structural |
| Trackers independent | `TestS374_FailureIsolation_IndependentTrackers` | Structural |
| Deterministic redelivery | `TestS374_FailureIsolation_ActorHandlesRedelivery` | Structural |
| Staleness guard on restart | `TestS374_FailureIsolation_StalenessGuardProtectsAfterRestart` | Structural |
| Tracker survives recreation | `TestS374_FailureIsolation_TrackerSurvivesActorRecreation` | Structural |
| Gate safety on KV unavailability | `TestS374_FailureIsolation_GateSafetyOnRestart` | Structural |
| Derive restart isolation | Smoke FI-1 | Compose |
| Execute restart isolation | Smoke FI-2 | Compose |
| Store restart isolation | Smoke FI-3 | Compose |
| Pipeline resumption | Smoke FI-4 | Compose |
| Stream integrity | Smoke FI-5 | Compose |
| Tracker isolation | Smoke FI-6 | Compose |

## Limitations

### L1: NATS Is Not Tested as a Failure Point

NATS is shared infrastructure. If NATS itself fails, ALL binaries degrade simultaneously. This is by design — NATS is the backbone, not a peer.

**Why acceptable:** NATS failure is an infrastructure concern, not a binary isolation concern. NATS itself has clustering and persistence guarantees for production.

**Residual risk:** A NATS connection timeout in one binary could trigger reconnect storms if multiple binaries detect the issue simultaneously.

### L2: Only Sequential Restarts

The smoke test restarts one binary at a time, waits for recovery, then restarts the next. Concurrent failures (e.g., derive AND execute fail simultaneously) are not tested.

**Why acceptable:** Concurrent binary failures are rare in practice and approach chaos engineering territory, which is explicitly out of scope for this wave.

**Residual risk:** Concurrent restarts could cause consumer group rebalancing or temporary stream subscription conflicts.

### L3: No Crash Simulation — Only Graceful Restart

`docker compose restart` sends SIGTERM with a 15-second grace period. This is a graceful restart, not a crash (SIGKILL). A crash may leave in-flight messages un-ACKed, which JetStream would redeliver.

**Why acceptable:** The structural test `ActorHandlesRedelivery` proves deterministic redelivery handling. The JetStream MaxDeliver=5 ensures bounded retry.

**Residual risk:** A crash during a ClickHouse INSERT could leave the writer's batch buffer in an inconsistent state (events ACKed to JetStream but not flushed to ClickHouse).

### L4: Writer Buffer Loss on Crash

Events that are ACKed by the writer's NATS consumer but not yet flushed to ClickHouse are lost on crash. This is a known architectural trade-off documented since S280.

**Why acceptable:** The writer uses batch flushes for performance. The window is small (configurable flush interval). The alternative (ACK-after-flush) would severely degrade throughput.

**Residual risk:** ClickHouse row counts may be slightly lower than NATS consumer delivered counts after a writer restart.

### L5: No Long-Duration Endurance

The smoke test restarts each binary once and verifies recovery. It does not run for extended periods to detect memory leaks, goroutine leaks, or slow degradation.

**Why acceptable:** Endurance testing is a separate concern. S374 focuses on isolation correctness, not sustained reliability.

### L6: Gateway Transient Errors During Store Restart

Gateway KV endpoints may return HTTP errors (500 or timeout) during the ~10-second window while store is restarting. This is expected behavior, not a bug.

**Why acceptable:** The gateway is a stateless proxy. Its liveness (`/healthz`) is maintained. KV endpoints recover automatically when store returns.

## What This Means for the Wave

S374 establishes that the multi-binary architecture provides meaningful failure isolation:
- Binaries can be restarted independently without contaminating others
- The pipeline resumes automatically after localized failures
- Stream integrity is maintained across restart cycles
- The control plane (gate) is durable across all binary restarts

Combined with S373 (E2E data flow) and S371–S372 (boundaries and wiring), the wave has now proven correctness, connectivity, AND resilience.
