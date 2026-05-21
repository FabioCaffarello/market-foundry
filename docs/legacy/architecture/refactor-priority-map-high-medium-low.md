# Refactor Priority Map — High / Medium / Low

> Stage: S212 — Repository Architecture Census and Refactor Map
> Date: 2026-03-20
> Status: Active
> Companion to: repository-boundaries-coupling-duplication-and-smells.md

---

## 1. Purpose

This document assigns a defensible priority to each identified refactoring item, based on three criteria:

1. **Evolution cost reduction** — how much does this refactoring lower the cost of adding the next family or feature?
2. **Blast radius** — how many files/packages does this affect?
3. **Risk** — how likely is this refactoring to introduce regressions?

Items are grouped into HIGH (do in S213), MEDIUM (do in S213 if time permits or defer to S214), and LOW (defer or skip).

---

## 2. Priority Scoring Matrix

| Factor | Weight | Score 3 (High) | Score 2 (Medium) | Score 1 (Low) |
|--------|--------|----------------|-------------------|----------------|
| Evolution cost | 40% | Blocks every new family | Blocks some expansions | Cosmetic or rare |
| Blast radius | 30% | >5 packages affected | 2–5 packages | 1 package |
| Risk | 30% (inverted) | Low regression risk | Moderate risk | High regression risk |

Priority = weighted score. HIGH ≥ 7, MEDIUM 5–6, LOW ≤ 4.

---

## 3. HIGH Priority — Execute in S213

### H-01: NATS Adapter Sub-Packaging (SD-01)

- **What:** Split `internal/adapters/nats/` (10,110 lines, 80+ files) into sub-packages: `registry/`, `consumer/`, `publisher/`, `gateway/`, `kvstore/`, `codec/`.
- **Why:** Largest single package in the codebase. Navigability is poor. Every family addition touches this package. Sub-packaging reduces cognitive load and enables targeted testing.
- **Evolution cost:** 3 — every family expansion adds 4+ files here
- **Blast radius:** 3 — affects store, derive, writer, gateway, execute
- **Risk:** 3 — mechanical move, no logic changes
- **Score: 9**
- **Estimated effort:** Medium (2–3 sessions)
- **Dependencies:** None

### H-02: Consumer Spec Factory (SD-02)

- **What:** Replace 25+ individual consumer spec functions with a factory: `NewConsumerSpec(service, layer, family) ConsumerSpec`. Keep codegen markers for the 4 integrated specs.
- **Why:** Each consumer spec is 11–12 lines of pure template. Factory reduces 1,008 lines to ~200 lines and eliminates copy-paste errors.
- **Evolution cost:** 3 — every family requires 2–4 new spec functions today
- **Blast radius:** 3 — registries, pipeline entries, store supervisor, codegen integration
- **Risk:** 2 — must preserve durable consumer names exactly (NATS offset migration risk)
- **Score: 8**
- **Estimated effort:** Medium (1–2 sessions)
- **Dependencies:** H-01 (easier after sub-packaging)

### H-03: ClickHouse Reader Consolidation (SD-03)

- **What:** Extract a generic query builder and row scanner. Each reader becomes: table name, column list, scan function. Introduce query parameter struct to replace 10+ positional args.
- **Why:** 884 lines → ~350 lines. Positional parameter signatures are the #1 reported pain point (TD-02).
- **Evolution cost:** 3 — every analytical family duplicates full reader
- **Blast radius:** 2 — clickhouse adapters + analytical use cases
- **Risk:** 2 — query correctness must be verified per family
- **Score: 7.4**
- **Estimated effort:** Medium (2 sessions)
- **Dependencies:** None

### H-04: Store Actor Generic Consumer/Projection (SD-04, SD-05)

- **What:** Create generic `FamilyConsumerActor[T]` and `FamilyProjectionActor[T]` with type-parametric implementations. Each family provides: consumer spec, decode function, KV store interface.
- **Why:** 2,400 lines of store actors → ~600 lines. Largest duplication cluster in actors.
- **Evolution cost:** 3 — every family requires 2 new actors today
- **Blast radius:** 2 — store scope only
- **Risk:** 2 — actor lifecycle must be preserved exactly
- **Score: 7.4**
- **Estimated effort:** Medium-Large (2–3 sessions)
- **Dependencies:** H-02 (consumer specs used by actors)

### H-05: Documentation Entropy Reduction (SS-06)

- **What:** Execute the 12-phase consolidation plan from S209 documentation-entropy-archive-delete-consolidate-map.md. Target: 449 → ≤150 active architecture docs.
- **Why:** Documentation entropy directly blocks onboarding, decision discovery, and architectural reasoning. Every conversation about "where is X documented" costs time.
- **Evolution cost:** 3 — misleading docs cause wrong decisions
- **Blast radius:** 3 — affects entire team
- **Risk:** 3 — docs only, no code regression risk
- **Score: 9**
- **Estimated effort:** Large (3–4 sessions)
- **Dependencies:** None

