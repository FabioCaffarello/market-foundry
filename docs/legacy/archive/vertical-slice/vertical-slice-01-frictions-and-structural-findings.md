# Vertical Slice 01 — Frictions and Structural Findings

> Stage S110 — Friction capture from operational validation.
> Date: 2026-03-19

---

## Classification Legend

| Category | Definition |
|----------|-----------|
| **BUG** | Code that is broken right now and must be fixed |
| **GAP** | Missing coverage or capability the slice exposes |
| **STRUCTURAL DEBT** | Real maintenance burden revealed by the slice |
| **TRADE-OFF** | Intentional limitation; acceptable for current stage |

| Priority | Criteria |
|----------|---------|
| **P0** | Blocks runtime validation or represents safety risk |
| **P1** | High maintenance burden; affects every new domain addition |
| **P2** | Moderate burden; affects specific areas |
| **P3** | Minor friction; acceptable in current scope |

---

## Finding F01: Raccoon-CLI Test Compilation Broken

**Category:** BUG
**Priority:** P0
**Location:** `tools/raccoon-cli/src/main.rs:1450-1736`

**Evidence:** `cargo test` fails with 12 compile errors. Unit tests in `main.rs` reference `Commands::TracePack`, `Commands::ResultsInspect`, and `Commands::ScenarioSmoke` — enum variants that were removed during S109 cleanup (the corresponding modules `trace_pack/` and `results_inspect/` were deleted).

**Impact:** raccoon-cli test suite is completely broken. Cannot verify guardian tool correctness.

**Fix:** Remove or update the 12 test functions that reference deleted variants.

---

## Finding F02: Docker Compose ClickHouse Env Interpolation

**Category:** BUG
**Priority:** P1
**Location:** `deploy/compose/docker-compose.yaml:80`

**Evidence:** `docker compose config` fails with:
```
required variable CLICKHOUSE_PASSWORD is missing a value: Set CLICKHOUSE_PASSWORD in env file
```

The healthcheck uses `${CLICKHOUSE_PASSWORD:?Set CLICKHOUSE_PASSWORD in env file}` which requires the variable in the compose shell environment, not just in the env_file. The `env_file` directive (line 78) loads variables into the container, but `${...}` in the `healthcheck.test` field is interpolated by Docker Compose at parse time, before the container starts.

**Impact:** `docker compose config` and `docker compose up` fail without manually exporting `CLICKHOUSE_PASSWORD` in the host shell.

**Fix:** Either use a static password in the healthcheck command, or use a `.env` file at the compose file's directory level (not `env_file`), or use `docker compose --env-file` flag.

---

## Finding F03: Raccoon-CLI Dead Code Accumulation

**Category:** GAP
**Priority:** P2
**Location:** `tools/raccoon-cli/src/smoke/scenarios.rs:594-752`

**Evidence:** 4 functions are unused: `run_missing_binding`, `run_readiness_probe`, `run_stages_sequential`, `skip_remaining`. Additionally, 26 compiler warnings for dead code across the crate. The `smoke/scenarios.rs` module has 942 lines but significant portions are unreachable.

**Impact:** Increases cognitive load and masks real warnings.

**Fix:** Either wire the unused scenarios into the CLI dispatch or remove them.

---

## Finding F04: Client UseCase Boilerplate — Severe Duplication

**Category:** STRUCTURAL DEBT
**Priority:** P1
**Location:** `internal/application/*client/` (7 packages, 20+ files)

**Evidence:** Every client UseCase follows an identical 30-line pattern:
```go
type {Name}UseCase struct { gateway ports.{Family}Gateway }
func New{Name}UseCase(gateway) *{Name}UseCase { ... }
func (uc *{Name}UseCase) Execute(ctx, cmd) (reply, *problem.Problem) {
    if uc == nil || uc.gateway == nil { return unavailable }
    cmd = cmd.Normalize()
    if prob := cmd.Validate(); prob != nil { return prob }
    return uc.gateway.{Method}(ctx, cmd)
}
```

