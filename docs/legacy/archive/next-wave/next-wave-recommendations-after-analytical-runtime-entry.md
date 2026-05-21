# Next Wave Recommendations After Analytical Runtime Entry

> Produced as part of the S150 readiness review. Defines the conditional sequence for work following S143–S149.

---

## 1. Recommendation

**Harden the analytical runtime before expanding it.**

The S143–S149 wave delivered a viable skeleton. The skeleton has structural integrity (optionality, separation, minimal schema) but lacks operational confidence (no tests on critical paths, fragile failure handling, no observability). Expanding now — adding tables, endpoints, or cold-start bootstrap — would amplify risks that are currently contained.

---

## 2. Candidate Waves and Sequencing

Four candidate waves were evaluated:

| Wave | Description | Verdict |
|------|-------------|---------|
| **A. Hardening** | Test coverage, failure handling, observability | **Do first** |
| **B. Controlled expansion** | Additional query endpoints, deferred writer families | Do second, conditionally |
| **C. Cold-start bootstrap** | Derive queries ClickHouse on startup for warm state | Defer — high complexity, optionality boundary risk |
| **D. Deliberate pause** | No analytical work; focus elsewhere | Not recommended — hardening debt compounds if left |

### 2.1 Wave A — Analytical Runtime Hardening (recommended next)

**Objective:** Make the existing skeleton reliable and observable before adding surface area.

**Scope:**

| Item | Deliverable | Priority |
|------|-------------|----------|
| Writer mapper tests | Unit tests for all 6 mappers | High |
| Writer inserter tests | Unit tests for batch logic, buffer overflow, flush timing | High |
| Writer supervisor tests | Unit test for pipeline lifecycle, failure propagation | High |
| Writer integration test | NATS → ClickHouse end-to-end flow validation | High |
| Reader adapter tests | Unit tests for query building and row scanning | High |
| Pipeline recovery | Supervisor restarts failed consumer-inserter pairs with backoff | High |
| INSERT retry alignment | Either implement documented retry-with-backoff or update docs to match single-attempt behavior | High |
| Buffer overflow visibility | Counter for evicted events per family; structured log with count | Medium |
| Mapper error visibility | Counter for parse/marshal failures per family | Medium |
| Write-path observability | Per-family counters: events_consumed, events_flushed, events_dropped, batch_latency | Medium |
| Migration runner integration test | Verify Up/Status/Validate against real ClickHouse | Medium |

**Exit criteria:**
- All writer mappers have unit tests with edge cases (nil values, empty strings, zero decimals).
- Inserter batch logic tested: normal flush, timer flush, buffer overflow, INSERT failure.
- At least one integration test proves NATS event → ClickHouse row → gateway HTTP response.
- Pipeline recovery demonstrated: family failure → automatic restart → resumed consumption.
- INSERT failure behavior matches architecture documentation.
- Buffer overflow and mapper errors are countable in structured logs.

**What this wave does NOT include:**
- No new tables or migrations.
- No new query endpoints.
- No cold-start bootstrap.
- No schema evolution (ALTER migrations).
- No materialized views.

### 2.2 Wave B — Controlled Schema and Query Expansion (conditional on A)

**Precondition:** Wave A hardening complete. Writer and reader paths tested. Failure handling aligned with docs.

**Objective:** Expand the analytical query surface to cover the remaining core tables.

**Candidate scope (prioritized by analytical value):**

| Item | Value | Effort |
|------|-------|--------|
| Signal history endpoint (`/analytical/signals`) | Enables signal quality analysis over time | Low — pattern established |
| Decision history endpoint (`/analytical/decisions`) | Enables decision outcome analysis | Low |
| Execution history endpoint (`/analytical/executions`) | Enables fill tracking and slippage analysis | Low |
| Strategy history endpoint (`/analytical/strategies`) | Lower priority — strategies change infrequently | Low |
| Risk assessment history endpoint (`/analytical/risk-assessments`) | Lower priority — exposure tracking | Low |
| Deferred writer families (tradeburst, volume, ema_crossover, venue_market_order) | Only if analytical consumers exist | Medium |
| Deferred tables (tradebursts, volumes, fills) | Only if query endpoints justify them | Medium |