### H-06: Module Consolidation Evaluation (SS-01, AD-01)

- **What:** Evaluate and execute consolidation of 19 modules → ~8–10 modules. Primary candidates: merge 4 adapter modules into 1–2, merge cmd/ modules where feasible.
- **Why:** Module fragmentation increases go.sum maintenance, workspace sync time, and dependency graph complexity.
- **Evolution cost:** 2 — moderate friction on dependency management
- **Blast radius:** 3 — all modules, go.work, CI pipeline
- **Risk:** 2 — must preserve build isolation; adapter modules have different external deps
- **Score: 7**
- **Estimated effort:** Medium (2 sessions for evaluation + execution)
- **Dependencies:** None, but should be done early (affects all other refactoring)

---

## 4. MEDIUM Priority — Execute in S213 If Time Permits

### M-01: Analytical Use Case Consolidation (SD-06)

- **What:** Extract a generic `HistoryUseCase[Query, Reply]` with shared validation, nil-check, logging, and error handling. Each family provides only the reader interface and type-specific validation rules.
- **Why:** 584 lines → ~200 lines. Pattern is clear and mechanical.
- **Evolution cost:** 2 — affects analytical family addition
- **Blast radius:** 2 — analyticalclient package only
- **Risk:** 2 — validation rules vary slightly per family
- **Score: 6**
- **Estimated effort:** Small-Medium (1 session)
- **Dependencies:** H-03 (reader consolidation informs interface)

### M-02: HTTP Handler Abstraction (SD-07)

- **What:** Create a generic handler factory for the 4 simple handlers (signal, decision, strategy, risk). Analytical handler: extract template method for the 6 query methods.
- **Why:** ~740 lines → ~300 lines. High duplication percentage.
- **Evolution cost:** 2 — each new operational family needs a handler
- **Blast radius:** 2 — handlers + routes
- **Risk:** 2 — HTTP interface changes require careful testing
- **Score: 6**
- **Estimated effort:** Small-Medium (1 session)
- **Dependencies:** M-01 (use case interface informs handler shape)

### M-03: Writer Pipeline Configuration-Driven (SD-08, SD-09)

- **What:** Convert pipeline.go from hardcoded struct literals to a registry pattern driven by family specifications. Integrate remaining 5 families into codegen or declarative config.
- **Why:** 451 lines → ~150 lines. Currently only 2 of 7 entries are under codegen.
- **Evolution cost:** 2 — each family needs manual pipeline entry
- **Blast radius:** 1 — cmd/writer only
- **Risk:** 2 — pipeline startup sequence must be preserved
- **Score: 5.6**
- **Estimated effort:** Medium (1–2 sessions)
- **Dependencies:** H-02 (consumer spec factory)

### M-04: NATS KV Store Generic Implementation (SS-04)

- **What:** Create `GenericKVStore[T]` with encode/decode/put/get/list operations. Each family provides: bucket name, key function, codec.
- **Why:** ~1,400 lines → ~400 lines.
- **Evolution cost:** 2 — each family needs a KV store
- **Blast radius:** 2 — NATS adapters + store actors
- **Risk:** 2 — KV key generation must match existing data
- **Score: 6**
- **Estimated effort:** Medium (1–2 sessions)
- **Dependencies:** H-01 (NATS sub-packaging)

### M-05: NATS Consumer/Publisher Generic (SS-05)

- **What:** Create generic consumer and publisher implementations parameterized by event type and spec.
- **Why:** ~2,400 lines → ~800 lines.
- **Evolution cost:** 2 — each family needs consumer + publisher
- **Blast radius:** 2 — NATS adapters + all consuming services
- **Risk:** 2 — must preserve message compatibility
- **Score: 6**
- **Estimated effort:** Medium (2 sessions)
- **Dependencies:** H-01, H-02

### M-06: Gateway Compose Simplification (SD-10)

- **What:** Replace conditional wiring blocks with a declarative dependency registration pattern. Each family registers its connections and use cases; compose.go iterates the registry.
- **Why:** 246 lines → ~100 lines. Wiring is the main friction point for gateway changes.
- **Evolution cost:** 2 — each family adds wiring blocks
- **Blast radius:** 1 — cmd/gateway only
- **Risk:** 2 — connection lifecycle must be preserved
- **Score: 5.6**
- **Estimated effort:** Small (1 session)
- **Dependencies:** M-02 (handler abstraction informs wiring shape)

### M-07: Settings Schema Decomposition (CP-02, SS-02)

