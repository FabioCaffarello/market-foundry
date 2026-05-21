# Stage S145 — Writer Service Architecture Decision Report

> **Status:** Complete
> **Date:** 2026-03-19
> **Scope:** Formal architectural decision for the analytical writer service.
> **Prerequisite stages:** S142 (preparation gate), S143 (entry architecture), S144 (schema design).

---

## 1. Executive Summary

This stage formalizes the architectural decision for the writer service — the component that bridges the operational NATS pipeline and the analytical ClickHouse layer. The decision is: **dedicated standalone runtime** (`cmd/writer/`), following the canonical 6-phase composition root, consuming events via independent NATS durable consumers, and appending them to ClickHouse in batches with a buffer-and-drop failure policy.

The writer is structurally optional. The operational baseline functions identically with or without it. No operational service is modified, extended, or made aware of the writer's existence.

All architectural decisions, failure semantics, delivery guarantees, and boundary constraints are now formally documented and ready for disciplined implementation.

## 2. Architectural Decision

### 2.1 Core Decision

| Question | Decision |
|----------|----------|
| Is the writer a dedicated runtime? | **Yes.** Standalone binary at `cmd/writer/`. |
| How does it consume events? | Via NATS JetStream durable consumers with `writer-*` prefix. Independent cursors from store. |
| What is the write policy? | Batch INSERT: 1000 events or 5 seconds, whichever comes first. |
| How does it handle ClickHouse failure? | Buffer up to 10K events, then drop oldest. Never blocks NATS consumer. |
| What delivery guarantee? | At-least-once from NATS to ClickHouse. Duplicates tolerated; query-time dedup if needed. |
| Is it optional relative to baseline? | **Yes.** Structurally optional. Removing the container is the complete removal path. |

### 2.2 Why This Decision Is Correct Now

1. **Failure isolation is non-negotiable.** The operational pipeline (ingest → derive → store → execute) is the proven, stable foundation. Analytical writes must not introduce any failure mode that can propagate to operational services. A dedicated process boundary is the only mechanism that provides this guarantee without relying on developer discipline.

2. **The pattern already exists.** The store service establishes the consumer–projection dual-actor pattern, the declarative pipeline catalog, the health tracker model, and the shutdown sequence. The writer replicates this pattern with ClickHouse as the destination instead of NATS KV. This is extension, not invention.

3. **Optionality must be structural, not behavioral.** A configuration flag (`clickhouse.enabled: true`) inside an existing service creates conditional branches that erode over time. A separate binary that is present or absent is a binary state with no ambiguity.

4. **Scale is known and small.** 2 symbols × 4 timeframes × 6 families × ~3,700 events/day total. Batch insertion at 1000 events/5s is dramatically over-provisioned. This means the writer can be simple — no parallelism, no sharding, no complex buffering needed.

## 3. Consumption, Write, and Failure Semantics

### 3.1 Consumption

- 6 independent NATS durable consumers, one per event family.
- Consumer names: `writer-evidence-candle-consumer`, `writer-signal-consumer`, etc.
- Same NATS subjects as store. No new streams or subjects created.
- Consumer actor deserializes JSON and forwards typed messages to inserter actor.

### 3.2 Write Policy

- Inserter actor accumulates events in memory buffer.
- Flushes on `batch_size` (1000) or `flush_interval` (5s), whichever comes first.
- Single batch INSERT per table per flush.
- Minimal mechanical transformation: metadata extraction, decimal parsing, JSON serialization of nested structs.
- No filtering, aggregation, deduplication, or enrichment.

### 3.3 Failure Modes

| Failure | Response | Recovery |
|---------|----------|----------|
| ClickHouse down | Buffer → drop oldest beyond 10K | Auto-resume on reconnect; replay from NATS for gaps |
| INSERT rejected | Exponential backoff × 5 attempts → drop batch | Fix schema; replay if needed |
| NATS down | Standard NATS reconnection | Resume from last-acked position |
| Deserialization error | Term message; log WARN | Fix producer-side bug |
| Writer crash | Docker restarts; NATS re-delivers unacked | At most one batch of duplicates |

### 3.4 Idempotency

Not enforced at write time. Duplicates are rare (crash boundaries only) and tolerable at current scale. Query-time dedup (`GROUP BY event_id`) is the documented mitigation. ReplacingMergeTree is the documented upgrade path if needed.

## 4. Risks and Anti-Patterns

### 4.1 Risks

| Risk | Severity | Likelihood | Mitigation |
|------|----------|-----------|------------|
| ClickHouse import leaks into operational service | High | Medium | Import-level enforcement; code review |
| Writer failure misattributed to pipeline issue | Medium | Medium | Clear health endpoints; separate `/statusz` |
| Buffer overflow during extended ClickHouse outage | Medium | Low | 10K buffer covers hours at current scale; NATS replay for gaps |
| Schema drift between Go structs and ClickHouse DDL | Medium | Medium | Schema-follows-events rule; manual review discipline |
| Scope creep during implementation | High | High | First-version limits are explicit guard rails |

### 4.2 Anti-Patterns to Avoid