The `configctlclient` alone has 10 files — each ~31 lines of near-identical code. The `evidenceclient`, `signalclient`, `decisionclient`, `strategyclient`, `riskclient`, `executionclient` repeat the same structure with minor name variations.

**Impact:** Adding a new client operation requires copy-pasting an entire file. Changing the nil-check or validation pattern requires 20+ edits. Estimated 500+ lines of pure redundancy.

**Why this matters now:** The vertical slice activated all 7 client packages. If the next slice adds families, this pattern multiplies.

---

## Finding F05: Projection Actor Duplication and Inconsistency

**Category:** STRUCTURAL DEBT
**Priority:** P1
**Location:** `internal/actors/scopes/store/` (7 projection actors)

**Evidence:** All 7 projection actors (`candle`, `decision`, `risk`, `signal`, `strategy`, `trade_burst`, `volume`) follow the same 150-200 line flow:
1. Final check
2. Domain validation
3. Context with timeout
4. KV Put with result switch (PutSkippedStale, PutSkippedDuplicate, PutWritten)
5. Tracker recording

However, the implementations diverge in subtle, inconsistent ways:

| Aspect | Candle | Decision | Risk | Signal | Strategy | TradeBurst | Volume |
|--------|--------|----------|------|--------|----------|------------|--------|
| Has `received` counter | NO | YES | YES | NO | YES | NO | NO |
| Has `checkStatsInvariant()` | NO | NO | YES | NO | YES | NO | NO |
| Logger initialization | Generic | With labels | With labels | With labels | With labels | Generic | Generic |

**Impact:** Adding a new projection type requires choosing which variant to copy. Debugging is inconsistent — some projections have stats invariant checks, others don't. The inconsistency looks accidental, not intentional.

---

## Finding F06: Publisher Actor Duplication

**Category:** STRUCTURAL DEBT
**Priority:** P2
**Location:** `internal/actors/scopes/derive/` (5 publisher actors)

**Evidence:** `decision_publisher_actor.go`, `risk_publisher_actor.go`, `signal_publisher_actor.go`, `strategy_publisher_actor.go`, and `publisher_actor.go` (evidence) share nearly identical `Receive()` implementations. The only differences are the domain type names and message struct types.

Additional inconsistency: `signal_publisher_actor.go` is missing the `correlation_id` log field that all others include.

The evidence publisher (`publisher_actor.go`) handles 3 message types (candle, tradeburst, volume) with the same 5-second timeout and error logging pattern repeated 3 times within a single file.

**Impact:** Timeout changes, logging format updates, or error handling improvements require 5+ file edits. The missing correlation_id in signal publisher silently degrades observability.

---

## Finding F07: Execute Actor — No Unit Tests for Safety-Critical Logic

**Category:** GAP
**Priority:** P0
**Location:** `internal/actors/scopes/execute/venue_adapter_actor.go` (350 LOC)

**Evidence:** The venue adapter actor contains:
- **Kill switch gate check** — reads `ExecutionControlKVStore.IsHalted(ctx)` before submitting orders
- **Staleness guard** — rejects intents older than configured threshold
- **Submit timeout** — 2-second context deadline for gate checks

None of this logic has unit tests. The `internal/actors/scopes/execute` package has `[no test files]`.

**Impact:** The execution boundary to venue adapters is safety-critical. A regression in kill switch logic could result in unintended order submission. The staleness guard silently skips stale intents without logging source/symbol details.

---

## Finding F08: Ingest Actor — No Unit Tests for Dynamic Scope Creation

**Category:** GAP
**Priority:** P2
**Location:** `internal/actors/scopes/ingest/` (611 LOC, no test files)

**Evidence:** The ingest supervisor dynamically creates exchange scope actors, which in turn spawn WebSocket adapters per symbol. The binding watcher actor queries configctl and translates binding changes into supervisor messages. None of this actor lifecycle logic is unit-tested.

