# Post-Wave A Analytical Readiness Review

> Stage S156 formal review. Covers S151–S155 as the Wave A hardening cycle. Determines whether the analytical layer is ready for Wave B expansion.

## 1. Executive Summary

Wave A hardening delivered substantive, evidence-backed improvements to the analytical layer. The writer service moved from zero test coverage to 43 new test cases, from silent data loss to explicit retry semantics, from process-level-only recovery to supervisor-managed per-family restart, and from near-zero observability to minimal-but-useful diagnostic signals. Every stage built on the prior stage's explicit gap accounting, and scope freeze was enforced throughout: no new tables, no new endpoints, no new pipeline families.

However, "hardened" is not "proven." The improvements are real but operate within narrow boundaries: unit tests cover correctness but not integration, retry semantics are explicit but never validated under real ClickHouse failure, pipeline recovery works in isolation but was never exercised under production-like load, and observability signals exist but the read path has zero instrumentation. The analytical layer improved from "valid projection" to "minimally disciplined skeleton" — not yet to "reliably operational capability."

**Readiness verdict: the analytical layer is ready for controlled Wave B expansion with explicit preconditions, not unconditional expansion.** The hardening addressed the most critical gaps but left several structural debts that should be resolved early in Wave B rather than deferred further.

---

## 2. Review Criteria and Findings

### 2.1 Do writer and reader adapter now have a sufficient confidence base?

**Verdict: Partial — writer improved significantly, reader adapter remains undertested.**

#### Writer

Evidence of improvement:
- **Mappers**: 25 tests covering all 6 event types, column count verification against DDL, metadata position consistency, edge cases (empty decimals → 0.0, nil metadata → "{}"), JSON roundtrip correctness.
- **Inserter**: 10 tests covering FIFO buffer eviction, tracker metric recording, nil safety, retry behavior (transient failure recovery, exhaustion drop, buffer retention during retries).
- **Supervisor**: 4 tests for backoff calculation, lifecycle state transitions (active → restarting → degraded).

Remaining gaps:
- No integration test for the full NATS → ClickHouse write path.
- Consumer actor message handling is untested (deserialization, forwarding, failure reporting).
- Supervisor message handling (handlePipelineFailure, handlePipelineRestart) is not directly tested.
- Actor lifecycle (Started/Stopped) and timer-based flush are not tested.
- Config validation is absent (batchSize=0, maxPending=0, flushInterval=0 produce undefined behavior).

#### Reader Adapter

Evidence of improvement:
- 8 new tests for query builder (parameterized SQL structure, conditional time filters, argument ordering, float formatting precision).
- Existing 7 use case tests + 5 handler tests remain adequate.

Remaining gaps:
- Zero logging or instrumentation in the entire read path (reader, client, handler).
- No integration test against real ClickHouse.
- Row scanning, type conversion, and context cancellation are not tested.
- ClickHouse client has no timeout configuration (hard-coded 30s per operation).
- Errors lose detail as they propagate through layers.

**Assessment**: Writer confidence moved from "zero" to "reasonable for unit correctness." Reader confidence is structurally sound but operationally invisible — problems in the read path would be discovered by users, not by operators.

### 2.2 Is failure handling coherent with the architecture?

**Verdict: Yes — the three critical divergences identified in S150/S151 were resolved.**

What was fixed:
1. **Buffer-clear-on-error bug** (critical data loss): Buffer is now retained during retry. Only cleared after successful INSERT or all retries exhausted. Tests prove the fix.
2. **INSERT retry semantics**: Exponential backoff implemented (1s initial, 30s cap, 5 attempts configurable). Matches the documented architecture.
3. **Mapper error visibility**: parseFloat and marshalJSON now log WARN with family/field/value context on fallback injection.

What the fix means operationally:
- A transient ClickHouse outage of ~15s (5 retries with backoff) is absorbed without data loss.
- Beyond the retry window, data loss is explicit: ERROR log + `events_dropped` + `flush_failures` counters.
- Mapper fallbacks still insert degraded data (0.0 for invalid floats, "{}" for nil JSON) — visible via WARN logs but not rejected.

