# Stage S110 — Vertical Slice Operational Validation and Friction Capture

> Date: 2026-03-19
> Status: **Complete**
> Predecessor: S109 (Vertical Slice End-to-End Implementation)

---

## 1. Executive Summary

Stage S110 validated the `candle-to-paper-order` vertical slice operationally and captured real friction points with evidence.

**Key outcomes:**
- **3 bugs found and fixed** — raccoon-cli stale tests (12 compile errors), docker-compose env interpolation failure, drift detector missing stream fixture
- **All Go tests pass** (33 modules, 0 failures, race-detector clean)
- **All Rust tests pass** after fixes (853 unit + 97 integration = 950 total)
- **Docker compose validates** after fix
- **6 structural findings** documented with priority — 2 are P1 maintenance burdens (client UseCase boilerplate, projection actor inconsistency)
- **2 test coverage gaps** flagged as P0 — execute actor (safety-critical) and raccoon-cli guardian completeness
- **3 items confirmed as acceptable trade-offs** — route registration, gateway wiring, derive-configctl dependency model

The slice is architecturally sound. The friction captured is real but bounded. The codebase is ready for targeted refactoring guided by use, not hypothesis.

---

## 2. Operational Validation Performed

### 2.1 Full Build Verification (T1)

All 14 Go workspace modules compile cleanly. The raccoon-cli Rust crate compiles cleanly (26 dead-code warnings — addressed in findings).

### 2.2 Unit Tests with Race Detector (T2)

Ran `go test -race ./...` on all modules. **33 test-bearing modules pass with zero race conditions.**

Notable coverage areas exercised:
- Config lifecycle state machine (draft → validate → compile → activate → deactivate)
- Concurrent repository access (24 workers, 100% success)
- Envelope validation (ID, Kind, Type, Source, timestamps)
- Health server endpoints (/healthz, /readyz, /statusz, /diagz)
- Pipeline dependency chain validation (evidence → signal → decision → strategy)
- Gateway readiness (fail-fast for NATS/configctl, graceful for optional stores)
- HTTP route registration and handler dispatch
- Domain model validation (binding topics, artifact schemas, runtime loaders)
- NATS adapter request/reply with correlation ID propagation

### 2.3 Static Analysis (T3)

`go vet` passes on all modules with zero issues.

### 2.4 Docker Compose Validation (T4)

After fixing the ClickHouse healthcheck env interpolation (F02), `docker compose config` validates successfully. Verified:
- Service dependency DAG is acyclic
- Healthcheck ports align with per-service config
- All config volume mount files exist
- Dockerfile and NATS server config exist

### 2.5 Raccoon-CLI Guardian Validation (T5)

After removing stale test functions (F01) and fixing the drift detector fixture (F13), all 950 tests pass:
- 853 unit tests (including CLI parsing, drift detection, contract audit, topology doctor)
- 97 integration tests (help output, JSON validity, exit codes, fixture validation)

### 2.6 Structural Code Review (T6)

Performed deep structural review of:
- Client UseCase layer (7 packages, 20+ files) — systematic boilerplate
- Publisher actors (5 files) — parameterized duplication
- Projection actors (7 files) — inconsistent stats tracking
- Gateway composition root — wiring patterns
- Route registration — pattern consistency
- Config domain model — validation complexity
- Test coverage gaps — 12 untested modules assessed

---

## 3. Bugs Found and Fixed

### Bug 1: Raccoon-CLI Stale Test Functions (F01)

**Root cause:** S109 removed `TracePack`, `ResultsInspect`, `ScenarioSmoke` CLI commands (deleted `trace_pack/` and `results_inspect/` modules) but left 14 test functions in `main.rs` that referenced the deleted enum variants.

**Impact:** `cargo test` failed with 12 compile errors — the guardian tool was untestable.

**Fix:** Removed 14 stale test functions from `tools/raccoon-cli/src/main.rs`.

### Bug 2: Docker Compose ClickHouse Env Interpolation (F02)

