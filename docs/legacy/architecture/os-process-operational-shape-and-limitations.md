# OS-Process Operational Shape and Limitations

**Stage**: S279
**Date**: 2026-03-21

## Operational Shape

### Services in the Proven Shape

| Service | Role | Process Isolation | Communication |
|---------|------|-------------------|---------------|
| **nats** | Message broker, JetStream streams, KV buckets | Container (alpine) | TCP 4222 |
| **clickhouse** | Time-series analytical storage | Container (official image) | TCP 9000 (native), 8123 (HTTP) |
| **configctl** | Binding lifecycle management | Container (Go binary) | NATS req/reply |
| **ingest** | Market data capture (Binance WS) | Container (Go binary) | NATS publish |
| **derive** | Signal → decision → strategy → risk → execution chain | Container (Go binary) | NATS consume + publish + KV read |
| **store** | KV materialization + query responder | Container (Go binary) | NATS consume + KV write + req/reply |
| **execute** | Venue submission with safety gate | Container (Go binary) | NATS consume + KV read + publish |
| **writer** | NATS → ClickHouse batch inserter (10 pipelines) | Container (Go binary) | NATS consume + ClickHouse native |
| **gateway** | HTTP API surface | Container (Go binary) | HTTP + NATS req/reply + ClickHouse query |

### Data Planes

1. **Event Plane** (NATS JetStream): observation → evidence → signal → decision → strategy → risk → execution → fill
2. **State Plane** (NATS KV): control gate, latest projections per family per symbol/timeframe
3. **Analytical Plane** (ClickHouse): append-only time-series storage, queried by gateway
4. **Control Plane** (HTTP → NATS → KV): operator actions via gateway API

### What This Shape Proves

- **Process crash isolation**: A crash in `execute` does not affect `derive`, `store`, or `writer`.
- **Shared-nothing communication**: All inter-service data flows through NATS or ClickHouse; no shared memory, files, or Unix sockets.
- **Kill switch propagation**: An HTTP PUT to gateway sets KV state that `derive` and `execute` observe independently, with no direct coupling between the three processes.
- **Analytical observability**: Events produced by `derive` are consumed by `writer` (separate process), persisted to `clickhouse` (separate process), and queried through `gateway` (separate process) — four process boundaries in the read path.

## Limitations

### L1: External Data Dependency

The pipeline requires live Binance WebSocket data to produce candle observations. Without internet connectivity or during exchange maintenance, no new data flows through the pipeline. The smoke test handles this gracefully by accepting zero-count results for downstream families.

**Impact**: The smoke cannot be run in fully air-gapped CI without a mock data source.
**Mitigation**: Evidence candles (ingest → derive) are the first to appear; downstream families (signals, decisions, etc.) depend on market conditions triggering their thresholds.

### L2: Execution Data is Condition-Dependent

Paper order executions only appear when the RSI oversold decision fires AND the strategy resolver produces a long entry AND the risk evaluator approves. This may not happen during the smoke window (120s default).

**Impact**: OP-4 (halt propagation via execution count delta) may show `delta=0` both during halt and after resume, making it a non-distinguishing test.
**Mitigation**: The control gate round-trip (OP-3) is proven independently of execution data. OP-4 is a best-effort strengthening.

### L3: Writer Batch Window Creates Observation Lag

The writer flushes to ClickHouse every 5 seconds (configurable). Events produced just before a halt may appear in ClickHouse after the halt is set, creating a small false-positive window.

**Impact**: Up to 2 executions may appear during the halt window from pre-halt queue flush.
**Mitigation**: The smoke script accepts `delta <= 2` as PASS with a warning. The actual gate enforcement is at the `derive` and `execute` publish/submit boundary, not at the writer.

### L4: No Crash/Restart Recovery Proof

S279 does not test service crashes, container restarts, or JetStream consumer redelivery after process death. All services remain running throughout the smoke.

**Impact**: Consumer durability across process restart is unproven (OD-OH6 from S278 remains open).
**Mitigation**: This is explicitly out of scope per the S279 charter. A future stage should kill a container mid-flow and verify recovery.

### L5: Single NATS Node

The compose stack runs a single NATS server. Cluster behavior (leader election, route failover, KV replication) is not tested.

**Impact**: Production deployments with NATS clusters may exhibit different KV propagation semantics.
**Mitigation**: Acceptable for paper trading validation. Cluster proofs belong to a production-readiness stage.

### L6: No Concurrent Writer Enforcement

The sole-writer constraint for KV buckets is by convention. Nothing prevents two `store` instances from writing to the same KV bucket simultaneously.

**Impact**: Multi-replica deployments could produce KV conflicts.
**Mitigation**: The current compose stack runs exactly one instance of each service. Multi-replica safety is a future concern.

### L7: Gateway Port Collision in Compose

In the Docker Compose stack, services expose ports on localhost. The smoke script accesses services via `127.0.0.1:{port}`. This works because each service has a unique port mapping.

**Impact**: Port conflicts with host-side services could cause false failures.
**Mitigation**: The compose file uses deterministic port mappings (8080-8085) and health checks validate service identity.

### L8: No TLS or Authentication

All NATS connections and HTTP endpoints are unencrypted and unauthenticated in the compose stack.

**Impact**: Not representative of a production security posture.
**Mitigation**: Acceptable for functional smoke validation. Security hardening is a separate concern.

## Synchronization Assumptions

| Component | Assumption | Confidence |
|-----------|-----------|------------|
| NATS KV | Read-after-write consistent within single node | High (documented guarantee) |
| JetStream | At-least-once delivery with explicit ack | High (configured) |
| ClickHouse | Batch insert visible after flush | High (writer confirms flush) |
| Docker health checks | Service ready when `/readyz` returns | High (tested pattern) |
| Gate propagation | Derive/execute read gate on every operation | High (code-verified, S275/S276) |
| Writer deduplication | ClickHouse ReplacingMergeTree handles re-delivery | Medium (depends on merge timing) |

## Fragilities

1. **Timing sensitivity**: The 120s flush wait is empirical. Under heavy load or slow CI runners, writer may need more time.
2. **Market hours dependency**: Some markets have reduced activity during weekends/holidays, reducing event flow.
3. **ClickHouse merge timing**: COUNT queries may include duplicates if ReplacingMergeTree hasn't merged yet (counts may be slightly higher than expected).
4. **Docker Compose resource limits**: ClickHouse has a 4GB memory limit in compose; large batch windows could cause OOM under unusual conditions.
