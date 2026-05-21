# Next Wave Recommendations After Analytical Wave A

> Actionable recommendations for what follows Wave A hardening. Based on the S156 readiness review.

## 1. Recommendation

**Proceed to Wave B with preconditions.** The analytical layer earned conditional expansion rights through Wave A, but three items must be resolved before or concurrently with the first Wave B deliverable.

This is not a pause, not an indefinite hardening extension, and not unconditional expansion. It is a controlled transition with explicit gates.

---

## 2. Wave B Preconditions (Must Complete Before Expanding)

These are the minimum prerequisites for adding any new pipeline family, table, or query endpoint.

### P1: Reader Path Minimum Instrumentation

**What**: Add structured logging and timing to the analytical read path.

**Where**: `cmd/gateway/analytical_reader.go`, `internal/application/analyticalclient/get_candle_history.go`, `internal/interfaces/http/handlers/analytical.go`.

**Minimum signals**:
- Query execution duration (ms) at reader adapter level.
- Row count returned per query.
- Error logging with query parameters (source, symbol, timeframe) at handler level.
- Request logging (INFO) at handler level.

**Why this blocks expansion**: Adding new query endpoints without instrumentation repeats the S150 mistake — expanding a capability that operators cannot see. Every new endpoint would inherit the same blind spot.

**Effort**: Small. Inject `*slog.Logger`, add 4–6 log statements, record timing.

### P2: One End-to-End Integration Test

**What**: A compose-based test that validates the full path: publish event to NATS → writer consumes → writer inserts to ClickHouse → reader queries → HTTP returns result.

**Where**: New test file or script, using existing `docker-compose.yaml` infrastructure.

**Why this blocks expansion**: Unit tests prove internal logic. They cannot prove that mapper output matches ClickHouse DDL, that NATS consumer config aligns with stream config, or that reader queries return data the writer actually inserted. Adding new families without this validation multiplies unverified contracts.

**Effort**: Medium. Requires compose startup, wait-for-ready loop, NATS publish, HTTP query, assertion.

### P3: Writer Config Validation at Startup

**What**: Validate writer configuration at startup and reject invalid values.

**Checks**:
- `batchSize` > 0.
- `maxPending` >= `batchSize` (otherwise buffer overflow triggers before batch threshold).
- `flushInterval` > 0.
- `maxRetries` >= 0.
- `initialBackoff` > 0 when `maxRetries` > 0.
- At least one pipeline family enabled.

**Why this blocks expansion**: New families will bring new configuration. If the validation surface is absent, misconfiguration failures will be silent and difficult to diagnose.

**Effort**: Small. Add validation function to `run.go`, call before engine start.

---

## 3. Wave B Scope Recommendation

### 3.1 What Wave B Should Add

Wave B should be a **controlled expansion** — not a broad buildout. Recommended scope:

1. **One new writer pipeline family** (suggested: `tradeburst` or `volume`, whichever has the simplest mapper and clearest analytical value).
2. **One new query endpoint** to exercise the reader path with a second table.
3. **Apply Wave A patterns to each addition**: tests first, observable, explicit failure semantics.

**Why only one family**: Adding one family validates the expansion pattern (new mapper, new consumer, new table, new endpoint) without multiplying risk. If the pattern works cleanly, subsequent families become mechanical.

### 3.2 What Wave B Should Also Address

These are Wave A debts that fit naturally into Wave B work without separate staging:

- **Backoff jitter**: Add random jitter to exponential backoff (trivial, prevents thundering herd).
- **ClickHouse client timeout configuration**: Make the 30s timeout configurable.
- **Consumer/supervisor message handling tests**: Test the paths that new families will exercise.

### 3.3 What Wave B Should NOT Address

These remain deferred:

- Dead-letter queue.
- Push-based alerting (Prometheus/Grafana).
- Auto-recovery from degraded state.
- Cold-start bootstrap from NATS replay.
- Per-family batch configuration.
- Materialized views or secondary indexes.
- Concurrent migration protection.
- Event schema versioning.

---

## 4. Wave B Sequencing Recommendation

| Step | Description | Dependencies |
|------|-------------|--------------|
| B0 | Resolve P1, P2, P3 (preconditions) | None — can start immediately |
| B1 | New migration for expansion table | P3 (config validation) |
| B2 | New mapper + consumer + inserter for one family | B1, P2 (integration test validates pattern) |
| B3 | New query endpoint for expansion table | B2, P1 (reader instrumented) |
| B4 | Validate expansion end-to-end | B3 |
| B5 | Wave B readiness review | B4 |

**B0 can overlap with early B1 planning** but must complete before B2 code lands.

---

## 5. Success Criteria for Wave B

Wave B succeeds when:

1. One new pipeline family operates with the same hardening standards as existing families (tests, retry, observable, recoverable).
2. One new query endpoint is instrumented (timing, error logging, row count).
3. The integration test passes with both old and new pipeline families.
4. Config validation rejects invalid writer configurations at startup.
5. Reader path has minimum instrumentation.
6. No regressions in existing families' test suites.

Wave B does NOT need to:
- Add all deferred families.
- Achieve full ClickHouse integration test coverage.
- Introduce external observability tooling.
- Implement dead-letter or deduplication mechanisms.

---

## 6. Anti-Patterns to Avoid

1. **Do not add multiple families simultaneously.** Validate the expansion pattern with one family first.
2. **Do not skip tests for "simple" mappers.** Every mapper interacts with DDL; every interaction can drift.
3. **Do not expand the query surface without reader instrumentation.** S150 already demonstrated this gap; repeating it wastes Wave A's hardening investment.
4. **Do not treat preconditions as optional.** P1/P2/P3 exist because their absence was the primary finding of this review.
5. **Do not extend hardening indefinitely.** The purpose of preconditions is to make expansion safe, not to prevent it. Once P1/P2/P3 are resolved, expansion should proceed.

---

## 7. Decision Summary

| Question | Answer |
|----------|--------|
| Is the analytical layer ready for Wave B? | Conditionally yes |
| What conditions must be met? | Reader instrumentation, integration test, config validation |
| How large should Wave B be? | One new family + one new endpoint |
| What should remain deferred? | DLQ, alerting, auto-recovery, schema versioning, materialized views |
| When can Wave B start? | Immediately — preconditions can be the first Wave B deliverables |