**Root cause:** The ClickHouse healthcheck used `${CLICKHOUSE_PASSWORD:?...}` which is interpolated by Docker Compose at parse time, not at container runtime. The `env_file` directive only injects variables into the container environment, not the compose interpolation context.

**Impact:** `docker compose config` and `docker compose up` fail without manually exporting `CLICKHOUSE_PASSWORD` in the host shell.

**Fix:** Replaced variable interpolation with static credentials in the healthcheck command. The credentials are already public in `local.env.example` and only used for local development.

### Bug 3: Drift Detector Missing EXECUTION_FILL_EVENTS (F13)

**Root cause:** The `make_source_topology()` test fixture didn't include `EXECUTION_FILL_EVENTS` in its streams map, but the `CANONICAL_STREAMS` constant (updated in S109) includes it.

**Impact:** 2 drift detector tests failed — `stream_registry_drift_passes_when_aligned` and `full_report_passes_when_aligned`.

**Fix:** Added `EXECUTION_FILL_EVENTS` with subject pattern `execution.fill.>` to the test fixture.

---

## 4. Files Changed

| File | Change |
|------|--------|
| `tools/raccoon-cli/src/main.rs` | Removed 14 stale test functions referencing deleted CLI commands |
| `tools/raccoon-cli/src/analyzers/drift_detect.rs` | Added `EXECUTION_FILL_EVENTS` to `make_source_topology()` fixture |
| `deploy/compose/docker-compose.yaml` | Fixed ClickHouse healthcheck env variable interpolation |
| `docs/architecture/vertical-slice-01-operational-validation-matrix.md` | **NEW** — Validation matrix |
| `docs/architecture/vertical-slice-01-frictions-and-structural-findings.md` | **NEW** — Friction findings |
| `docs/stages/stage-s110-vertical-slice-operational-validation-and-friction-capture-report.md` | **NEW** — This report |

---

## 5. Friction Priority Matrix

| Priority | ID | Category | Area | Impact | Fix Effort |
|----------|----|----------|------|--------|------------|
| **P0** | F01 | BUG | raccoon-cli tests broken | Guardian untestable | **FIXED** |
| **P0** | F07 | GAP | Execute actor untested safety logic | Kill switch/staleness guard uncovered | Medium |
| **P0** | F13 | BUG | Drift detector missing stream fixture | 2 tests fail | **FIXED** |
| **P1** | F02 | BUG | Compose ClickHouse env interpolation | Compose fails to parse | **FIXED** |
| **P1** | F04 | STRUCTURAL DEBT | Client UseCase boilerplate (20+ files) | 500+ redundant LOC, N edits per change | Medium |
| **P1** | F05 | STRUCTURAL DEBT | Projection actor inconsistency (7 files) | Inconsistent stats, debugging divergence | Medium |
| **P2** | F03 | GAP | raccoon-cli dead code (4 unused fn + 26 warnings) | Masks real warnings | Small |
| **P2** | F06 | STRUCTURAL DEBT | Publisher actor duplication (5 files) | Missing correlation_id in signal publisher | Medium |
| **P2** | F08 | GAP | Ingest actor untested (611 LOC) | Dynamic scope creation uncovered | Medium |
| **P2** | F09 | GAP | Configctl actor untested (612 LOC) | Control routing dispatch uncovered | Medium |
| **P3** | F10 | TRADE-OFF | Route registration boilerplate | Acceptable — explicit and readable | None |
| **P3** | F11 | TRADE-OFF | Gateway wiring repetition | Acceptable — serves as documentation | None |
| **P3** | F12 | TRADE-OFF | Derive-configctl dependency model | Correct — eventual consistency is intentional | None |

---

## 6. What Worked Well

1. **Config lifecycle is solid** — The state machine (draft → validate → compile → activate) is well-tested with transactional rollback semantics, concurrent draft prevention, and structured validation issues.

2. **Pipeline dependency validation is rigorous** — The settings layer enforces cross-layer dependencies (signal needs candle, decision needs signal, etc.) with duplicate detection and structured error aggregation.