Remaining coherence gaps:
- No dead-letter queue: batches lost permanently after retry exhaustion. Events remain in NATS JetStream but no automatic replay.
- Retry blocks the actor during backoff sleep, limiting throughput during extended outages.
- No backpressure between consumer and inserter: consumer continues ingesting during inserter retry, potentially exacerbating buffer overflow.
- Exponential backoff has no jitter: multiple pipelines retry simultaneously after a shared outage (thundering herd potential).
- No deduplication on retry: possible duplicate inserts on network partition (acceptable for analytical projection).

### 2.3 Are overflow and potential loss operationally visible and acceptable?

**Verdict: Yes — every loss category is now logged and counted.**

Loss visibility:
| Category | Signal | Level |
|----------|--------|-------|
| Buffer overflow (FIFO eviction) | `events_overflowed` counter + ERROR log | Immediate |
| Retry exhaustion (batch drop) | `flush_failures` counter + `events_dropped` + ERROR log | Immediate |
| Mapper fallback (degraded insert) | WARN log with field/value context | Per-row |
| Deserialization failure | WARN log (event stays in NATS) | Per-event |
| Pipeline degraded | `pipeline_degraded` counter + `/statusz` phase + ERROR log | Per-family |

Acceptability assessment:
- At current scale (paper-trading, small symbol set), bounded loss with visibility is appropriate.
- The loss semantics are clearly documented and align with "analytical projection, not source of truth."
- Operators can detect loss within one diagnostic cycle (polling `/statusz` or reading structured logs).
- No automatic escalation or alerting exists — detection requires active monitoring.

Gap: loss is visible but not actionable without manual intervention. No circuit breaker pauses ingestion when ClickHouse is durably unavailable.

### 2.4 Do analytical pipelines fail and recover in a better-delimited way?

**Verdict: Yes — significant improvement over pre-Wave A behavior.**

Before Wave A:
- Any consumer startup failure poisoned the entire supervisor → process-level restart was the only recovery path.
- All pipeline families died together regardless of which one failed.

After Wave A:
- Consumer startup failure → `pipelineFailedMsg` → supervisor manages per-family restart.
- Exponential backoff: 2s, 4s, 8s, 16s, 30s (5 attempts, ~60s to degraded).
- Unaffected families continue operating.
- Degraded state is terminal per process lifetime (requires restart to reset budget).
- `/statusz` emits `"degraded"` phase with `degraded_trackers` array identifying affected families.

Remaining limitations:
- **Sticky degradation**: Once a family is degraded, it cannot recover until process restart. No cooling-period budget reset.
- **Inserter failures are not supervisor-managed**: Only consumer startup failures trigger supervisor recovery. Inserter-level failures (ClickHouse permanently down) are handled by internal retry+drop only.
- **Unexpected actor death (panics)** is not supervisor-detected (Hollywood framework handles this separately).
- **No automatic recovery trigger**: If ClickHouse comes back online, degraded families remain degraded until process restart.

### 2.5 Is minimum observability sufficient for the current stage?

**Verdict: Sufficient for the writer. Insufficient for the reader.**

Writer observability:
- 7 tracker counters per family: `events_received`, `events_flushed`, `flush_total`, `flush_failures`, `events_dropped`, `events_overflowed`, `buffer_depth`.
- 2 supervisor counters: `pipeline_restarts`, `pipeline_degraded`.
- 1 latency gauge: `flush_duration_ms`.
- Structured logs at DEBUG (flush success), WARN (retry attempt, pipeline failure, idle), ERROR (overflow, exhaustion, degraded).
- `/statusz` and `/diagz` expose all counters. `/healthz` and `/readyz` confirm liveness and dependency reachability.
- `diag-check.sh` now includes writer.

Reader observability:
- **Zero instrumentation**. No query timing, no row counting, no error logging, no request tracing.
- If the read path degrades (slow queries, schema mismatch, connection pool exhaustion), operators have no signal until users report failures.
- This is the single most significant observability gap remaining after Wave A.

### 2.6 Is the analytical layer ready for Wave B?

**Verdict: Conditionally ready. Three preconditions must be met before expanding.**

