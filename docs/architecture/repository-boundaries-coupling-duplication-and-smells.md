# Repository Boundaries, Coupling, Duplication, and Structural Smells

> Stage: S212 — Repository Architecture Census and Refactor Map
> Date: 2026-03-20
> Status: Active
> Companion to: repository-architecture-census-and-refactor-map.md

---

## 1. Purpose

This document catalogs every identified structural smell, coupling issue, duplication pattern, and naming debt item in the repository. Each item is classified as **structural** (affects evolution cost) or **cosmetic** (low impact, not worth refactoring now).

---

## 2. Structural Duplication — High Impact

### SD-01: NATS Adapter Package — Flat Mega-Package

- **Location:** `internal/adapters/nats/` — 80+ files, 10,110 lines
- **Problem:** All NATS concerns (registries, consumers, publishers, gateways, KV stores, codecs, connection) live in a single flat package with no sub-packaging.
- **Impact:** Hard to navigate, hard to reason about responsibility scope, naming collisions risk grows with each family.
- **Classification:** Structural

### SD-02: NATS Registry Consumer Specs — Template Duplication

- **Location:** `internal/adapters/nats/*_registry.go` — 1,008 lines across 8 files
- **Problem:** Each consumer spec function is 11–12 lines of boilerplate differing only in durable name, subject, and stream name. 25+ consumer functions with ~75% structural duplication.
- **Example:** `WriterRSISignalConsumer()` vs `StoreRSISignalConsumer()` differ only in durable prefix ("writer-" vs "store-").
- **Impact:** Adding a family requires 2–4 new consumer functions per registry file, each copy-pasted. Error-prone.
- **Classification:** Structural

### SD-03: ClickHouse Readers — Near-Identical Query Implementations

- **Location:** `internal/adapters/clickhouse/*_reader.go` — 884 lines across 6 files
- **Problem:** Each reader replicates: constructor (11 lines), query method (65+ lines), timing instrumentation, error handling, row iteration, logging. ~70% structural duplication.
- **Secondary issue (from TD-02):** Query methods accept 10+ positional parameters (source, symbol, timeframe, limit, from, to, type/outcome/etc.).
- **Impact:** Adding a field or changing query semantics requires 6 parallel edits. Positional parameters are error-prone.
- **Classification:** Structural

### SD-04: Store Consumer Actors — Per-Family Copy-Paste

- **Location:** `internal/actors/scopes/store/*_consumer_actor.go` — 8 files, 82–91 lines each
- **Problem:** Each consumer actor follows identical structure (constructor, Receive handler, start/stop logic) with only the domain type and consumer spec varying.
- **Impact:** 6,296 lines in store scope; consumer actors alone account for ~700 lines of near-identical code.
- **Classification:** Structural

### SD-05: Store Projection Actors — Structural Mirrors

- **Location:** `internal/actors/scopes/store/*_projection_actor.go` — 8 files, 191–282 lines each
- **Problem:** Each projection actor implements identical: KV store update, deduplication check, counter increment, error logging. Only the domain type and store interface vary.
- **Impact:** ~1,700 lines of projection actors with ~60% structural duplication.
- **Classification:** Structural

### SD-06: Analytical Use Cases — Identical Execute() Skeletons

- **Location:** `internal/application/analyticalclient/get_*_history.go` — 6 files, 97 lines each (584 total)
- **Problem:** Each use case replicates: nil checks, input validation, query execution, error handling, logging, response construction. Only the domain type and validation rules differ.
- **Impact:** Adding a new analytical family requires copy-pasting 97 lines and adjusting types.
- **Classification:** Structural

### SD-07: HTTP Handlers — Parallel Structural Skeletons

