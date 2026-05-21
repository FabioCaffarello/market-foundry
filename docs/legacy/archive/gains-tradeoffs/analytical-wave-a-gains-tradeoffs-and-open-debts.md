# Analytical Wave A — Gains, Trade-offs, and Open Debts

> Formal accounting of what Wave A (S151–S155) delivered, what it traded away, and what remains unresolved.

## 1. Gains

### 1.1 Test Foundation (S152)

| What | Detail |
|------|--------|
| Mapper correctness | 25 tests proving column counts match DDL, metadata positions 0–3 consistent, edge cases (empty decimals, nil metadata, JSON roundtrips) handled |
| Inserter buffer logic | 10 tests proving FIFO eviction, correct eviction counts, tracker integration, nil safety, retry behavior |
| Reader query builder | 8 tests proving parameterized SQL structure, conditional time filters, argument ordering, float formatting precision |
| Coverage boundaries | Explicit documentation of what is and is not under test — no false confidence |

**Net gain**: Writer and reader unit correctness moved from "assumed" to "proven within stated boundaries."

### 1.2 Failure Handling Alignment (S153)

| What | Detail |
|------|--------|
| Buffer retention during retry | Critical data-loss bug fixed. Buffer cleared only after successful INSERT or all retries exhausted |
| Exponential backoff | 5-attempt retry with 1s–30s backoff. Absorbs ~15s transient outages without loss |
| Overflow counters | `events_overflowed` distinct from `flush_failures`. Every loss category has its own counter |
| Mapper fallback logging | parseFloat and marshalJSON log WARN with family/field/value context on zero-value injection |
| Configurable retry | `maxRetries` and `initialBackoff` exposed in writer config |

**Net gain**: Failure semantics now match architecture documentation. Code and docs agree.

### 1.3 Pipeline Recovery (S154)

| What | Detail |
|------|--------|
| Per-family restart | Consumer startup failure no longer kills all families. Supervisor manages individual restart |
| Exponential backoff | 2s, 4s, 8s, 16s, 30s — ~60s total before degraded |
| Degraded state | Terminal per process lifetime. `/statusz` emits `"degraded"` phase, `degraded_trackers` identifies affected families |
| Unaffected families continue | Healthy pipelines keep consuming and inserting while failed families restart |

**Net gain**: Localized failure no longer cascades. Recovery is bounded, observable, and per-family.

### 1.4 Observability (S155)

| What | Detail |
|------|--------|
| Buffer depth gauge | Real-time backpressure visibility per family |
| Flush total counter | Batch-level throughput tracking |
| Flush duration gauge | ClickHouse latency signal per flush |
| Events received counter | Inflow tracking enabling inflow-vs-outflow comparison |
| Degraded trackers array | Quick identification of degraded families in `/statusz` |
| Diagnostic script | Writer runtime included in `diag-check.sh` |
| Heartbeat counter snapshot | Idle warnings now include full counter context |
| Runbook | Operational playbooks for 6 scenarios with signal interpretation guide |

**Net gain**: Operator can answer "is data flowing?", "is backpressure building?", "are we losing data?", "which pipeline is degraded?" without reading code.

### 1.5 Discipline Gains

| What | Detail |
|------|--------|
| Scope freeze enforced | Zero new tables, endpoints, families, or infrastructure added during Wave A |
| Gap-driven sequencing | Each stage built on prior stage's explicit gap list — no speculative work |
| Honest accounting | Every stage documented what was NOT covered alongside what was delivered |
| No operational baseline contamination | All changes confined to writer, reader adapter, and health infrastructure |

---

## 2. Accepted Trade-offs

### 2.1 Unit Tests Over Integration Tests

**What was traded**: No end-to-end validation (NATS → ClickHouse → HTTP) exists.

**Why**: Integration tests require compose infrastructure, ClickHouse containers, and NATS streams. Adding this infrastructure was out of scope for Wave A, which focused on correctness boundaries rather than system-level validation.

**Consequence**: Unit tests prove internal logic but not inter-component contracts. A schema mismatch between mapper output and ClickHouse DDL would not be caught.

**Risk level**: Medium. Mitigated by column-count tests matching DDL, but not eliminated.

### 2.2 Sticky Degradation Over Auto-Recovery

**What was traded**: Degraded families never recover without process restart.

**Why**: Auto-recovery (cooling-period budget reset, ClickHouse health-triggered restart) adds complexity and risk of infinite restart storms. Simple mental model: 5 attempts, then stop.

**Consequence**: Extended outages require manual process restart to resume degraded families. Acceptable at current scale; may become painful with more families.

**Risk level**: Low at current scale. Will increase with Wave B expansion.

### 2.3 Pull-Only Observability Over Push Alerting

**What was traded**: No Prometheus, no Grafana, no push-based alerting.

**Why**: External observability tooling is infrastructure overhead disproportionate to current scale. Structured logs and HTTP endpoints are sufficient when operator count is small.

**Consequence**: Problem detection requires active polling or log watching. No automatic notification when pipelines degrade or data loss exceeds threshold.

