# ClickHouse and Migrations Preparation Gate

> Formal gate defining what must be true before the Foundry begins implementing ClickHouse integration and migration tooling.

---

## 1. Purpose

This document defines the **preparation gate** — the set of pre-conditions that must be satisfied before any ClickHouse implementation work begins. Passing this gate means the Foundry is ready to build migration tooling and ClickHouse integration. Failing any critical pre-condition means the system is not ready, regardless of desire or schedule pressure.

This gate separates **planning** (which can happen now) from **implementation** (which requires these conditions).

---

## 2. Gate Status Summary

| Category | Status | Notes |
|---|---|---|
| Baseline consolidation | **PASSED** | S137–S141 complete, baseline canonical |
| Entry principles | **PASSED** | 7 principles defined and documented |
| Migration catalog conventions | **PASSED** | Naming, numbering, idempotency rules defined |
| Signal candidates | **PASSED** | Analytics signals catalogued with priorities |
| Persistence triggers | **PASSED** | Decision matrix with thresholds defined |
| Migration tooling | **NOT STARTED** | `cmd/migrate` does not exist |
| ClickHouse schema | **NOT STARTED** | No DDL written |
| Writer service | **NOT STARTED** | No code exists |
| Event schema stability proof | **PARTIAL** | Schemas stable in practice; no formal versioning contract |

---

## 3. Pre-Conditions for Implementation

### 3.1 Critical Pre-Conditions (all must pass)

#### PC-01: Migration Tool Exists
**Requirement:** A `cmd/migrate` binary that can:
- Discover migration files from `deploy/migrations/`
- Apply migrations in order (by numeric prefix)
- Track applied migrations in a `_migrations` metadata table
- Detect schema drift (checksum comparison)
- Support dry-run mode

**Rationale:** Without tooling, schemas will be applied ad-hoc and drift will be undetectable. Principle P-02 (migrations are versioned code) and P-03 (one migration tool) require this.

**Current status:** Not started.

#### PC-02: Core Tables Schema Designed
**Requirement:** DDL files for core event tables:
- `evidence` (candle events)
- `signals` (RSI, EMA crossover events)
- `decisions` (RSI oversold events)
- `strategies` (mean reversion entry events)
- `risk_assessments` (position exposure events)
- `executions` (paper order, venue order events)

Each table must follow the event schema from NATS subjects, with appropriate ClickHouse engines (MergeTree family), partition keys (by date), and order keys (by symbol + timeframe + timestamp).

**Rationale:** Principle P-04 (schema follows events) requires schemas to mirror the existing event structure, not invent new ones.

**Current status:** Not started. Event schemas in Go code are stable and well-defined but no ClickHouse DDL exists.

#### PC-03: Writer Architecture Decided
**Requirement:** A clear design for the ClickHouse writer service that:
- Consumes from existing NATS subjects (not new ones)
- Writes asynchronously (not on the hot path)
- Is a consumer, not a producer (Principle P-05)
- Can be stopped without affecting the pipeline
- Handles backpressure and retry

**Rationale:** Principle P-07 (no dual-write complexity) and P-05 (writer is consumer) require this architecture to be settled before implementation begins.

**Current status:** Architecture is conceptually clear from the entry principles. No design document exists.

#### PC-04: ClickHouse Remains Optional at Runtime
**Requirement:** The existing pipeline (ingest → derive → store → execute) must continue to function identically with ClickHouse stopped or absent. No service startup depends on ClickHouse availability.

**Rationale:** Principle P-01 (ClickHouse remains optional). The current docker-compose already satisfies this — ClickHouse is defined but not in any dependency chain. This pre-condition ensures that implementation does not introduce coupling.

**Current status:** **PASSED** — ClickHouse container is isolated in docker-compose.

### 3.2 Important Pre-Conditions (strongly recommended)

