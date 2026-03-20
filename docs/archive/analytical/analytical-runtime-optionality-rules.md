# Analytical Runtime Optionality Rules

> **Stage:** S143 — Migrations and ClickHouse Entry Architecture
> **Status:** Definitive
> **Scope:** Invariant rules for preserving ClickHouse optionality at runtime.

---

## 1. Purpose

This document codifies the rules that ensure ClickHouse remains optional throughout all phases of its integration into Market Foundry. These rules are **invariants** — they must hold at every commit, every phase, and every future expansion. Violating any rule means the system has regressed from its architectural contract.

The rules exist because the operational pipeline (ingest → derive → store → execute) was designed, validated, and proven without ClickHouse. ClickHouse is an analytical augmentation. The operational pipeline must never depend on it.

---

## 2. The Optionality Principle

> **P-01 (restated):** The pipeline must function without ClickHouse. No service may add ClickHouse to its readiness checks. No event path may block on ClickHouse availability. ClickHouse is an analytical augmentation, not an operational dependency.

This principle was defined in S141 and validated as PASSED (PC-04) in S142. The rules below operationalize it into enforceable constraints.

---

## 3. Rules

### R-01: No Operational Service Depends on ClickHouse

**Rule:** The services `configctl`, `gateway`, `ingest`, `derive`, `store`, and `execute` MUST NOT have ClickHouse in their dependency chain — not in docker-compose `depends_on`, not in readiness checks, not in configuration requirements, not in import paths.

**Verification:**
```bash
# docker-compose: no operational service depends on clickhouse
grep -A5 'depends_on' deploy/compose/docker-compose.yaml | grep clickhouse
# Expected: only writer service has clickhouse dependency

# Go imports: no operational service imports clickhouse driver
grep -r "clickhouse" cmd/configctl/ cmd/gateway/ cmd/ingest/ cmd/derive/ cmd/store/ cmd/execute/
# Expected: zero matches
```

**Exception:** None. This rule has no exceptions.

---

### R-02: No Readiness Check References ClickHouse (Except Writer)

**Rule:** The `/readyz` endpoint of every operational service checks NATS connectivity only. ClickHouse is not part of any readiness check except `cmd/writer`.

**Verification:**
```go
// In any operational service's health setup:
healthz.NewHealthServer(addr, []healthz.ReadinessCheck{
    bootstrap.NATSReadinessCheck(config),
    // NO ClickHouse check here
}, trackers)
```

**Rationale:** If ClickHouse is in a readiness check, the service will report unhealthy when ClickHouse is down, which could trigger orchestrator restarts or traffic routing changes.

---

### R-03: No Event Path Blocks on ClickHouse

**Rule:** No NATS message handler in the operational pipeline may perform a synchronous ClickHouse operation. The flow from event receipt to event processing to event publish must be ClickHouse-free.

**Verification:** Code review of all message handlers in `internal/actors/scopes/` for services other than writer. No ClickHouse client calls.

**Rationale:** A ClickHouse timeout or connection failure must never cause NATS message processing to stall, back up, or fail.

---

### R-04: Writer Uses Independent Consumer Names

**Rule:** The writer's NATS durable consumer names MUST use a `writer-` prefix, never sharing consumer names with `store` or any other service.

| Service | Consumer Name Pattern |
|---------|----------------------|
| store | `store-{family}-{type}-consumer` |
| writer | `writer-{family}-{type}-consumer` |

**Verification:**
```bash
grep -r "ConsumerName\|DurableName\|Durable:" internal/actors/scopes/writer/
# All must start with "writer-"
```

**Rationale:** Shared consumer names mean events are load-balanced between consumers. If writer shares with store, store would miss events when writer is running, and vice versa. Each consumer must have independent position tracking.

---

### R-05: Writer Tolerates ClickHouse Absence

**Rule:** `cmd/writer` MUST start successfully and maintain NATS connections even when ClickHouse is unreachable. It buffers events up to `max_pending`, drops oldest beyond that threshold, and resumes normal operation when ClickHouse becomes available.

**Behavior matrix:**

| ClickHouse State | Writer Behavior | NATS Consumer | Health |
|-----------------|-----------------|---------------|--------|
| Healthy | Normal: consume → buffer → flush | Advancing | Healthy |
| Unreachable | Buffering: consume → buffer (no flush) | Advancing | Degraded |
| Unreachable + buffer full | Dropping: consume → drop oldest → buffer newest | Advancing | Unhealthy |
| Recovered after outage | Resume: flush buffer → normal | Advancing | Healthy |

**Critical:** The NATS consumer NEVER stops advancing. Events that are dropped during a ClickHouse outage can be replayed from NATS stream retention (72h) if needed, but the writer does not attempt this automatically.

---

### R-06: Smoke Tests Pass Without ClickHouse and Writer

**Rule:** The existing smoke tests (`smoke-first-slice.sh`, `smoke-multi-symbol.sh`) MUST pass with ClickHouse and writer containers stopped or absent.