The analytical layer demonstrated that:
1. The write path can be hardened without contaminating the operational baseline.
2. Failure semantics can be made explicit and observable.
3. Pipeline recovery can be bounded and per-family.
4. The architecture supports incremental improvement without redesign.

But Wave B expansion (new tables, new families, new endpoints) should not proceed until:

1. **Reader observability**: The read path needs minimum instrumentation (query timing, error logging) before adding new query endpoints. Expanding query surface without visibility is reckless.
2. **Integration test**: At least one end-to-end test (NATS → writer → ClickHouse → reader → HTTP) must validate the full path before adding new paths to it.
3. **Config validation**: Writer startup should reject invalid configurations (batchSize=0, maxPending < batchSize, flushInterval=0) before those configurations silently break new pipeline families.

These are scoped, achievable items — not an invitation to extend hardening indefinitely.

---

## 3. Expansion Blocker Assessment

The 11 expansion blockers defined in S151:

| # | Blocker | Status | Evidence |
|---|---------|--------|----------|
| 1 | Writer mapper unit tests | **Cleared** | 25 tests in mappers_test.go |
| 2 | Inserter batch logic tests | **Cleared** | 10 tests in inserter_test.go |
| 3 | Reader query builder tests | **Cleared** | 8 tests in analytical_reader_test.go |
| 4 | INSERT failure handling alignment | **Cleared** | Retry with backoff implemented, buffer retention proven |
| 5 | Buffer-clear-on-error fix | **Cleared** | Buffer retained during retry, test proves fix |
| 6 | Mapper error visibility | **Cleared** | WARN logs with field/value context |
| 7 | Pipeline recovery with backoff | **Cleared** | Supervisor-managed, 5-attempt budget, per-family |
| 8 | Per-family degraded state | **Cleared** | `pipeline_degraded` counter, `/statusz` phase, `degraded_trackers` |
| 9 | Write-path structured counters | **Cleared** | 7 counters + 1 latency gauge per family |
| 10 | Diagnostic script includes writer | **Cleared** | `diag-check.sh` queries writer:8085 |
| 11 | Integration test (NATS → CH → HTTP) | **Not cleared** | No integration test exists |

**Result: 10 of 11 blockers cleared.** The integration test blocker remains open. This is a genuine gap — not a technicality.

---

## 4. Honest Assessment — What Wave A Did and Did Not Prove

### What Wave A proved:
- The writer service can be tested, hardened, and made observable without architectural changes.
- Failure semantics can be made explicit without introducing external dependencies.
- Pipeline recovery can be bounded and per-family without redesigning the actor hierarchy.
- The operational baseline remained uncontaminated throughout hardening.
- Scope freeze discipline held: no feature creep, no expansion, no new infrastructure.

### What Wave A did not prove:
- That the write path works correctly under real ClickHouse failure scenarios (no integration test).
- That the read path is operationally reliable (zero instrumentation).
- That retry+backoff handles extended outages gracefully (theoretical, not validated).
- That pipeline recovery works under concurrent multi-family failure (not tested).
- That the analytical layer performs adequately under sustained load (no load testing).
- That mapper fallbacks produce analytically useful data rather than noise.

### What remains aspirational rather than proven:
- "At-least-once delivery" is documented but never validated end-to-end.
- "Bounded loss with visibility" is architecturally correct but operationally unproven.
- "Supervisor-managed recovery" works in unit tests but never under real failure conditions.

---

## 5. Verdict

**The analytical layer has moved from "valid projection" to "minimally hardened skeleton."** This is genuine progress — the gap between S150 and S156 is substantive and evidence-based.

The layer is **conditionally ready for Wave B** with three preconditions:
1. Reader path minimum instrumentation (query timing, error logging).
2. One end-to-end integration test.
3. Writer config validation at startup.

These preconditions are scoped to prevent the same class of blind-spot that Wave A itself was created to fix: expanding a capability before its existing surface is operationally visible.

Wave B should focus on **controlled expansion** — one or two new pipeline families, one new query endpoint — not a broad buildout. The hardening patterns established in Wave A (test-first, observable, explicit failure semantics) should be applied to each new addition as it is introduced, not deferred for a future Wave C.