- **Location:** `internal/interfaces/http/handlers/{signal,decision,strategy,risk}.go` — 4 files, 60–61 lines each
- **Problem:** 95% structural duplication. Each handler: use case interface (3 lines), struct (3 lines), constructor (3 lines), response struct (3 lines), handler method (30 lines). Differ only in type names.
- **Extended:** Analytical handler (502 lines) has 6 query methods with 60% structural duplication.
- **Impact:** 4 simple handlers + 1 analytical handler = ~740 lines with ~65% redundancy.
- **Classification:** Structural

### SD-08: Writer Pipeline Entries — Hardcoded Struct Literals

- **Location:** `cmd/writer/pipeline.go` — 248 lines, 7 pipeline entries
- **Problem:** Each entry is 22–24 lines of identical structure. Only 2 of 7 are under codegen. The remaining 5 are manual copy-paste.
- **Classification:** Structural

### SD-09: Writer Mappers — Repetitive Row Extraction

- **Location:** `cmd/writer/mappers.go` — 203 lines, 6 mapper functions
- **Problem:** Each mapper extracts metadata + domain fields into `[]any` slice. 90% structural duplication across mappers. Only field list and helper calls vary.
- **Classification:** Structural

### SD-10: Gateway Compose — Wiring Boilerplate

- **Location:** `cmd/gateway/compose.go` — 246 lines
- **Problem:** 8 `newGatewayConn()` calls + 7 conditional wiring blocks. Each block follows identical pattern: check connection, create use case, assign to deps struct. ~70% boilerplate.
- **Impact:** Adding a new query family requires 3 new blocks of wiring code.
- **Classification:** Structural

---

## 3. Coupling Issues — Cross-Layer

### CP-01: Family Addition Blast Radius

- **Problem:** Adding a new pipeline family requires coordinated changes across 8+ files:
  1. `internal/domain/{family}/` — new entity
  2. `internal/adapters/nats/{family}_registry.go` — consumer specs
  3. `internal/adapters/nats/{family}_consumer.go` — consumer implementation
  4. `internal/adapters/nats/{family}_publisher.go` — publisher
  5. `internal/adapters/nats/{family}_kv_store.go` — KV projection
  6. `internal/application/{family}client/` — query use cases
  7. `internal/interfaces/http/handlers/{family}.go` — HTTP handler
  8. `internal/interfaces/http/routes/{family}.go` — route registration
  9. `cmd/gateway/compose.go` — wiring
  10. `cmd/writer/pipeline.go` — pipeline entry
  11. `cmd/writer/mappers.go` — row mapper
  12. `internal/adapters/clickhouse/{family}_reader.go` — analytical reader
  13. `internal/application/analyticalclient/get_{family}_history.go` — analytical use case
  14. `internal/shared/settings/schema.go` — config validation
  15. `deploy/migrations/` — ClickHouse table
- **Impact:** 15+ files per family. High coordination cost, error-prone.
- **Classification:** Structural — primary architectural pressure point

### CP-02: Settings Schema Monolith

- **Location:** `internal/shared/settings/schema.go` — 1,071 lines
- **Problem:** All config sections (log, http, nats, venue, pipeline, clickhouse), all validation rules, all known-family registries, and all default values live in one file. Changes to pipeline validation affect compile-time of unrelated services.
- **Classification:** Structural (moderate)

### CP-03: Analytical Client ↔ ClickHouse Reader 1:1 Coupling

- **Problem:** Each `analyticalclient/get_{family}_history.go` use case depends on exactly one `clickhouse/{family}_reader.go`. The use case adds almost no logic beyond nil checks and validation before delegating to the reader.
- **Question:** Is the use case layer adding value or just indirection?
- **Classification:** Structural (moderate — may be acceptable for consistency)

### CP-04: Store Supervisor Family Wiring

- **Location:** `internal/actors/scopes/store/store_supervisor.go` — 439 lines
- **Problem:** Supervisor hardcodes family-to-actor mapping. Each family requires a new block in the supervisor's spawn logic.
- **Classification:** Structural (moderate)

### CP-05: NATS Consumer Naming Convention Coupling