**Verification:**
```bash
# Stop ClickHouse and writer (if running)
docker compose -f deploy/compose/docker-compose.yaml stop clickhouse writer

# Run smoke tests — must pass
./scripts/smoke-first-slice.sh
./scripts/smoke-multi-symbol.sh
```

**Rationale:** This is the functional definition of optionality. If smoke tests fail without ClickHouse, the pipeline has an undeclared dependency.

---

### R-07: No Conditional Behavior in Operational Services

**Rule:** Operational services MUST NOT contain code paths that behave differently based on ClickHouse availability. There are no `if clickhouseEnabled` branches in `ingest`, `derive`, `store`, `execute`, or `gateway`.

**Rationale:** Conditional paths create testing combinatorics (with CH / without CH) and tend to degrade over time as the "without CH" path gets less testing.

**Exception:** The gateway MAY have new endpoints (R-08) that are ClickHouse-dependent, but these are additive, not modifications of existing behavior.

---

### R-08: Historical Endpoints Are Additive

**Rule:** When ClickHouse-backed query endpoints are added to the gateway (future), they MUST be new routes, not modifications of existing routes. Existing routes continue to serve from NATS KV regardless of ClickHouse state.

| Route Type | Source | ClickHouse Required |
|-----------|--------|-------------------|
| Existing: `GET /evidence/candle/{source}/{symbol}/{tf}` | NATS KV | **No** — unchanged |
| Future: `GET /evidence/candle/{source}/{symbol}/{tf}/history` | ClickHouse | **Yes** — returns 503 if CH down |

**Rationale:** Modifying existing routes to conditionally use ClickHouse would violate R-07 and make the gateway's behavior dependent on ClickHouse state.

---

### R-09: Cold-Start Bootstrap Is Opportunistic

**Rule:** When derive gains the ability to bootstrap from ClickHouse (future), the bootstrap MUST be opportunistic:
- If ClickHouse is available and has data → use it (faster warm-up)
- If ClickHouse is unavailable or empty → fall back to current behavior (wait for live candles)
- Bootstrap MUST NOT block derive startup
- Bootstrap MUST have a timeout (suggested: 5 seconds)

**Rationale:** Derive must always start. ClickHouse bootstrap is a performance optimization, not a correctness requirement.

---

### R-10: Configuration Does Not Require ClickHouse

**Rule:** The configuration lifecycle (draft → validate → compile → activate via configctl) has no ClickHouse dependency. Config validation does not check ClickHouse schema. Config activation does not trigger migrations.

**Rationale:** Configuration and schema are independent concerns. Mixing them creates a coupling where a ClickHouse outage blocks configuration changes.

---

## 4. Enforcement Mechanisms

### 4.1 Structural Enforcement (Strongest)

| Mechanism | What It Prevents |
|-----------|-----------------|
| Separate `cmd/writer/` service | ClickHouse code can't leak into other services |
| No ClickHouse driver in operational services' `go.mod` imports | Compile-time prevention of accidental dependency |
| docker-compose dependency graph | Only writer depends on clickhouse |

### 4.2 Smoke Test Enforcement (Runtime)

| Test | What It Verifies |
|------|-----------------|
| `smoke-first-slice.sh` without CH | Full pipeline works without ClickHouse |
| `smoke-multi-symbol.sh` without CH | Multi-symbol pipeline works without ClickHouse |

### 4.3 Code Review Enforcement (Process)

| Check | What Reviewer Looks For |
|-------|------------------------|
| No clickhouse imports in operational services | `grep -r "clickhouse" cmd/{configctl,gateway,ingest,derive,store,execute}/` |
| No new readiness checks referencing CH | Review `healthz.NewHealthServer` calls |
| Writer consumer names use `writer-` prefix | Review consumer config |

---

## 5. Violation Severity

| Violation | Severity | Response |
|-----------|----------|----------|
| Operational service imports ClickHouse driver | **Critical** | Block merge. Remove dependency immediately. |
| Operational service readiness checks ClickHouse | **Critical** | Block merge. Remove check immediately. |
| Writer shares consumer name with store | **Critical** | Block merge. Rename consumer. |
| Smoke tests fail without ClickHouse | **Critical** | Block merge. Fix dependency. |
| Existing gateway route behavior changes with CH state | **High** | Block merge. Separate into new route. |
| Cold-start bootstrap blocks derive startup | **High** | Fix: add timeout and fallback. |
| Config validation checks ClickHouse schema | **Medium** | Decouple in next stage. |

---

## 6. Summary Matrix

| Component | ClickHouse Awareness | Allowed? |
|-----------|---------------------|----------|
| cmd/configctl | None | Required |
| cmd/gateway (existing routes) | None | Required |
| cmd/gateway (future history routes) | Direct consumer | Allowed (additive only) |
| cmd/ingest | None | Required |
| cmd/derive (current) | None | Required |
| cmd/derive (future bootstrap) | Opportunistic consumer | Allowed (non-blocking only) |
| cmd/store | None | Required |
| cmd/execute | None | Required |
| cmd/writer | Direct dependency | Required (that's its purpose) |
| cmd/migrate | Direct dependency | Required (that's its purpose) |
| smoke tests | Must pass without CH | Required |
| config lifecycle | None | Required |
