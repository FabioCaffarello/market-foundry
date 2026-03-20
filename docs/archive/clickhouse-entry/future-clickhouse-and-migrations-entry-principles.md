# Future ClickHouse and Migrations Entry Principles

> **Stage:** S141 — Ergonomics and Governance Consolidation
> **Scope:** Principles only. No implementation.

---

## 1. Purpose

This document defines the principles and constraints for how ClickHouse integration and database migrations should enter the Market Foundry monorepo. It is a governance document — it authorizes nothing but sets the rules for when authorization is eventually granted.

---

## 2. Core Principles

### P-01: ClickHouse Remains Optional

The pipeline must function without ClickHouse. No service may add ClickHouse to its readiness checks. No event path may block on ClickHouse availability. ClickHouse is an analytical augmentation, not an operational dependency.

**Test:** Remove the ClickHouse container from `docker-compose.yaml`. All smoke tests must still pass.

### P-02: Migrations Are Code, Not Ad-Hoc DDL

Every ClickHouse schema change must be expressed as a versioned migration file. No manual `CREATE TABLE` or `ALTER TABLE` in production-like environments. Migrations are reviewed, versioned, and applied through a controlled tool.

### P-03: One Migration Tool, One Catalog

The monorepo uses a single migration runner (`cmd/migrate` when introduced) and a single catalog directory (`deploy/migrations/`). No per-service migration directories. No competing tools.

### P-04: Schema Follows Events

ClickHouse table schemas are derived from the existing NATS event structures. The event schema is the source of truth; the ClickHouse schema is a projection. If an event schema changes, the migration must follow.

### P-05: Writer Is a Consumer, Not a Producer

The ClickHouse writer follows the same consumer pattern as the `store` service: subscribe to NATS event stream → INSERT into ClickHouse. It does not produce events, modify state, or participate in the pipeline's hot path.

### P-06: Migrations Are Idempotent

Every migration must be safe to run multiple times. Use `CREATE TABLE IF NOT EXISTS`, `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, etc. A failed migration that is re-run must not corrupt data or leave partial state.

### P-07: No Dual-Write Complexity

Events flow independently to NATS KV (via store) and ClickHouse (via writer). These are parallel consumers, not a coordinated dual-write. Store does not know about ClickHouse. ClickHouse does not know about store.

---

## 3. Entry Sequence

When the decision to implement ClickHouse is made, these steps must be followed in order:

### Step 1: Migration Infrastructure (`cmd/migrate`)

- Create `cmd/migrate/` with a minimal CLI that:
  - Connects to ClickHouse
  - Reads migration files from `deploy/migrations/`
  - Applies pending migrations in version order
  - Records applied migrations in a `_migrations` metadata table
- No domain logic, no event consumption, no NATS dependency
- The migration tool must work standalone (independent of the rest of the monorepo)

### Step 2: Schema Design (migration files)

- Create `deploy/migrations/` with numbered SQL files
- Start with P1 tables: `evidence_candles`, `evidence_tradebursts`, `evidence_volumes`
- Each file is a self-contained, idempotent SQL migration
- Review schema against event structures in `internal/domain/`

### Step 3: Writer Service or Module

- Decide: new `cmd/writer/` service or optional module in `store`
- Implement NATS consumer → ClickHouse INSERT with batch buffering
- Writer must tolerate ClickHouse downtime (buffer or drop, never block)
- Writer gets its own NATS durable consumer name (never shares with store)

### Step 4: Query Surface Extension

- Add gateway routes for ClickHouse-backed historical queries
- Clear boundary: NATS KV = latest, ClickHouse = history/analytics
- New endpoints, not modifications to existing ones

### Step 5: Cold-Start Bootstrap (optional)

- Derive reads historical candles from ClickHouse on startup
- Pre-seeds RSI and other stateful indicators
- Only if operational requirements demand faster recovery

---

## 4. Pre-Conditions Checklist

Before starting Step 1, these conditions must be met:

| ID | Condition | How to verify |
|----|-----------|---------------|
| CH-01 | Baseline is canonical and stable | S137–S140 completed |
| CH-02 | Ergonomics and governance consolidated | S141 completed (this document exists) |
| CH-03 | Event schemas are stable | No pending event structure changes |
| CH-04 | Retention policy defined | Decision on TTL per table type |
| CH-05 | Query use cases documented | Who queries what, how often |
| CH-06 | Migration tool selected or designed | `cmd/migrate` spec reviewed |

---

## 5. Anti-Patterns

| Anti-Pattern | Why It's Dangerous |
|-------------|-------------------|
| Making ClickHouse a startup dependency | Breaks P-01; pipeline stops working if CH is down |
| Per-service migration directories | Violates P-03; fragmented schema management |
| Manual DDL in production | Violates P-02; untracked schema state |
| Sharing NATS consumer with store | Store's KV projection disrupted by writer failures |
| Synchronous writes from derive | Blocks hot path; violates P-05 |
| Designing for massive scale before proving value | Current cardinality is tiny; premature optimization |
| Adding CH without migration infrastructure | Ad-hoc tables with no version control |

---

## 6. Relationship to Existing Documents

This document builds on and supersedes the strategic direction in:

- `current-baseline-and-future-clickhouse-preparation-notes.md` — S137 awareness
- `future-analytics-signals-candidates-for-clickhouse.md` — signal catalog
- `future-state-persistence-and-clickhouse-trigger-notes.md` — trigger conditions

Those documents provide context and rationale. This document provides governance rules.

---

## 7. Decision Authority

The decision to proceed from principles to implementation requires:

1. At least one trigger condition from the trigger matrix (S140 document) is met
2. All pre-conditions in Section 4 are satisfied
3. The implementation follows the entry sequence in Section 3
4. The stage report documents which trigger and pre-conditions were evaluated