- **Problem:** Consumer durable names (e.g., "writer-signal-rsi", "store-signal-rsi") encode service identity. This creates an implicit contract between registry definitions and runtime consumer identity.
- **Risk:** Renaming or restructuring services requires migration of NATS durable consumer offsets.
- **Classification:** Structural (low risk now, high cost if triggered)

---

## 4. Naming Debt

### ND-01: "Evidence" vs "Candle" Terminology Ambiguity

- **Scope:** Domain-wide
- **Problem:** "Evidence" is the domain boundary (evidence layer contains candle, tradeburst, volume). But most code refers to "candle" directly: `CandleLatest`, `CandleHistory`, `candle_reader.go`, `evidence_candles` (ClickHouse table).
- **Impact:** Unclear whether "evidence" means all evidence types or specifically candles. New developers will confuse the terms.
- **Classification:** Naming debt — structural (affects boundary clarity)

### ND-02: "Observation" vs "Trade" Separation

- **Scope:** Domain boundary
- **Problem:** Raw market data is `ObservationTrade` (domain/observation) while processed data is `EvidenceCandle` (domain/evidence). The relationship between observation and evidence is implicit — no documentation or naming convention explains the transformation boundary.
- **Classification:** Naming debt — cosmetic (boundary is clear in code, just poorly named)

### ND-03: Float Formatting Function Naming

- **Scope:** Read/write path
- **Problem:** `FormatFloat()` (exported, in clickhouse/candle_reader.go) vs `parseFloat()` (unexported, in cmd/writer/mappers.go). Same logical operation (string ↔ float conversion) with different names and visibility in read vs write paths.
- **Classification:** Naming debt — cosmetic (functional correctness not affected)

### ND-04: Codegen Family Names vs Domain Names

- **Scope:** codegen/families/ YAML files
- **Problem:** Inconsistent naming convention: indicators (`rsi`, `ema`), outcomes (`rsi_oversold`), strategies (`mean_reversion_entry`), mechanics (`paper_order`, `position_exposure`). No naming convention documented.
- **Classification:** Naming debt — cosmetic (codegen frozen, no expansion risk now)

### ND-05: EMA Naming Inconsistency

- **Scope:** Signal family
- **Problem:** Codegen family is named `ema` but domain signal type is `ema_crossover`. Registry references both names. Writer config uses `ema`, store config uses `ema_crossover`.
- **Classification:** Naming debt — structural (config validation and consumer routing affected)

---

## 5. Structural Smells

### SS-01: Module Over-Fragmentation

- **Problem:** 19 Go modules where 7–8 would suffice. The 4 adapter modules (`clickhouse`, `exchanges`, `nats`, `repositories`) share no conflicting dependencies yet are separate modules. All 8 cmd/ modules have identical dependency patterns.
- **Cost:** go.sum maintenance overhead, workspace complexity, longer `go work sync` operations.
- **Classification:** Structural — quantified in TD registry as AD-01

### SS-02: Settings Schema As Global Knowledge Hub

- **Problem:** `settings/schema.go` knows about every family in every layer (evidence, signal, decision, strategy, risk, execution). It maintains canonical family registries, cross-layer dependency rules, and default values. Any new family type requires editing this file.
- **Cost:** Single point of change for unrelated concerns.
- **Classification:** Structural

### SS-03: Analytical Handler Size

- **Location:** `internal/interfaces/http/handlers/analytical.go` + `analytical_test.go` — 502+ lines
- **Problem:** Handler was 502 lines before `parseAnalyticalParams()` extraction in S210. Still contains 6 nearly-identical query methods.
- **Cost:** High per-method duplication, hard to maintain consistently.
- **Classification:** Structural (partially addressed in S210)

### SS-04: NATS KV Store Duplication

