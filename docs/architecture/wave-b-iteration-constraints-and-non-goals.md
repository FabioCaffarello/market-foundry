# Wave B Iteration Constraints and Non-Goals

> Formal boundaries for Wave B expansion. These constraints are inherited from S162 gate approval
> and supplemented with operational guardrails derived from Wave A experience.

## S162 Inherited Constraints

These constraints are non-negotiable. They were established as conditions of Wave B approval.

### C-1: One Family Per Iteration

Each iteration introduces exactly one new analytical family and one new HTTP query endpoint. No iteration may add two families simultaneously, regardless of perceived simplicity.

**Rationale:** Controlled blast radius. Each family touches schema, writer, reader, gateway, and tests. Combining families multiplies the risk of schema coherence errors and makes regression attribution harder.

### C-2: Pattern Discipline

Every new family must meet the same hardening standard established by the candle family during Wave A. This includes:
- Unit tests for mapper, reader, and handler.
- Schema coherence verification (DDL ↔ mapper ↔ reader).
- Observability coverage (inserter counters, reader timing, handler Server-Timing).
- Smoke test coverage (write path and read path).

No family ships with "we'll add tests later" or "observability comes in the next iteration."

### C-3: CI Integration Before Second Family

The `smoke-analytical-e2e.sh` script must be integrated into CI before the second Wave B family can begin. The first family may proceed without CI integration, but the gate review for the first family must confirm CI integration is scheduled or in progress.

**Rationale:** Manual smoke execution does not scale. By the second family, automated regression detection is mandatory.

### C-4: No External Infrastructure

Wave B does not introduce:
- External metrics infrastructure (Prometheus, Grafana, StatsD).
- Auto-recovery mechanisms beyond the existing per-family supervisor restart.
- Dead-letter queues or overflow persistence.
- Message replay or backfill infrastructure.

These remain deferred until Wave B is complete and a dedicated infrastructure stage is planned.

---

## Operational Constraints (Derived from Wave A)

### C-5: Schema Coherence Is Blocking

Any detected mismatch between DDL, writer mapper, and reader adapter is a blocking defect. The family cannot proceed past the schema step until all three artifacts are column-aligned and this alignment is tested.

### C-6: Optionality Invariant Preserved

All ten analytical runtime optionality rules (R-01 through R-10) remain in force. Specifically:
- No operational service gains a dependency on ClickHouse.
- Operational smoke tests must pass without ClickHouse running.
- Writer uses independent consumer names with `writer-` prefix.
- No event path blocks on ClickHouse availability.

Any family expansion that would violate an optionality rule is rejected.

### C-7: No Horizontal Redesign

Wave B is vertical expansion (add families one at a time). It is NOT:
- A redesign of the inserter buffering strategy.
- A refactor of the reader adapter interface.
- A rearchitecture of the health endpoint model.
- A change to the supervisor lifecycle.

If a family reveals a need for horizontal changes, that need is documented as a debt and deferred to a future stage. The family ships within the current architecture or it does not ship.

### C-8: No Cross-Family Features

Wave B does not introduce:
- Queries that join data across multiple families.
- Composite endpoints that aggregate results from multiple tables.
- Shared materialized views.
- Cross-family correlation queries.

Each family is independent. Cross-family features are a future concern.

### C-9: Additive Only

All changes during Wave B are additive:
- New migration files (never modify existing migrations).
- New mapper functions (never modify existing mappers).
- New pipeline entries (never modify existing entries).
- New reader methods (never modify existing reader methods).
- New routes (never modify existing route registrations).
- New smoke test sections (never modify existing sections).

If a bug is found in an existing family during Wave B work, it is fixed in a separate commit with its own test, not bundled with the new family.

---

## Non-Goals

The following are explicitly NOT goals of Wave B:

### NG-1: Framework or Generic Expansion Engine

Wave B does not build a generic "family registration framework" or plugin system. The pattern is a documented convention enforced by code review, not a runtime abstraction. Adding a family means writing concrete code that follows the template, not configuring a generic engine.

### NG-2: Performance Optimization

Wave B does not optimize query performance, batch sizes, or buffer strategies. The existing defaults (batch_size=1000, flush_interval=5s, max_pending=10000) apply to all families. Per-family tuning is a future concern.

### NG-3: Custom Retention Per Family

All families use the same 90-day TTL. Differentiated retention policies are out of scope.

### NG-4: Schema Evolution Tooling

Wave B adds new tables. It does not build tooling for altering existing table schemas, migrating data between schema versions, or managing backwards-compatible schema changes.

### NG-5: Multi-Instance or Sharding

Wave B targets a single ClickHouse instance. Sharding, replication, or multi-instance deployment are out of scope.

### NG-6: Real-Time Streaming Queries

All analytical endpoints are request-response HTTP queries against stored data. WebSocket streams, SSE, or push-based analytical notifications are out of scope.

### NG-7: User-Facing Documentation

Wave B produces internal architecture documentation and runbooks. End-user API documentation, OpenAPI specs, or client SDK guides are out of scope.

### NG-8: Backfill or Historical Import

Wave B writes events as they arrive. Historical backfill of events that occurred before a family's pipeline was activated is out of scope.

---

## Expansion Validity Test

Use this decision tree to determine whether a proposed change belongs in a Wave B iteration:

```
Is it adding exactly one new family?
├── No  → REJECT (violates C-1)
└── Yes
    ├── Does it modify existing families? → REJECT (violates C-9)
    ├── Does it add cross-family features? → REJECT (violates C-8)
    ├── Does it require horizontal refactoring? → REJECT (violates C-7)
    ├── Does it introduce external infrastructure? → REJECT (violates C-4)
    ├── Does it satisfy the full checklist? → ACCEPT
    └── Does it NOT satisfy the full checklist? → REJECT (violates C-2)
```

## Distinguishing Valid Expansion from Premature Expansion

| Signal | Valid Expansion | Premature Expansion |
|---|---|---|
| Scope | One family, all nine artifacts | Multiple families, or partial artifacts |
| Dependencies | NATS subject exists, event structure stable | Event structure still changing, subject not yet published |
| Motivation | Analytical query need for existing events | "While we're here, let's also add..." |
| Architecture | Fits within current inserter/reader/handler model | Requires new adapter patterns or infrastructure |
| Tests | Full coverage before merge | "Tests will follow in next PR" |
| Smoke | Extended and passing | "Will update smoke later" |

If any column falls in the "Premature" row, the expansion is premature and must be deferred.

---

## Debt Capture During Wave B

When a family iteration reveals a need that falls outside Wave B scope:

1. Document the need in the iteration's gate review.
2. Classify: architectural debt, operational debt, or tooling debt.
3. Assess urgency: blocking (stops future families) vs. non-blocking (can wait).
4. Add to the cumulative Wave B debt register (maintained in the final Wave B readiness review).

Debts do not block the current iteration unless they violate a constraint (C-1 through C-9).