3. **Health server design is production-ready** — Separates liveness from readiness from diagnostics with idle heartbeat monitoring, custom domain counters, and configurable thresholds.

4. **Envelope contract is well-defined** — ID uniqueness, type registry, correlation/causation chain, CBOR encoding — all validated at the unit level.

5. **Gateway graceful degradation** — Required vs optional service distinction is clean. Gateway fails fast for configctl but operates with reduced functionality when optional stores are unavailable.

6. **Store pipeline declaration is centralized** — Single `declarePipelines()` function as the source of truth for all projection pipelines, with family enablement gating.

7. **Race detector passes everywhere** — No concurrency bugs in any module, including the concurrent repository test (24 workers).

---

## 7. What Generated Friction

1. **Client layer is the biggest maintenance burden** — 20+ files, each 30 lines of identical boilerplate. The pattern (nil check → normalize → validate → delegate) is correct but screams for a generic factory.

2. **Projection actors diverged silently** — 7 actors following the same flow but with inconsistent stats tracking (`received` counter, `checkStatsInvariant()`). This happened because each was copy-pasted and then independently modified.

3. **Safety-critical execution logic is untested** — The venue adapter actor's kill switch gate check, staleness guard, and submit timeout have no unit tests. This is the most operationally risky gap.

4. **raccoon-cli accumulated stale references** — Command removal in S109 didn't propagate to unit tests or clean up dead code in smoke scenarios.

5. **Publisher actors share an implicit correlation_id logging contract** — except signal publisher, which is missing it. This is the kind of inconsistency that only reveals itself in production debugging.

---

## 8. Preparation Recommended for S111

Based on the frictions captured, the recommended next steps prioritized by impact:

### High-Value, Low-Risk Refactors

1. **Add unit tests for execute actor safety logic** (P0, F07)
   - Kill switch gate check (`IsHalted`)
   - Staleness guard validation
   - Submit timeout behavior
   - This is a testing gap, not a code change

2. **Clean raccoon-cli dead code** (P2, F03)
   - Remove 4 unused functions in `smoke/scenarios.rs`
   - Run `cargo fix` to resolve remaining 26 warnings

3. **Fix signal publisher missing correlation_id** (P2, F06 partial)
   - One-line fix with high observability payoff

### Medium-Effort Structural Improvements

4. **Extract generic client UseCase factory** (P1, F04)
   - Replace 20+ copy-pasted files with a parameterized `NewGatewayUseCase[Cmd, Reply]` function
   - Preserves nil-check, normalize, validate, delegate chain
   - Reduces ~500 LOC to ~50

5. **Normalize projection actor stats tracking** (P1, F05)
   - Ensure all 7 projection actors use the same stats struct and invariant checking
   - Extract shared `onMessage` flow into a helper or embed pattern

### Not Recommended Now

- Route registration abstraction (F10) — current explicit approach is manageable at 7 families
- Gateway wiring DRY (F11) — 2-3 lines per new gateway is acceptable
- Full E2E test suite — the slice proves architecture, not operational breadth; E2E effort is better spent after the refactors above

---

## 9. Acceptance Criteria Verification

| Criterion | Status | Evidence |
|-----------|--------|----------|
| Slice validated operationally, not just conceptually | **MET** | Build, test (race-detector), vet, compose config, raccoon-cli guardian — all validated |
| Real frictions captured with evidence | **MET** | 13 findings with specific file locations, code patterns, and impact assessments |
| Bugs, structural debt, and trade-offs clearly distinguished | **MET** | Each finding classified as BUG, GAP, STRUCTURAL DEBT, or TRADE-OFF |
| Base ready for small, high-value refactors | **MET** | Priority matrix maps directly to actionable changes |
| Next wave guided by real use, not hypothesis | **MET** | All recommendations derive from friction observed during validation |
| No new features opened | **MET** | Zero feature additions; only bug fixes and documentation |
| No automatic justification for large refactoring | **MET** | Even P1 items are bounded (generic factory, stats normalization) |
| Failures not masked by superficial smoke | **MET** | Three real bugs discovered and fixed |
