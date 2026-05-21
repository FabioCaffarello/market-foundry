# Durable Restart and Consumer Recovery Proof

**Stage:** S280
**Date:** 2026-03-21
**Status:** Proven

## Objective

Prove that the market-foundry paper flow maintains operational continuity when critical components are restarted. This is not a full fault-tolerance program — it validates the minimum recovery envelope required for confidence in operational stability.

## Architecture Summary

The system consists of stateless Go binaries communicating exclusively via NATS JetStream/KV and ClickHouse. All durable state lives in external stores:

| State | Location | Persistence |
|-------|----------|-------------|
| Stream events | NATS JetStream (file-backed) | Survives restart |
| Consumer position | NATS durable consumer | Survives restart |
| Control gate | NATS KV (`EXECUTION_CONTROL`) | Survives restart |
| Execution KV | NATS KV (`EXECUTION_PAPER_ORDER_LATEST`) | Survives restart |
| Analytical data | ClickHouse tables | Survives restart |
| Inserter buffer | Writer process memory | **Lost on restart** |

## Components Under Test

### 1. Writer (highest restart risk)
- **Consumer side:** 11 durable JetStream consumers with explicit ACK
- **Inserter side:** In-memory batch buffer (1000 rows or 5s flush interval)
- **Recovery mechanism:** Supervisor actor with exponential backoff (2s→4s→8s→16s→30s, max 5 retries)
- **Risk:** Buffer content lost on crash; events between last ACK and crash are redelivered

### 2. Execute
- **Consumer side:** Core NATS subscription for paper order intents
- **Safety gate:** Reads `EXECUTION_CONTROL` KV on every intent
- **Recovery mechanism:** Stateless — reconnects to NATS, re-reads gate

### 3. Store
- **Role:** KV projection and control gate mediator
- **Recovery mechanism:** Stateless — reconnects to NATS, recreates KV handles
- **Risk:** In-flight request/reply operations fail during restart

### 4. Gateway
- **Role:** HTTP API server
- **Recovery mechanism:** Stateless — reconnects to NATS and ClickHouse
- **Risk:** HTTP requests during restart return connection errors

## Proven Scenarios

### Adapter-Level (integration tests against real NATS)

| ID | Scenario | Test | Result |
|----|----------|------|--------|
| RR-1 | Durable consumer resumes from last ACK after restart | `TestRestartRecovery_DurableConsumer_ResumesFromLastACK` | Events published during downtime are delivered to restarted consumer; already-ACKed events are not redelivered |
| RR-2 | Control gate KV survives reconnect | `TestRestartRecovery_ControlGateKV_SurvivesReconnect` | Halted state persists across store close/reopen; write-after-reconnect works |
| RR-3 | KV projection persists across restart | `TestRestartRecovery_KVProjection_PersistsAcrossRestart` | Execution intent data survives store restart; monotonicity guard still enforced |
| RR-4 | Publisher reconnect delivers to consumer | `TestRestartRecovery_PublisherReconnect_DeliversToRestartedConsumer` | New publisher instance publishes to same stream; existing consumer receives |
| RR-5 | Full cycle: publish → restart → resume → no loss | `TestRestartRecovery_FullCycle_PublishRestartResumeNoLoss` | 10 events across 2 batches, consumer restart mid-stream, zero loss |
| RR-6 | Execute binary restart: safety gate re-reads KV | `TestRestartRecovery_ExecuteRestart_SafetyGateReReadsKV` | New SafetyGate instance reads correct halt/active state from KV |
| RR-7 | Dedup boundary: republished events idempotent | `TestRestartRecovery_DedupBoundary_RepublishedEventsIdempotent` | Same MsgID published twice within dedup window → single delivery |
| RR-8 | Multi-binary: derive restart, execute survives | `TestRestartRecovery_MultiBinary_DeriveRestartExecuteSurvives` | New derive binary publishes, execute (never restarted) receives from both |
| RR-9 | Writer consumer durable resumes stream position | `TestRestartRecovery_WriterConsumerDurable_ResumesStreamPosition` | ACKed events not redelivered; new events delivered on resume |
| RR-10 | Control gate cross-binary restart coherent | `TestRestartRecovery_ControlGate_CrossBinaryRestartCoherent` | Store binary restart → write active → derive reads active → execute reads active |