**Sequencing within Wave B:**
1. Add query endpoints for existing tables first (no schema change needed).
2. Add deferred writer families only for tables that have both write and read paths.
3. Add deferred tables only when a concrete analytical question requires them.

**Exit criteria:**
- Each new endpoint follows the established pattern (conditional registration, 503 fallback, max limit, time range).
- Each new endpoint has unit tests for use case and handler.
- Each new reader adapter has unit tests.

### 2.3 Wave C — Cold-Start Bootstrap (conditional on B, high complexity)

**Precondition:** Wave B complete. Multiple query endpoints proven. Schema stable.

**Objective:** Allow `derive` to query ClickHouse on startup to warm its state from historical data instead of waiting for live events.

**Why defer:**
- This is the first cross-boundary dependency between operational and analytical layers.
- It risks violating optionality rule R-09 (cold-start must be opportunistic, non-blocking fallback).
- The derive service currently works without historical data — it derives from live events.
- The value is real (faster warm-up after restart) but the risk is structural contamination.

**When to reconsider:**
- When the pipeline restarts frequently enough that cold-start delay is operationally painful.
- When the analytical schema is stable enough that derive can reliably query it.
- When the optionality boundary can be enforced at the interface level (derive depends on a `HistoryProvider` interface, not on ClickHouse directly).

### 2.4 Wave D — Deliberate Pause (not recommended)

A deliberate pause would mean doing no analytical work and focusing on other pipeline areas. This is not recommended because:

- The hardening debt in Wave A is small (primarily test coverage and failure handling alignment).
- Leaving the writer untested while it runs in the background accumulates silent data quality risk.
- The gap between documented and actual failure semantics will confuse future contributors.

However, if other pipeline priorities are more urgent, a short pause (1–2 stages) is acceptable as long as the writer is either:
- Tested and monitored during the pause, or
- Disabled in docker-compose until hardening is done.

Running an untested writer silently in the background while focusing elsewhere is the worst outcome.

---

## 3. Anti-Patterns to Avoid

| Anti-pattern | Why it's tempting | Why it's wrong |
|--------------|-------------------|----------------|
| Skip hardening, jump to more endpoints | "The pattern works, just replicate it" | Replicating an untested pattern amplifies untested surface area |
| Add cold-start bootstrap now | "It's the highest-value analytical feature" | It crosses the operational/analytical boundary; premature coupling |
| Add materialized views for aggregation | "ClickHouse is designed for this" | No query patterns justify aggregation yet; premature optimization |
| Expand schema before testing write path | "More data = more useful" | More data through an untested writer = more potential silent corruption |
| Treat hardening as optional polish | "Tests can come later" | The writer is running and dropping data silently; this is not polish |

---

## 4. Decision Matrix

| Question | Answer | Implication |
|----------|--------|-------------|
| Is the analytical runtime functional? | Yes | Skeleton is viable |
| Is the analytical runtime reliable? | Not proven | Needs test coverage |
| Is the analytical runtime observable? | Minimally | Needs counters and structured diagnostics |
| Should we expand the schema now? | No | Harden first |
| Should we add query endpoints now? | No | Harden first |
| Should we attempt cold-start bootstrap? | No | Defer until Wave B is proven |
| Should we pause all analytical work? | No | Hardening is small and the debt compounds |
| Should we disable the writer until hardening? | Consider it | Running untested code silently is worse than not running it |

---

## 5. Summary

The analytical runtime entry was disciplined and well-bounded. The next wave should match that discipline by hardening what exists before expanding what is offered. The sequence is: **harden → expand → bootstrap**, with each step conditional on the previous one being proven.