- **Location:** `internal/adapters/nats/*_kv_store.go` — 9 files, 120–271 lines each
- **Problem:** Each KV store implements: Put (encode + store), Get (fetch + decode), List (iterate + decode). ~60% structural duplication across stores.
- **Total:** ~1,400 lines of KV store code.
- **Classification:** Structural

### SS-05: NATS Consumer/Publisher Duplication

- **Location:** `internal/adapters/nats/*_consumer.go` + `*_publisher.go`
- **Problem:** 10 consumer files (~120 lines each) + 8 publisher files (~100-160 lines each). Each follows identical pattern. Consumers: subscribe, decode, forward. Publishers: encode, publish, ack.
- **Total:** ~2,400 lines.
- **Classification:** Structural

### SS-06: Documentation Entropy

- **Problem:** 449 architecture docs, many superseded or redundant. 208 stage reports with no index.
- **Cost:** Misleading or contradictory information. Search friction for canonical decisions.
- **Classification:** Structural — execution plan exists (S209 entropy map)

---

## 6. Non-Issues — Items That Look Like Smells But Are Not

### NI-01: Domain Input Types (Decision.SignalInput, etc.)

- **Appearance:** Looks like duplication — `Signal` and `Decision.SignalInput` have similar fields.
- **Reality:** Intentional decoupling. Each domain owns its predecessor's input representation. This prevents cross-domain imports and allows independent evolution.
- **Verdict:** Correct architectural pattern. Do not consolidate.

### NI-02: Per-Family Actor Pairs in Store

- **Appearance:** 8 consumer + 8 projection actors looks redundant.
- **Reality:** Each actor has independent lifecycle, supervision, and failure isolation. The pattern is correct for fault tolerance.
- **Verdict:** Duplication is structural (SD-04, SD-05) but the actor-per-family pattern itself is sound. Consolidate implementation, keep topology.

### NI-03: Separate Operational and Analytical Query Paths

- **Appearance:** Two separate systems for querying the same data (NATS KV vs ClickHouse).
- **Reality:** Different SLAs (latency vs completeness), different data lifetimes, different query patterns. CQRS boundary is intentional.
- **Verdict:** Correct architectural decision. Do not merge.

### NI-04: Settings OrDefault() Methods

- **Appearance:** Many small methods like `BatchSizeOrDefault()`, `FlushIntervalOrDefault()`.
- **Reality:** Defensive defaults prevent zero-value bugs. Pattern is idiomatic Go.
- **Verdict:** Keep. Not debt.

### NI-05: Multiple go.mod Files for cmd/ Binaries

- **Appearance:** 8 separate modules for 8 binaries.
- **Reality:** This is a valid Go workspace pattern. However, since all binaries share the same internal dependency graph and none has conflicting version requirements, consolidation into fewer modules is viable (see SS-01). The pattern is not wrong, just suboptimal.
- **Verdict:** Optimization opportunity, not a smell per se.

---

## 7. Quantified Duplication Summary

| ID | Component | Lines | Duplication % | Recoverable Lines |
|----|-----------|-------|---------------|-------------------|
| SD-02 | NATS registries | 1,008 | 75% | ~700 |
| SD-03 | ClickHouse readers | 884 | 70% | ~530 |
| SD-04+05 | Store consumer+projection actors | ~2,400 | 60% | ~1,440 |
| SD-06 | Analytical use cases | 584 | 85% | ~496 |
| SD-07 | HTTP handlers (simple + analytical) | ~740 | 65% | ~480 |
| SD-08+09 | Writer pipeline + mappers | 451 | 88% | ~397 |
| SD-10 | Gateway compose | 246 | 70% | ~172 |
| SS-04 | NATS KV stores | ~1,400 | 60% | ~840 |
| SS-05 | NATS consumers + publishers | ~2,400 | 55% | ~1,320 |
| **Total** | | **~10,113** | **~66%** | **~6,375** |

**Bottom line:** ~6,375 lines of structurally recoverable duplication out of 57,289 total lines (11% of codebase).