**Risk level**: Low at current scale. Becomes a liability if operational team grows or SLA expectations emerge.

### 2.4 Mapper Fallbacks Over Rejection

**What was traded**: Invalid floats become 0.0, nil JSON becomes "{}". Rows are inserted with degraded data rather than rejected.

**Why**: Rejecting rows requires a dead-letter mechanism and operator workflow. Fallback injection keeps the pipeline flowing with degraded-but-queryable data.

**Consequence**: Analytical queries may silently include zero-value rows that don't represent real market data. Detection requires WARN log scanning.

**Risk level**: Low for current candle-only scope. Increases if decision/execution analytics depend on precision.

### 2.5 Actor-Blocking Retry Over Async Retry

**What was traded**: During retry backoff, the inserter actor is blocked (sleeping). No new rows are flushed during this window.

**Why**: Async retry adds concurrency complexity (multiple in-flight batches, ordering guarantees, buffer ownership). Blocking retry is simple and correct.

**Consequence**: Sustained ClickHouse latency causes buffer accumulation. If buffer exceeds maxPending during retry window, overflow eviction occurs.

**Risk level**: Low for transient failures. Medium for extended outages (>30s) with high throughput.

### 2.6 No Consumer–Inserter Backpressure

**What was traded**: Consumer continues ingesting from NATS regardless of inserter state. No flow control at the consumer→inserter boundary.

**Why**: Backpressure requires bidirectional signaling between actors, which contradicts the unidirectional message-passing model. Buffer overflow is the implicit backpressure mechanism.

**Consequence**: Under sustained inserter failure, buffer fills, overflow eviction begins, and consumer continues consuming — generating continuous data loss.

**Risk level**: Low at current throughput. Increases with higher-frequency data feeds.

---

## 3. Open Debts

### 3.1 Must Address Before or Early in Wave B

| # | Debt | Impact if Ignored | Effort |
|---|------|-------------------|--------|
| 1 | **Reader path has zero instrumentation** — no query timing, no error logging, no request counting | Read path problems invisible to operators; users discover failures before operators | Small (add slog + timing to reader/handler) |
| 2 | **No integration test** (NATS → writer → ClickHouse → reader → HTTP) | Inter-component contract violations undetectable until runtime | Medium (compose-based test harness) |
| 3 | **Writer config validation absent** — batchSize=0, maxPending=0, flushInterval=0 produce undefined behavior | Misconfiguration causes silent failures | Small (startup validation) |

### 3.2 Should Address During Wave B

| # | Debt | Impact if Ignored | Effort |
|---|------|-------------------|--------|
| 4 | **Backoff has no jitter** — multiple pipelines retry simultaneously after shared outage | Thundering herd on ClickHouse recovery | Trivial (add random jitter to backoff) |
| 5 | **Consumer/supervisor message handling untested** — handlePipelineFailure, handlePipelineRestart, consumer Started/Stopped | Recovery path logic only indirectly validated | Medium |
| 6 | **ClickHouse client timeout not configurable** — hard-coded 30s | Slow queries hang, no tuning possible | Small |
| 7 | **No NATS consumer lag visibility** — no way to tell how far behind consumer is | Lag buildup invisible until buffer overflow | Medium (requires JetStream admin API) |

### 3.3 Can Defer Without Risk

| # | Debt | Why Deferral is Safe |
|---|------|---------------------|
| 8 | Dead-letter queue for dropped batches | Events remain in NATS JetStream. Manual replay possible. Scale doesn't justify DLQ infrastructure |
| 9 | Auto-recovery from degraded state | Process restart resets budget. Current family count is small. Docker `restart: unless-stopped` covers crashes |
| 10 | Push-based alerting (Prometheus/Grafana) | Pull-only observability adequate for single-operator, paper-trading scale |
| 11 | Per-family batch configuration | Global config adequate while all families share similar throughput characteristics |
| 12 | Deduplication on retry | Duplicate inserts acceptable for analytical projection. MergeTree handles eventual cleanup |
| 13 | Reader response encoding error handling | JSON encode failure on candle response is extremely unlikely; partial response is detectable client-side |
| 14 | Concurrent migration protection | Single-operator deployment; no concurrent migration scenario exists |
| 15 | Cold-start bootstrap from NATS replay | Bounded by 90-day TTL; manual replay sufficient at current scale |

---

## 4. Debt Trajectory

Wave A reduced the debt load significantly:

| Category | Pre-Wave A (S150) | Post-Wave A (S156) |
|----------|-------------------|-------------------|
| Test coverage | Zero | 43 tests, explicit coverage boundaries |
| Failure handling | Silent data loss, code/doc divergence | Explicit retry, logged loss, counters |
| Pipeline recovery | Process-level only | Per-family, 5-attempt budget, degraded state |
| Observability | Health check only | 10 counters + 1 gauge + runbook + diag script |
| Reader instrumentation | None | None (unchanged — primary remaining gap) |
| Integration testing | None | None (unchanged — secondary remaining gap) |

The remaining debts are concentrated in two areas: **reader visibility** and **system-level validation**. These are the correct next targets, not expansion of the write path or addition of new infrastructure.
