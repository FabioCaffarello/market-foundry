# Analytical Wave A — Hardening Plan

> Master plan for S152–S156. Decomposes S150 gaps into responsibility-bounded work fronts with explicit sequencing, exit criteria, and expansion gates.

---

## 1. Executive Summary

The S143–S149 wave delivered a structurally correct analytical projection layer. The S150 readiness review confirmed that it is **not operationally reliable**: zero writer/reader tests, single-attempt INSERT with silent data loss, no pipeline recovery, no observability beyond health checks.

Wave A exists to close these gaps **without expanding scope**. No new tables, no new endpoints, no cold-start bootstrap. The goal is a small, tested, observable, recoverable analytical skeleton — not a bigger one.

This plan decomposes Wave A into 5 responsibility fronts, each mapped to a single stage (S152–S156), with strict sequencing where dependencies exist and parallelism where they don't.

---

## 2. Responsibility Fronts

| Front | ID | Owner Domain | Stage | Depends On |
|-------|----|-------------|-------|------------|
| Writer Correctness | WC | `cmd/writer/` | S152 | — |
| Failure Handling Alignment | FH | `cmd/writer/inserter.go`, `supervisor.go` | S153 | WC |
| Pipeline Recovery | PR | `cmd/writer/supervisor.go` | S154 | FH |
| Observability | OB | `cmd/writer/`, `cmd/gateway/` | S155 | WC |
| Expansion Gate | EG | Architecture review | S156 | WC, FH, PR, OB |

### 2.1 Front WC — Writer Correctness (S152)

**Objective:** Prove that the write path produces correct ClickHouse rows from NATS events.

**Scope:**
- Unit tests for all 6 mappers (edge cases: nil values, empty strings, zero decimals, malformed JSON).
- Unit tests for inserter batch logic (normal flush, timer flush, buffer overflow, INSERT error path).
- Unit tests for consumer deserialization and message forwarding.
- Unit tests for reader adapter query building and row scanning.
- At least 1 integration test: NATS event → ClickHouse row → gateway HTTP response.

**Exit criteria:**
- Every mapper function has tests covering happy path + at least 2 error/edge cases.
- Inserter flush paths (size-triggered, time-triggered, overflow) are individually tested.
- Reader adapter SQL construction is tested against expected output.
- Integration test proves end-to-end data path.

**Not in scope:** Changing mapper behavior (e.g., making `parseFloat` return errors). Tests validate current behavior first; behavior changes belong to FH.

### 2.2 Front FH — Failure Handling Alignment (S153)

**Objective:** Make the writer's actual failure behavior match its documented semantics, or update docs to match a deliberately simpler model.

**Decision required before implementation:**

| Option | Description | Trade-off |
|--------|-------------|-----------|
| A. Implement retry | Add exponential backoff (1s–30s, 5 attempts) to inserter flush as documented in `writer-service-failure-and-delivery-semantics.md` | More resilient; matches docs; moderate effort |
| B. Accept single-attempt | Keep current single-attempt INSERT; update architecture docs to reflect this as intentional | Simpler; honest; less resilient under transient failures |

**Scope (regardless of option chosen):**
- Resolve the code/docs divergence for INSERT retry.
- Fix buffer clearing on INSERT failure (current code clears buffer even on error — this is data loss, not a design choice).
- Add error returns or structured logging to `parseFloat` and `marshalJSON` so silent zero-value injection is visible.
- Ensure `msg.Term()` is called on deserialization failures (verify, not assume).

**Exit criteria:**
- Code and docs agree on INSERT failure behavior.
- Buffer is NOT cleared on INSERT failure (rows retained for retry or explicit drop).
- Mapper errors are logged with family, field name, and raw value.
- Deserialization failure path has a test.

**Depends on:** WC (tests must exist before changing behavior).

### 2.3 Front PR — Pipeline Recovery (S154)

**Objective:** Allow the writer supervisor to restart a failed consumer-inserter pair without restarting the entire process.

**Scope:**
- Supervisor detects individual pipeline failure (consumer or inserter actor stops unexpectedly).
- Supervisor restarts the failed pair with exponential backoff (1s, 2s, 4s, 8s, capped at 30s).
- After N consecutive restart failures for a single pipeline (configurable, default 5), supervisor marks the family as degraded and continues operating remaining families.
- Degraded family status is visible via `/statusz`.
- Supervisor no longer poisons itself on single-pipeline failure.

**Exit criteria:**
- Unit test: single pipeline failure → restart → resumed consumption.
- Unit test: repeated pipeline failure → degraded state → other pipelines unaffected.
- `/statusz` shows degraded families.
- Process-level restart is no longer the only recovery path.

**Depends on:** FH (failure handling must be aligned before adding recovery on top of it).

### 2.4 Front OB — Observability (S155)

**Objective:** Make the write and read paths observable enough to diagnose issues without log diving.