#### PC-05: Event Schema Versioning Convention
**Requirement:** A lightweight convention for versioning event schemas (e.g., a version field in the NATS envelope or a schema registry document) so that the ClickHouse writer knows which schema version it is consuming.

**Rationale:** Without schema versioning, the writer cannot detect or handle schema evolution. This is manageable at current scale (single developer, stable schemas) but becomes a correctness risk as schemas evolve.

**Current status:** Not started. Event schemas are implicitly versioned by Go struct definitions.

#### PC-06: Retention Policy Defined
**Requirement:** Explicit retention policy for ClickHouse data:
- How long to retain raw events (days, months, indefinite)
- Whether materialized views or aggregations should replace raw data after a retention window
- Disk space budget for the development environment

**Rationale:** Without a retention policy, disk usage is unbounded. ClickHouse TTL and partitioning strategies depend on retention requirements.

**Current status:** Not defined. NATS retention is 72 hours; ClickHouse retention is undecided.

#### PC-07: Query Surface Extension Design
**Requirement:** A design for how ClickHouse data will be exposed through the gateway:
- New endpoints or extended existing endpoints?
- How to distinguish "latest from KV" vs. "historical from ClickHouse"?
- Query parameters for time range, aggregation, filtering

**Rationale:** ClickHouse without a query surface is storage without value. The design should be settled before implementation to avoid building a writer with no consumer.

**Current status:** Not started. Current query surface is NATS KV only (latest values).

---

## 4. Implementation Sequence (Once Gate Passes)

The following sequence respects the entry principles and dependency order:

```
Phase 1: Migration Infrastructure
├── Build cmd/migrate tool
├── Create deploy/migrations/ directory structure
├── Create _migrations metadata table DDL
└── Validate: migrate tool can apply and track migrations

Phase 2: Core Schema
├── Design and write core table DDL (evidence, signals, decisions, strategies, risk, executions)
├── Apply via cmd/migrate
└── Validate: tables exist, schema matches event structure

Phase 3: Writer Service
├── Implement ClickHouse writer as NATS consumer
├── Wire into docker-compose (optional dependency)
├── Validate: events flow from NATS to ClickHouse without affecting pipeline

Phase 4: Query Surface Extension
├── Add historical query endpoints to gateway
├── Implement ClickHouse read path
└── Validate: historical queries return data, KV queries unaffected

Phase 5 (optional): Cold-Start Bootstrap
├── Implement ClickHouse → in-memory bootstrap for RSI warm-up
└── Validate: cold-start time reduced for long timeframes
```

---

## 5. Anti-Patterns to Prevent

These are explicitly prohibited based on the entry principles:

| Anti-Pattern | Principle Violated | Prevention |
|---|---|---|
| Ad-hoc DDL execution (manual `CREATE TABLE`) | P-02 (migrations are versioned code) | All schema changes go through `cmd/migrate` |
| Multiple migration tools or scripts | P-03 (one migration tool) | Single `cmd/migrate` binary, single `deploy/migrations/` directory |
| Inventing new event schemas for ClickHouse | P-04 (schema follows events) | ClickHouse tables mirror NATS event structure |
| Writer publishing back to NATS | P-05 (writer is consumer) | Writer reads from NATS, writes to ClickHouse, publishes nothing |
| Synchronous writes on the hot path | P-07 (no dual-write) | Writer is asynchronous, pipeline-independent |
| Making ClickHouse a startup dependency | P-01 (ClickHouse optional) | No service health check depends on ClickHouse |

---

## 6. Gate Decision

**Current gate status: NOT PASSED for implementation.**

The Foundry has passed the planning pre-conditions (baseline consolidated, principles defined, conventions documented). It has **not** passed the implementation pre-conditions (no migration tool, no schema, no writer design).

**Recommended next step:** A dedicated preparation wave that addresses PC-01 through PC-03 as its primary deliverables, with PC-05 through PC-07 as secondary goals. This wave should be scoped as infrastructure preparation, not feature delivery.
