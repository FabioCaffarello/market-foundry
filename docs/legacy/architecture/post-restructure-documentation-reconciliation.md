# Post-Restructure Documentation Reconciliation

**Stage:** S221
**Date:** 2026-03-20
**Scope:** Documentation reconciliation after S218–S220 structural refactoring tranche

---

## 1. Purpose

This document records the documentation reconciliation performed after the S218–S220 structural refactoring tranche (H-01, H-04, H-06). The tranche introduced significant architectural changes that created drift between code and documentation. This reconciliation aligns the documentation with the new code reality.

---

## 2. Architectural Changes Requiring Reconciliation

### 2.1 H-01: NATS Adapter Sub-Packaging (S218)

**Before:** 73 files in a flat `internal/adapters/nats/` directory. All domain adapters (decision, evidence, execution, risk, signal, strategy, configctl, observation) shared a single package namespace.

**After:** 8 domain-specific sub-packages + 1 shared utilities package:
- `natskit/` — shared codec, connection, consumer spec factory, request/reply, content types
- `natsconfigctl/` — binding consumer, event publisher, gateway, registry
- `natsdecision/` — consumer, gateway, kv store, publisher, registry
- `natsevidence/` — candle consumer, candle kv store, trade burst consumer/kv, volume consumer/kv, gateway, publisher, registry
- `natsexecution/` — consumer, fill consumer, control gateway/kv, gateway, kv store, publisher, registry
- `natsobservation/` — consumer, publisher, registry
- `natsrisk/` — consumer, gateway, kv store, publisher, registry
- `natssignal/` — consumer, gateway, kv store, publisher, registry
- `natsstrategy/` — consumer, gateway, kv store, publisher, registry

**Documentation impact:** References to "73 files flat" or "flat NATS adapter" are now stale. The adapter is domain-organized.

### 2.2 H-04: Actor Migration Completion (S219)

**Before:** 9 structurally identical store consumer actor implementations (`decision_consumer_actor.go`, `evidence_consumer_actor.go`, etc.), each ~60 lines with only constructor, event type, and message type varying.

**After:** 1 `GenericConsumerActor` with closure-captured variance via `ConsumerStartFn`. 8 consumer actor files deleted. ~510 lines of duplication eliminated. Adding a new pipeline reduced from 4 steps to 1 step (single closure ~8 lines).

**What was NOT migrated (intentionally):** Projection actors remain domain-specific because their logic diverges. Derive scope publisher actors are candidates for future consolidation.

**Documentation impact:** References to "per-family consumer actors" or "1,800 lines projected savings" need updating — the migration is complete and actual savings were ~510 lines.

### 2.3 H-06: Module Graph Simplification (S220)

**Before:** 19 Go modules in `go.work`.

**After:** 17 Go modules. Two absorptions:
1. `internal/migrate` (478 LOC) → `cmd/migrate/migrate` (1:1 consumer relationship)
2. `internal/adapters/repositories` (1,434 LOC) → `internal/application/configctl/memoryrepo` (application already defines ports)

**Modules NOT merged (justified):** `internal/interfaces/http` (size), `clickhouse`/`exchanges`/`nats` (external dependency isolation).

**Documentation impact:** All references to "19 modules" are now stale. The workspace has 17 modules. Import paths changed for 3 consumers.

---

## 3. Documents Reconciled

| Document | Changes Applied |
|----------|----------------|
| `refactor-wave-gains-tradeoffs-and-open-debts.md` | H-01/H-04/H-06 status → DONE; depth-vs-breadth trade-off updated; debt disposition 10→7 deferred; net assessment updated |
| `pre-refactor-technical-debt-registry-and-cleanup-plan.md` | AD-01 status updated (19→17 modules, H-06 done) |
| `next-wave-recommendations-after-post-refactor-and-documentation-gate.md` | Path B marked COMPLETED with deliverables; recommendation updated to post-Path-B state |
| `post-refactor-and-documentation-exit-gate.md` | XC-4 module count corrected; S213 grade upgraded; S221 reconciliation header added |
| `documentation-canonical-map-after-consolidation.md` | Doc counts updated (249 arch docs, 219 stage reports); S221 count note added |
| `docs/stages/INDEX.md` | Phase 18 expanded: S218, S220, S221 entries added |

---

## 4. Structural Debt Status After Reconciliation

| ID | Item | Priority | Status |
|----|------|----------|--------|
| H-01 | NATS adapter sub-packaging | HIGH | **DONE** (S218) |
| H-02 | Consumer spec factory | HIGH | **DONE** (S213) |
| H-03 | ClickHouse query builder | HIGH | **DONE** (S213) |
| H-04 | Per-family actor migration | HIGH | **DONE** (S219) |
| H-05 | Handler extraction | HIGH | **DONE** (S217 verified) |
| H-06 | Module graph simplification | HIGH | **DONE** (S220) |
| M-01–M-07 | Medium-priority items | MEDIUM | NOT STARTED |

**All 6 HIGH-priority structural items from the S212 census are now DONE.**

---

## 5. Module Graph — Current Canonical State

17 modules in `go.work`:

```
cmd/configctl    cmd/derive    cmd/execute    cmd/gateway
cmd/ingest       cmd/migrate   cmd/store      cmd/writer
codegen
internal/actors
internal/adapters/clickhouse
internal/adapters/exchanges
internal/adapters/nats
internal/application
internal/domain
internal/interfaces/http
internal/shared
```

---

## 6. NATS Adapter — Current Canonical Structure

```
internal/adapters/nats/
├── natskit/           (shared: codec, connection, consumer spec factory, types)
├── natsconfigctl/     (config control domain)
├── natsdecision/      (decision domain)
├── natsevidence/      (evidence domain: candle, trade burst, volume)
├── natsexecution/     (execution domain: orders, fills, control)
├── natsobservation/   (observation domain)
├── natsrisk/          (risk domain)
├── natssignal/        (signal domain)
└── natsstrategy/      (strategy domain)
```

---

## 7. Store Actor Layer — Current Canonical State

```
internal/actors/scopes/store/
├── generic_consumer_actor.go      (single generic consumer)
├── store_supervisor.go            (supervisor with ConsumerStartFn closures)
├── projection_store.go            (shared projection store)
├── query_responder_actor.go       (query responder)
├── *_projection_actor.go          (9 domain-specific projection actors)
└── *_projection_actor_test.go     (9 corresponding test files)
```

No per-domain consumer actor files remain. All consumer variance is captured via closures in the supervisor.