- **What:** Split `settings/schema.go` (1,071 lines) into: `core.go` (log, http, nats), `pipeline.go` (families, timeframes), `clickhouse.go` (connection, batching), `venue.go`, `validation.go` (cross-layer rules).
- **Why:** Single-file monolith. Changes to pipeline validation trigger recompilation of all services.
- **Evolution cost:** 2 — every family addition edits this file
- **Blast radius:** 2 — shared package, all services import
- **Risk:** 2 — validation logic is well-tested
- **Score: 6**
- **Estimated effort:** Small (1 session)
- **Dependencies:** None

---

## 5. LOW Priority — Defer to S214+

### L-01: EMA Naming Inconsistency (ND-05)

- **What:** Align `ema` (codegen) vs `ema_crossover` (domain/store) naming.
- **Why:** Config validation gap, but codegen is frozen so no expansion risk now.
- **Score: 4** — Low evolution cost, moderate blast radius, moderate risk (NATS consumer name migration)
- **When:** After codegen freeze lifts

### L-02: Evidence vs Candle Terminology (ND-01)

- **What:** Document the evidence/candle relationship clearly. Consider renaming for clarity.
- **Why:** Naming confusion exists but doesn't block any operation.
- **Score: 3** — Cosmetic, no functional impact
- **When:** During broader naming convention pass

### L-03: Float Formatting Consolidation (ND-03)

- **What:** Unify `FormatFloat()` and `parseFloat()` into a single shared utility.
- **Why:** Two implementations of the same logic, but both work correctly.
- **Score: 3** — Cosmetic
- **When:** Opportunistic, during reader/writer consolidation

### L-04: Store Supervisor Hardcoded Wiring (CP-04)

- **What:** Make store supervisor family-to-actor mapping data-driven.
- **Why:** 439 lines but well-structured. Wiring is explicit and correct.
- **Score: 4** — Moderate evolution cost, small blast radius
- **When:** After H-04 (generic actors make this trivial)

### L-05: Codegen Family Name Convention (ND-04)

- **What:** Document naming convention for codegen families.
- **Why:** Codegen frozen. No new families being added.
- **Score: 2** — Pure documentation
- **When:** When codegen freeze lifts

### L-06: Test Hardcoded Family Counts (TD-03)

- **What:** Replace ~10 hardcoded assertions like `assert.Len(families, 3)` with registry-driven counts.
- **Why:** Tests break when families are added, but family addition is frozen.
- **Score: 4** — Small effort, but no immediate value during freeze
- **When:** When family expansion resumes

---

## 6. Execution Order for S213

### Wave 1: Foundations (do first)
1. **H-06** — Module consolidation evaluation (affects all subsequent work)
2. **H-05** — Documentation entropy reduction (parallelizable with code work)

### Wave 2: NATS Layer (largest duplication cluster)
3. **H-01** — NATS adapter sub-packaging
4. **H-02** — Consumer spec factory
5. **M-05** — Generic consumer/publisher (if time)
6. **M-04** — Generic KV store (if time)

### Wave 3: Analytical Path
7. **H-03** — ClickHouse reader consolidation
8. **M-01** — Analytical use case consolidation (if time)

### Wave 4: Store + Handlers
9. **H-04** — Store actor generics
10. **M-02** — HTTP handler abstraction (if time)

### Wave 5: Writer + Gateway
11. **M-03** — Writer pipeline config-driven (if time)
12. **M-06** — Gateway compose simplification (if time)
13. **M-07** — Settings schema decomposition (if time)

---

## 7. Expected Outcomes After S213

### If Only HIGH Items Complete (6 items)

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Active architecture docs | 449 | ≤150 | -67% |
| NATS adapter navigability | 80+ files flat | 6+ sub-packages | Structured |
| Consumer spec duplication | 1,008 lines | ~200 lines | -80% |
| ClickHouse reader duplication | 884 lines | ~350 lines | -60% |
| Store actor duplication | 2,400 lines | ~600 lines | -75% |
| Go modules | 19 | ~10 | -47% |
| **Total recoverable lines** | | | **~4,100** |

### If HIGH + MEDIUM Items Complete (13 items)

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Structural duplication | ~10,100 lines | ~3,700 lines | -63% |
| Files per new family | 15+ | 5–7 | -60% |
| Largest single package | 10,110 lines | ~2,000 lines | -80% |
| Settings schema | 1,071 lines / 1 file | ~1,071 lines / 5 files | Navigable |

---

## 8. Guard Rails for Execution

1. **No functional changes** — refactoring only. All tests must pass before and after each item.
2. **Preserve NATS durable names** — consumer spec factory must generate identical durable names to current hardcoded values.
3. **Preserve ClickHouse queries** — reader consolidation must produce identical SQL.
4. **Preserve HTTP contracts** — handler abstraction must not change request/response shapes.
5. **No codegen template changes** — codegen markers and integrated slices must continue to validate.
6. **One item per commit** — each refactoring item is independently revertible.
7. **CI green gate** — no item is complete until CI passes.