**Scope:**
- Per-family structured log counters on write path:
  - `events_consumed` — events received from NATS.
  - `events_flushed` — events successfully written to ClickHouse.
  - `events_dropped` — events lost to buffer overflow (already exists, verify).
  - `batch_latency_ms` — time per INSERT operation.
  - `mapper_errors` — count of parse/marshal fallbacks.
  - `flush_errors` — count of failed INSERT attempts (already exists, verify).
- Per-endpoint structured log counters on read path:
  - `query_count` — queries executed.
  - `query_latency_ms` — time per ClickHouse query.
  - `rows_returned` — rows per response.
- Periodic summary log (every 60s or configurable) per family: events consumed/flushed/dropped since last summary.
- Diagnostic script enhancement: `scripts/diag-check.sh` extended to query writer `/statusz` and report anomalies.

**Exit criteria:**
- Writer emits periodic structured summaries per family.
- Read path logs query latency and row count.
- `diag-check.sh` includes writer health assessment.
- No Prometheus, no Grafana, no external dependencies — structured logs only.

**Depends on:** WC (counters must be testable).

### 2.5 Front EG — Expansion Gate (S156)

**Objective:** Formal review confirming Wave A is complete and the system is ready for Wave B (controlled expansion).

**Scope:**
- Verify all WC/FH/PR/OB exit criteria are met.
- Run end-to-end validation: start pipeline, produce events, verify ClickHouse rows, query via gateway, kill ClickHouse, verify recovery.
- Document remaining debts that are acceptable for Wave B entry.
- Define Wave B preconditions and scope boundaries.
- Produce readiness verdict: expand / conditional expand / block.

**Exit criteria:**
- All 4 preceding fronts pass their exit criteria.
- End-to-end validation passes.
- Gate document produced with clear verdict.

**Depends on:** WC, FH, PR, OB (all fronts must be complete).

---

## 3. Sequencing

```
S152 (WC: Writer Correctness)
  │
  ├──→ S153 (FH: Failure Handling) ──→ S154 (PR: Pipeline Recovery)
  │
  └──→ S155 (OB: Observability)
                                          │
                                          ▼
                                    S156 (EG: Expansion Gate)
```

- **S152 is the foundation** — all other fronts depend on test coverage existing first.
- **S153 and S155 can proceed in parallel** after S152, but S153 is recommended first because FH changes affect what OB needs to observe.
- **S154 depends on S153** — recovery logic must be built on aligned failure semantics.
- **S156 is the gate** — nothing expands until all fronts are verified.

---

## 4. What Wave A Does NOT Include

See [analytical-wave-a-scope-blockers-and-non-goals.md](analytical-wave-a-scope-blockers-and-non-goals.md) for the full list. Summary:

- No new ClickHouse tables or migrations.
- No new query endpoints (signals, decisions, executions, etc.).
- No cold-start bootstrap.
- No schema evolution (ALTER migrations).
- No materialized views or pre-aggregations.
- No deferred writer families (tradeburst, volume, ema_crossover, venue_market_order).
- No Prometheus/Grafana/external monitoring.
- No per-family batch configuration.
- No concurrent migration protection.
- No deduplication infrastructure.

---

## 5. Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Hardening creeps into redesign | Medium | High — delays Wave A, introduces regression | Each front has explicit "not in scope"; review at stage boundaries |
| FH decision (retry vs. single-attempt) stalls | Low | Medium — blocks S153 and downstream | Decision is made at S153 start, not debated across stages |
| Integration tests require complex infra | Medium | Medium — slows S152 | Accept compose-based tests; do not build custom test harness |
| Observability front expands into metrics platform | Low | High — scope explosion | Structured logs only; no external dependencies |
| Recovery logic introduces new failure modes | Medium | Medium — supervisor becomes more complex | Bounded restart attempts; degraded state is the safety valve |

---

## 6. Success Criteria for Wave A

Wave A is complete when:

1. **All writer mappers have unit tests** with edge cases.
2. **Inserter batch logic is tested** (all flush paths, overflow, error).
3. **Reader adapter has unit tests** for query construction and row scanning.
4. **At least 1 integration test** proves NATS → ClickHouse → HTTP.
5. **INSERT failure behavior matches documentation** (code and docs agree).
6. **Buffer is not cleared on INSERT failure** (rows retained).
7. **Mapper errors are visible** in structured logs.
8. **Supervisor can restart individual failed pipelines** with backoff.
9. **Write path emits per-family structured counters**.
10. **Read path logs query latency and row count**.
11. **Expansion gate review passes** with clear verdict.

---

## 7. Blocking Criteria for Expansion

Wave B (controlled expansion) is **blocked** until:

- All 11 success criteria above are met.
- End-to-end validation (produce → write → query → fail → recover) passes.
- No critical or high-priority debts remain open from Wave A.
- Gate document (S156) issues a clear "expand" or "conditional expand" verdict.

If any success criterion is not met, expansion is blocked. There is no partial-expansion path.
