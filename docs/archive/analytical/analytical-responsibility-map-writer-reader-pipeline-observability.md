# Analytical Responsibility Map — Writer, Reader, Pipeline, Observability

> Maps every Wave A gap to its owning component, responsible code path, current state, and target state.

---

## 1. Writer Responsibility Domain

### 1.1 Mappers (`cmd/writer/mappers.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| Type conversion correctness | Untested; assumed correct | Unit tests for all 6 mappers with edge cases |
| `parseFloat` error handling | Silent fallback to 0.0; error discarded | Log parse error with family + field + raw value; still return 0.0 (behavior preserved, visibility added) |
| `marshalJSON` error handling | Silent fallback to `"{}"` | Log marshal error with family + field; still return `"{}"` (behavior preserved, visibility added) |
| Nil/empty input handling | Undocumented; produces zero values | Tested and documented; behavior is explicit |

**Owner:** Front WC (tests) + Front FH (error visibility).

### 1.2 Inserter (`cmd/writer/inserter.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| Batch accumulation | Works; untested | Unit tests for size-triggered and time-triggered flush |
| Buffer overflow enforcement | Works; logs + metrics; untested | Unit test for FIFO eviction and counter increment |
| INSERT execution | Single attempt; buffer cleared on error (data loss) | Decision: implement retry OR accept single-attempt; buffer NOT cleared on error |
| Flush error recording | Counter exists; untested | Unit test for counter increment on flush failure |
| Batch latency measurement | Not measured | Structured log with INSERT duration per flush |

**Owner:** Front WC (tests) + Front FH (INSERT alignment) + Front OB (latency).

### 1.3 Consumer (`cmd/writer/consumer.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| NATS message deserialization | Works; untested | Unit test for valid and invalid message payloads |
| Message forwarding to inserter | Works; untested | Unit test for actor message delivery |
| Deserialization failure handling | `msg.Term()` + log (assumed, unverified) | Verified via test; counter for deser failures |
| Event counting | Not counted | `events_consumed` counter per family |

**Owner:** Front WC (tests) + Front OB (counter).

### 1.4 Supervisor (`cmd/writer/supervisor.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| Pipeline spawning | Works; untested | Unit test for pipeline startup sequence |
| Failure detection | Supervisor poisons itself on any pipeline error | Supervisor detects individual pipeline failure |
| Pipeline recovery | Not implemented; process restart is the only path | Restart failed pair with backoff (1s–30s, 5 attempts) |
| Degraded state management | Not implemented | Mark family as degraded after max restarts; visible via `/statusz` |
| Startup error handling | Poisons entire supervisor | Graceful degradation: start available families, log failed ones |

**Owner:** Front WC (spawn tests) + Front PR (recovery).

### 1.5 Pipeline Catalog (`cmd/writer/pipeline.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| Family declaration | Works; declarative; untested | Covered by supervisor tests |
| Family enable/disable | Works via config; untested | Covered by supervisor tests |

**Owner:** Front WC (indirect coverage via supervisor tests).

---

## 2. Reader Responsibility Domain

### 2.1 ClickHouse Reader Adapter (`cmd/gateway/analytical_reader.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| SQL query construction | Works; untested directly | Unit tests for query building with all param combinations |
| Row scanning and type conversion | Works; untested directly | Unit test for scan → domain struct conversion |
| `formatFloat` precision | Works; untested | Unit test for float64 → string round-trip accuracy |
| Error propagation to use case | Works; covered indirectly by use case tests | Direct unit test for reader error paths |
| Query latency measurement | Not measured | Structured log with query duration and row count |

**Owner:** Front WC (tests) + Front OB (latency).

### 2.2 ClickHouse Client (`internal/adapters/clickhouse/client.go`)

| Responsibility | Current State | Target State (Wave A) |
|---------------|---------------|----------------------|
| Connection management | Works; untested | Not in Wave A scope (infrastructure layer) |
| `InsertBatch` execution | Works; no retry | Retry logic added here or in inserter (Front FH decision) |
| `Ping` health check | Works; used by readiness | No change |

**Owner:** Front FH (retry placement decision).

### 2.3 Use Case and Handler (already tested)

