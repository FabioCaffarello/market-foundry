# Next Wave Recommendations After Pre-Wave-B Gate

## Gate Outcome

The pre-Wave-B readiness gate **passed with constraints**. All three S156 preconditions are satisfied. The analytical layer is cleared for controlled expansion — not broad buildout.

---

## Recommended Wave B Scope

### What to expand

1. **One new read-path adapter and HTTP endpoint** for a second analytical table.
   - Suggested candidate: `signals` — simplest domain model after candles, 12 columns, clear analytical value (signal history queries for strategy tuning).
   - Alternative: `decisions` or `executions` — choose based on which has the first concrete consumer need.

2. **Extend smoke-analytical-e2e.sh** to cover the new family's complete data path.

3. **Wire smoke-analytical into CI** — this was deferred from S159 and should land before the second Wave B family.

### What to also address (small debts that fit naturally)

- **Backoff jitter**: Add random jitter to writer retry backoff. Trivial change, prevents thundering herd.
- **ClickHouse client timeout**: Make the 30s query timeout configurable via writer/gateway config.

### What NOT to do in Wave B

- Do not add all 5 remaining read-path families simultaneously.
- Do not introduce Prometheus, Grafana, or OpenTelemetry.
- Do not implement auto-recovery from degraded state.
- Do not add dead-letter queue or deduplication.
- Do not add backfill or cold-start bootstrap.
- Do not change operational baseline services.
- Do not add materialized views or secondary indexes.

---

## Wave B Sequencing

| Step | Description | Depends On |
|---|---|---|
| B1 | Add read-path adapter for second family (e.g., signals) | Gate pass |
| B2 | Add HTTP endpoint for second family with three-layer instrumentation | B1 |
| B3 | Extend smoke-analytical-e2e.sh to cover new family | B2 |
| B4 | Wire smoke-analytical into CI | B3 (or parallel) |
| B5 | Add backoff jitter + configurable query timeout | Independent |
| B6 | Validate expansion end-to-end | B3 |
| B7 | Wave B readiness review | B6 |

B1–B3 are sequential. B4 and B5 can run in parallel with B3. B6 depends on B3. B7 closes the wave.

---

## Wave B Success Criteria

Wave B succeeds when:

1. A second pipeline family has a read-path adapter, HTTP endpoint, and three-layer instrumentation matching candles' standard.
2. Integration test covers both candles and the new family.
3. CI runs smoke-analytical on every relevant change.
4. No regressions in existing tests or integration proof.
5. Expansion followed the documented pattern (adapter → interface → use case → handler → route → compose → test).

Wave B does NOT need to:

- Cover all 6 families.
- Introduce external observability.
- Achieve production-ready load testing.
- Implement any items from the "Not Worth the Cost Now" list.

---

## Wave B Anti-Patterns to Avoid

1. **Do not add multiple families simultaneously.** Validate the expansion pattern with one family first. If it works cleanly, subsequent families become mechanical.

2. **Do not skip tests for "simple" adapters.** Every read-path adapter interacts with DDL column layout. Every interaction can drift.

3. **Do not expand without observability parity.** No new endpoint without adapter-level timing, use-case logging, and HTTP Server-Timing header.

4. **Do not treat CI integration as optional.** Two consecutive stages (S159, S162) noted this gap. It must close in Wave B.

5. **Do not mix hardening with expansion.** Each stage should have a clear checklist. Work not on the checklist does not happen in that stage.

6. **Do not celebrate Wave B as "analytical layer complete."** Even after Wave B, the layer will remain a controlled projection with known limits. Full analytical capability is a future milestone, not a Wave B outcome.

---

## Post-Wave-B Horizon

If Wave B succeeds cleanly, the following become candidates for Wave C:

| Candidate | Trigger |
|---|---|
| Remaining read-path families (decisions, strategies, risks, executions) | Concrete consumer demand |
| NATS consumer lag visibility | Throughput increase or buffer overflow incidents |
| Load testing and performance baseline | Before third production deployment |
| Connection pool monitoring | Connection-related incidents |
| Schema coherence tooling (codegen or shared constants) | Table count exceeds 10 |

If Wave B reveals unexpected friction:

| Signal | Response |
|---|---|
| Expansion pattern does not transfer cleanly | Pause expansion, diagnose pattern failure |
| Schema drift between adapter and DDL | Invest in compile-time validation or codegen |
| Integration test becomes unreliable | Stabilize before adding more families |
| Read path latency unacceptable | Establish load testing before further expansion |

---

## Decision Record

| Decision | Rationale | Date |
|---|---|---|
| Wave B opened for controlled expansion | All S156 preconditions met; evidence supports expansion | 2026-03-19 |
| Scope limited to one new family per iteration | Validates pattern without multiplying risk | 2026-03-19 |
| CI integration required before second family | Two consecutive stages identified this gap; risk compounds | 2026-03-19 |
| External observability deferred | Single-operator scale; structured logs sufficient | 2026-03-19 |
| Auto-recovery deferred | Complexity exceeds benefit at small family count | 2026-03-19 |