### Writer Pipeline Level (integration tests against real NATS)

| ID | Scenario | Test | Result |
|----|----------|------|--------|
| WR-1 | ConsumerStarter stop/restart resumes durable position | `TestWriterRestart_ConsumerStarter_ResumesFromDurablePosition` | Events published during downtime delivered via ConsumerStarter restart |
| WR-2 | Row mapping consistent across restart | `TestWriterRestart_RowMapping_ConsistentAcrossRestart` | mapExecutionRow produces identical 20-column structure before and after restart |
| WR-3 | Buffer loss boundary documented | `TestWriterRestart_BufferLoss_InFlightRedelivered` | ACKed events NOT redelivered (buffer loss is the gap, not stream loss) |
| WR-4 | Multiple restart cycles converge to correct total | `TestWriterRestart_MultipleCycles_ConvergesToCorrectTotal` | 3 restart cycles × mixed batch sizes = exact total (9 events) |

### Compose-Level (smoke script against running Docker stack)

| ID | Scenario | Script Phase | Result |
|----|----------|--------------|--------|
| RC-1 | Writer restart: durable consumer resumes | Phase 1 | Analytical counts non-decreasing after writer restart |
| RC-2 | Execute restart: safety gate re-initialized | Phase 2 | Execute readyz healthy; control gate unchanged |
| RC-3 | Store restart: KV projections queryable | Phase 3 | KV endpoints respond; gate write works after restart |
| RC-4 | Gateway restart: HTTP endpoints recover | Phase 4 | Readyz healthy; analytical and gate endpoints functional |
| RC-5 | Control gate survives store+gateway restart | Phase 5 | Gate status, reason, updated_by all persist across dual restart |
| RC-6 | Analytical projection continuity | Phase 6 | Candle and signal counts non-decreasing across all restart phases |

## Recovery Sequence Diagram

```
Time →  ─────────────────────────────────────────────────────────

Writer:   [RUNNING]──ACK──ACK──[CRASH]──────[RESTART]──ACK──ACK──
                                  │              │
NATS:     ──msg1──msg2──msg3──msg4──msg5──msg6──│──msg7──msg8──
                   ↑ACK    ↑ACK   ↑unACKed      │
                                                  ↓
                                          Resume from msg4
                                          (msg4 redelivered,
                                           msg5-msg6 delivered,
                                           msg1-msg3 NOT redelivered)

Buffer:   [m1,m2] → flush → [m3] → LOST     [m4,m5,m6] → flush
                                     ↑                      ↑
                               Buffer loss gap        New buffer
```

## Key Properties Proven

1. **No permanent stream loss:** NATS JetStream retains all unACKed messages. On restart, the durable consumer resumes from the last ACK position.

2. **Buffer loss is bounded:** Only the unflushed inserter buffer (up to `batchSize` rows or `flushInterval` seconds of data) is at risk. This is a known and documented gap.

3. **KV state is durable:** Both `EXECUTION_CONTROL` and `EXECUTION_PAPER_ORDER_LATEST` buckets use file-backed storage. All data persists across any component restart.

4. **Dedup prevents replay amplification:** JetStream MsgID deduplication (within the ~2 minute window) prevents the same event from being stored twice in the stream.

5. **Stateless binaries enable clean restart:** No binary carries state that cannot be reconstructed from NATS/ClickHouse. Restart is equivalent to a fresh start with the same durable consumer names.

## Files

### Tests
- `internal/adapters/nats/natsexecution/restart_recovery_test.go` — 10 adapter-level restart scenarios
- `internal/adapters/clickhouse/writerpipeline/restart_recovery_test.go` — 4 writer pipeline restart scenarios

### Scripts
- `scripts/smoke-restart-recovery.sh` — Compose-level restart smoke (6 phases)

### Documentation
- `docs/architecture/durable-restart-and-consumer-recovery-proof.md` — This document
- `docs/architecture/restart-recovery-semantics-and-operational-limits.md` — Semantics and limits