| Component | Test Count | Status |
|-----------|-----------|--------|
| `analyticalclient/get_candle_history.go` | 7 unit tests | Adequate |
| `handlers/analytical.go` | 5 unit tests | Adequate |
| `routes/analytical.go` | 0 tests | Low priority (simple conditional) |

**Owner:** No Wave A work needed. Existing tests are sufficient.

---

## 3. Pipeline Responsibility Domain

### 3.1 End-to-End Flow

```
NATS JetStream
  → Consumer actor (deserialize, forward)
    → Inserter actor (batch, flush)
      → ClickHouse (INSERT)
        → Reader adapter (query)
          → Use case (validate, transform)
            → HTTP handler (serialize, respond)
```

| Segment | Tested? | Wave A Action |
|---------|---------|---------------|
| NATS → Consumer | No | WC: unit test |
| Consumer → Inserter | No | WC: unit test |
| Inserter → ClickHouse | No | WC: unit test + integration test |
| ClickHouse → Reader | No (directly) | WC: unit test |
| Reader → Use case | Yes (7 tests) | None |
| Use case → Handler | Yes (5 tests) | None |
| Full path | No | WC: 1 integration test |

### 3.2 Recovery Flow (not yet implemented)

```
Pipeline failure detected
  → Supervisor catches actor termination
    → Backoff wait (1s, 2s, 4s, 8s, 16s, 30s cap)
      → Respawn consumer-inserter pair
        → Success: resume normal operation
        → Max retries exceeded: mark family degraded
          → Log ERROR; update /statusz
          → Other families continue unaffected
```

**Owner:** Front PR (S154).

---

## 4. Observability Responsibility Domain

### 4.1 Write Path Counters (per family)

| Counter | Source | Current State | Wave A Target |
|---------|--------|--------------|---------------|
| `events_consumed` | Consumer actor | Not counted | New: increment on successful deser |
| `events_flushed` | Inserter actor | Exists (`flush_count` × batch size) | Verify accuracy; add explicit counter |
| `events_dropped` | Inserter actor | Exists (buffer overflow) | Verify; add mapper-error drops |
| `flush_errors` | Inserter actor | Exists | Verify accuracy under test |
| `mapper_errors` | Mapper functions | Not counted | New: increment on parseFloat/marshalJSON error |
| `batch_latency_ms` | Inserter actor | Not measured | New: time INSERT duration |
| `deser_errors` | Consumer actor | Recorded via `RecordError()` | Verify; ensure counter is query-able via `/statusz` |

### 4.2 Read Path Counters (per endpoint)

| Counter | Source | Current State | Wave A Target |
|---------|--------|--------------|---------------|
| `query_count` | Reader adapter or handler | Not counted | New: increment per query |
| `query_latency_ms` | Reader adapter | Not measured | New: time query duration |
| `rows_returned` | Reader adapter | Not counted | New: log row count per response |

### 4.3 Diagnostic Surface

| Surface | Current State | Wave A Target |
|---------|--------------|---------------|
| `/healthz` | Process alive (200) | No change |
| `/readyz` | NATS + ClickHouse reachable | No change |
| `/statusz` | Per-pipeline tracker stats | Add degraded family indicator; verify all counters exposed |
| `/diagz` | Detailed diagnostics | Verify analytical pipeline included |
| `diag-check.sh` | Exists; checks operational services | Extend to query writer `/statusz` and flag anomalies |
| Periodic summary log | Not implemented | New: every 60s, log consumed/flushed/dropped per family |

---

## 5. Responsibility Boundary Rules

These rules prevent scope creep within Wave A:

| Rule | Description |
|------|-------------|
| RB-01 | Mapper tests validate current behavior; they do not change return types or error signatures. |
| RB-02 | Inserter retry (if implemented) is bounded (max 5 attempts); no infinite retry. |
| RB-03 | Pipeline recovery is per-family; supervisor never restarts itself. |
| RB-04 | Observability uses structured logs only; no external metric systems. |
| RB-05 | Reader tests cover existing query surface only; no new endpoints. |
| RB-06 | Integration tests use docker-compose; no custom test harness. |
| RB-07 | No migration changes; schema is frozen during Wave A. |
| RB-08 | No changes to NATS subjects, streams, or consumer prefixes. |
| RB-09 | Health model changes are additive (new counters); existing endpoints are not restructured. |
| RB-10 | Config schema changes are minimal (recovery backoff params only if needed). |