**Impact:** Dynamic scope creation involves 3 actor hops (supervisor → exchange scope → websocket). Failures in middle hops would not be caught before runtime.

---

## Finding F09: Configctl Actor Scope — No Unit Tests

**Category:** GAP
**Priority:** P2
**Location:** `internal/actors/scopes/configctl/` (612 LOC, no test files)

**Evidence:** The configctl actor scope includes control routing dispatch (10 handler methods), request/reply error mapping via generic `requestActor` function, and lazy initialization patterns (`ensureDefaults`). None of this is unit-tested.

**Impact:** The configctl actor scope is exercised indirectly via adapter tests (`configctl_gateway_test.go`), but actor-specific routing and error mapping logic is uncovered.

---

## Finding F10: Route Registration Boilerplate

**Category:** TRADE-OFF
**Priority:** P3
**Location:** `internal/interfaces/http/routes/` (7 route files)

**Evidence:** Each domain family has its own route builder function following identical structure: check `HasAny()`, conditionally register routes, create handler. No shared abstraction.

**Impact:** Adding a new family means writing a new route file from scratch. However, the current approach is explicit and readable. The duplication is manageable at current scale (7 families).

**Verdict:** Acceptable trade-off for now. Revisit if family count exceeds 12.

---

## Finding F11: Gateway Wiring Repetition

**Category:** TRADE-OFF
**Priority:** P3
**Location:** `cmd/gateway/compose.go:45-112`

**Evidence:** `buildGatewayConns()` creates 8 gateway connections with identical pattern but no iteration. Each connection is 2-3 lines of setup.

**Impact:** Low — adding a new gateway type requires 2-3 lines. Pattern is clear and readable.

**Verdict:** Acceptable trade-off. The explicit wiring serves as documentation.

---

## Finding F12: Compose Dependency on configctl for Derive

**Category:** TRADE-OFF
**Priority:** P3
**Location:** `deploy/compose/docker-compose.yaml:145-163`

**Evidence:** The `derive` service depends only on `nats` (not on `configctl`). In practice, derive needs configctl to receive binding activations. It works because derive's BindingWatcherActor queries configctl on startup — if configctl isn't ready, derive starts in an idle state and catches up when configctl publishes `IngestionRuntimeChangedEvent`.

**Impact:** This is actually correct behavior — derive should not hard-depend on configctl for startup. The eventual consistency model is intentional.

**Verdict:** Not a problem. The current dependency model correctly separates startup readiness from operational readiness.

---

## Friction Priority Matrix

| Priority | ID | Category | Area | Fix Effort |
|----------|----|----------|------|------------|
| **P0** | F01 | BUG | raccoon-cli tests broken | Small — delete stale test functions |
| **P0** | F07 | GAP | Execute actor untested safety logic | Medium — write unit tests for kill switch + staleness |
| **P1** | F02 | BUG | Compose ClickHouse env interpolation | Small — fix healthcheck variable handling |
| **P1** | F04 | STRUCTURAL DEBT | Client UseCase boilerplate | Medium — introduce generic UseCase factory |
| **P1** | F05 | STRUCTURAL DEBT | Projection actor inconsistency | Medium — normalize stats tracking, extract shared flow |
| **P2** | F03 | GAP | raccoon-cli dead code | Small — delete unused functions |
| **P2** | F06 | STRUCTURAL DEBT | Publisher actor duplication | Medium — extract parameterized publisher |
| **P2** | F08 | GAP | Ingest actor untested | Medium — write binding lifecycle tests |
| **P2** | F09 | GAP | Configctl actor untested | Medium — write routing dispatch tests |
| **P3** | F10 | TRADE-OFF | Route registration boilerplate | None — acceptable |
| **P3** | F11 | TRADE-OFF | Gateway wiring repetition | None — acceptable |
| **P3** | F12 | TRADE-OFF | Derive-configctl dependency model | None — correct behavior |