| Anti-Pattern | Correct Approach |
|-------------|------------------|
| Writer inside store (optional module) | Dedicated `cmd/writer/` binary |
| Per-event INSERT to ClickHouse | Batch INSERT only |
| Blocking NATS consumer on ClickHouse failure | Buffer + drop; never block |
| Acking before write confirmation | Ack only after successful INSERT |
| Writer publishing to NATS | Writer is read-only (INV-03) |
| Dedup infrastructure at current scale | Query-time dedup; upgrade path documented |
| Conditional branches in operational services | No `if writer/clickhouse` logic anywhere |
| Shared consumer names with store | `writer-*` prefix always |
| Complex transformation in inserter | Mechanical type mapping only |
| Backfill in first version | Forward-only from consumer cursor |

## 5. Boundaries Maintained

### 5.1 Baseline Decoupling

- No operational service imports ClickHouse driver.
- No operational readiness check references ClickHouse.
- No operational event handler blocks on ClickHouse.
- Smoke tests pass without ClickHouse and writer.
- configctl has no ClickHouse awareness.

### 5.2 First-Version Limits

| Limit | Description |
|-------|-------------|
| L-01 | 6 families only |
| L-02 | No deduplication at write time |
| L-03 | No transformation beyond mechanical type mapping |
| L-04 | No materialized views |
| L-05 | No backfill capability |
| L-06 | No dynamic family registration |
| L-07 | Single ClickHouse instance |

### 5.3 Optionality Compliance

All 10 analytical runtime optionality rules (R-01 through R-10) are satisfied. Compliance is documented per-rule in the optionality and runtime boundaries document.

## 6. Deliverables

| Document | Path | Content |
|----------|------|---------|
| Writer architecture | `docs/architecture/writer-service-architecture.md` | Runtime structure, NATS consumption, write policy, config, health model, shutdown, first-version limits |
| Failure and delivery semantics | `docs/architecture/writer-service-failure-and-delivery-semantics.md` | Failure taxonomy, retry policy, idempotency, replay, diagnostic visibility, anti-patterns |
| Optionality and runtime boundaries | `docs/architecture/writer-service-optionality-and-runtime-boundaries.md` | Structural/runtime/config boundaries, invariant compliance, package constraints, deployment topology |
| This report | `docs/stages/stage-s145-writer-service-architecture-decision-report.md` | Decision summary, rationale, risks, preparation for S146 |

## 7. Criteria de Aceite — Verificação

| Critério | Status |
|----------|--------|
| Arquitetura do writer formalmente decidida | **Satisfeito** — dedicated runtime, dual-actor, batch INSERT |
| Boundaries e semânticas claros | **Satisfeito** — 3 documentos com boundaries estruturais, runtime e config |
| Baseline atual permanece desacoplado | **Satisfeito** — 10 regras de opcionalidade verificadas |
| Riscos e anti-patterns explícitos | **Satisfeito** — 5 riscos + 10 anti-patterns documentados |
| Base pronta para implementação mínima disciplinada | **Satisfeito** — first-version limits explícitos, guard rails definidos |

## 8. Preparation for S146

S146 is the **writer service implementation** stage. With S145 complete, the following are now defined and ready:

### 8.1 Implementation Scope for S146

| Deliverable | Description |
|-------------|-------------|
| `cmd/writer/main.go` | Bootstrap entry point |
| `cmd/writer/run.go` | 6-phase composition root |
| `internal/actors/scopes/writer/` | WriterSupervisor, consumer actors, inserter actors |
| `internal/adapters/clickhouse/` | Connection pool, batch inserter |
| `deploy/configs/writer.jsonc` | Service configuration |
| Docker compose entry | `writer` service with ClickHouse dependency |
| Writer smoke test | Separate test validating writer pipeline |

### 8.2 Pre-Conditions for S146

| Pre-Condition | Status | Notes |
|---------------|--------|-------|
| Writer architecture decided (S145) | **Done** | This stage |
| Migration tool implemented | **Required** | `cmd/migrate` must exist before writer can write |
| Core tables created | **Required** | Migrations 001–006 must be applied |
| ClickHouse in docker-compose | **Required** | Container must be available for writer |

### 8.3 Recommended S146 Sequence

1. Add ClickHouse to docker-compose (service + healthcheck).
2. Implement `cmd/migrate` with `up`, `status`, `validate` commands.
3. Create migrations 000–006 (metadata + 6 core tables).
4. Validate: `make migrate-up` creates all tables on fresh ClickHouse.
5. Implement `internal/adapters/clickhouse/` (connection, batch inserter).
6. Implement writer actors (supervisor, consumers, inserters).
7. Implement `cmd/writer/` entry point.
8. Add writer to docker-compose.
9. Validate: writer processes events end-to-end with full pipeline running.
10. Add writer smoke test.

### 8.4 S146 Guard Rails

- Do not add ClickHouse imports to any operational service.
- Do not modify existing smoke tests to require ClickHouse.
- Do not add conditional branches to operational services.
- Do not implement materialized views, backfill, or historical endpoints.
- Do not add deduplication infrastructure.
- Keep the writer to 6 families only.
