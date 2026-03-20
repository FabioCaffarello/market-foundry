# Next Wave Recommendations After Current Capability Consolidation

> Strategic direction after S137–S141, grounded in what consolidation proved and what remains open.

---

## 1. Context

The consolidation wave (S137–S141) closed the gap between "the system works" and "the system is defined, observable, and governable." The post-consolidation readiness review confirmed:

- Baseline is canonical and testable (30 criteria, 5 tiers)
- Operations are observable (4 endpoints, phase classification, scripted diagnostics)
- Recovery semantics are explicit (survival/loss matrix, bounded shutdown)
- Ergonomics and governance are formalized (shared lib, config reference, entry principles)
- ClickHouse preparation is documented but entirely unimplemented

The question is: **what should the next wave focus on?**

---

## 2. Options Evaluated

### Option A: ClickHouse/Migrations Preparation Wave
**Scope:** Build `cmd/migrate`, design core schemas, implement writer service, extend query surface.
**Pros:**
- Addresses the single largest architectural constraint (in-memory state, no historical queries)
- Unblocks TC-02 (more timeframes) via cold-start bootstrap
- Entry principles and catalog conventions already defined — clear guardrails exist
- ClickHouse container already in docker-compose

**Cons:**
- Significant implementation effort (new binary, new service, new query paths)
- Risk of over-engineering if persistence needs are still speculative
- No external consumer exists for historical queries yet

**Verdict: RECOMMENDED as primary wave — but phased, not monolithic.**

### Option B: New Capability Addition (e.g., MACD signal, alert system)
**Scope:** Add a new family to the pipeline to exercise extensibility.
**Pros:**
- Tests the "add a family" ergonomic path documented in S141
- Delivers visible new capability quickly
- Low architectural risk (follows established pattern)

**Cons:**
- Does not address any open debt
- Does not advance the strategic goal (ClickHouse readiness)
- The pipeline extensibility was already proven with RSI, EMA crossover, and the full causal chain

**Verdict: NOT RECOMMENDED as a primary wave. Suitable as a validation exercise within another wave.**

### Option C: TC-02 (More Timeframes)
**Scope:** Expand beyond 4 timeframes (e.g., add 4h, daily).
**Pros:**
- Tests scaling on the temporal axis
- Exercises config-driven expansion further

**Cons:**
- Hard-gated by OD-01 (state persistence) — RSI warm-up at 14400s would be days
- Per-TF idle detection (OD-06) becomes a real problem
- Diminishing architectural return — config-driven scaling already proven at 4 TFs

**Verdict: NOT RECOMMENDED. Blocked by OD-01 until state persistence exists.**

### Option D: Targeted Hardening (automated validation, query observability, gateway trackers)
**Scope:** Address OD-03 through OD-05 from the open debts list.
**Pros:**
- Improves operational quality
- Lower risk than new infrastructure

**Cons:**
- Does not advance strategic goals
- Benefits are marginal at current scale (single developer, 2 symbols)
- Gateway tracker gap is cosmetic until gateway has independent health concerns

**Verdict: NOT RECOMMENDED as a standalone wave. Individual items can be addressed opportunistically.**

---

## 3. Recommendation

### Primary: Phased ClickHouse/Migrations Preparation

Execute the ClickHouse preparation as a **phased wave**, not a single large stage. Each phase delivers a working increment:

**Phase 1 — Migration Infrastructure (1 stage)**
- Build `cmd/migrate` with apply, track, drift-detect, dry-run
- Create `deploy/migrations/` with `_migrations` metadata table
- Deliverable: a working migration tool that can be used independently

**Phase 2 — Core Schema Design (1 stage)**
- Design DDL for the 6 core event tables
- Apply via `cmd/migrate`
- Validate schema matches NATS event structure
- Deliverable: ClickHouse tables exist and can receive data

**Phase 3 — Writer Service (1 stage)**
- Implement ClickHouse writer as NATS consumer
- Wire into docker-compose as optional service
- Validate events flow without affecting pipeline
- Deliverable: events are persisted in ClickHouse alongside NATS

**Phase 4 — Query Surface Extension (1 stage)**
- Add historical query endpoints to gateway
- Implement ClickHouse read path
- Deliverable: gateway can serve both "latest" (KV) and "historical" (ClickHouse) queries

**Phase 5 (conditional) — Cold-Start Bootstrap**
- Only if Phase 3 proves stable
- Bootstrap RSI accumulators from ClickHouse on cold start
- Deliverable: reduced warm-up time for long timeframes

### Why Phased

- Each phase has its own deliverable and can be validated independently
- If priorities change after Phase 1, the migration tool is still useful
- The writer can be built and tested before the query surface is designed
- Phase 5 is conditional — it may not be needed if warm-up time is acceptable

---

## 4. What Should NOT Be in the Next Wave

1. **Production deployment infrastructure** — The system runs locally. Kubernetes, CI/CD, or cloud deployment are out of scope.
2. **New pipeline families** — The pipeline is extensible and proven. Adding families is not a strategic priority.
3. **Multi-venue execution** — Venue abstraction exists (paper_simulator, binance_futures_testnet). Real venue integration is a separate concern.
4. **External API consumers** — No external consumer exists. Building for hypothetical consumers is premature.
5. **Monitoring/alerting infrastructure** — Prometheus, Grafana, or similar are out of scope until operational scale justifies them.

---

## 5. Success Criteria for the Next Wave

The next wave succeeds if:

1. `cmd/migrate` exists and can apply, track, and verify migrations
2. Core ClickHouse tables exist with schemas that mirror NATS event structure
3. A writer service consumes from NATS and persists to ClickHouse without affecting the pipeline
4. The existing pipeline continues to function identically with ClickHouse stopped
5. The preparation gate (PC-01 through PC-04) is fully passed

The wave fails if:
- ClickHouse becomes a startup dependency for any existing service
- The writer introduces latency or failure modes into the pipeline
- Migrations are applied ad-hoc instead of through `cmd/migrate`
- Schema diverges from the existing event structure

---

## 6. Decision Framework

Before starting the next wave, confirm:

| Question | Required Answer |
|---|---|
| Is the consolidation wave truly complete? | Yes — S142 confirms this |
| Are the ClickHouse entry principles understood and accepted? | Yes — 7 principles in `future-clickhouse-and-migrations-entry-principles.md` |
| Is the migration catalog convention clear? | Yes — `future-migration-catalog-organization-guidelines.md` |
| Is the phased approach agreed? | Confirm with stakeholder |
| Is the current baseline stable enough to build on? | Yes — 30 criteria, 5 tiers validated |
| Are there blocking debts that must be addressed first? | No — OD-01 (state persistence) is addressed by the wave itself |
