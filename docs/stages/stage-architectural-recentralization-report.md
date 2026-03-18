# Stage Report: Architectural Recentralization

> Date: 2026-03-16
> Status: Complete
> Scope: Correct architectural drift, crystallize canonical patterns, align naming/docs/code

---

## Objective

After stages S6-S17 (first vertical slice through multi-symbol proof), architectural drift had begun to emerge between the canonical design and the implemented runtime. This stage corrects drift, crystallizes patterns, and ensures naming, docs, and code communicate the same model.

This stage does NOT add features, extend domains, or increase system scope.

---

## Drifts Found and Corrected

### 1. Binary Identity Drift: `server` vs `gateway`

| Aspect | Before | After |
|--------|--------|-------|
| Binary directory | `cmd/server/` | `cmd/gateway/` |
| Go module | `cmd/server` | `cmd/gateway` |
| Actor package | `internal/actors/scopes/server/` | `internal/actors/scopes/gateway/` |
| Actor struct | `Server` (package `actorserver`) | `Gateway` (package `actorgateway`) |
| Actor spawn name | `"server"` | `"gateway"` |
| NATS source identifier | `"server.http"` | `"gateway.http"` |
| Config file | `deploy/configs/server.jsonc` | `deploy/configs/gateway.jsonc` |
| Compose service | `server` | `gateway` |
| Docker image | `market-foundry/server:dev` | `market-foundry/gateway:dev` |
| Makefile | `BUILDABLE_SERVICES := ... server ...` | `BUILDABLE_SERVICES := ... gateway ...` |
| go.work | `./cmd/server` | `./cmd/gateway` |

**Impact**: All references to "server" in code, config, deploy, docs, scripts, and test files now consistently use "gateway".

### 2. Readiness Function Naming

| Before | After |
|--------|-------|
| `newServerReadinessChecker()` | `newGatewayReadinessChecker()` |
| `"server readiness requires nats..."` | `"gateway readiness requires nats..."` |
| `TestServerReadinessChecker*` | `TestGatewayReadinessChecker*` |

### 3. Documentation Drift

| Document | Correction |
|----------|-----------|
| `runtime-target.md` | Removed "rename note", updated phase map (all 5 binaries now "Exists") |
| `actor-ownership.md` | Removed "currently cmd/server" parenthetical |
| `first-vertical-slice.md` | Updated gateway section header to `cmd/gateway` |
| `DEVELOPMENT.md` | All "server" references → "gateway" |
| `AGENTS.md` | Updated service table and troubleshooting examples |
| `README.md` | Rewritten to reflect current architecture state |

### 4. Operational File Updates

| File | Change |
|------|--------|
| `deploy/compose/docker-compose.yaml` | Service renamed, image renamed, config mount renamed |
| `scripts/smoke-first-slice.sh` | Comments updated |
| `scripts/smoke-multi-symbol.sh` | Comments updated |
| `scripts/seed-configctl.sh` | Prerequisites comment updated |
| `tests/http/evidence.http` | Comments and descriptions updated |

---

## Patterns Crystallized

### Gateway Pattern (new document)

`docs/architecture/gateway-pattern.md` formally defines:
- Gateway is a stateless HTTP→NATS translator
- Composition root structure: config → NATS clients → use cases → routes → actor engine
- What belongs vs what does NOT belong in the gateway
- Request/reply flow diagram
- Readiness strategy (configctl gates, evidence non-blocking)

### Derive Pipeline Pattern (new document)

`docs/architecture/derive-pipeline-pattern.md` formally defines:
- Canonical Consume → Transform → Publish pattern
- Pipeline structure: consumer → supervisor → scope → transform → publisher
- Key properties: pure transforms, scope isolation, per-scope publishers, deduplication
- Dynamic activation via BindingWatcherActor
- Reusability for future pipelines
- Anti-patterns to avoid

### Read Model Authority (new document)

`docs/architecture/read-model-authority.md` formally defines:
- Store as sole read-side authority
- Responsibilities and non-responsibilities
- Current projections (CANDLE_LATEST KV)
- Query serving pattern (request/reply via queue groups)
- Ownership boundaries (store serves, gateway queries, derive writes)
- Pattern for adding future projections

### Runtime Recentralization (new document)

`docs/architecture/runtime-recentralization.md` records:
- All drifts found and how they were corrected
- Decisions crystallized in this stage
- Intentional deviations that remain
- Remaining limits and known constraints

---

## Files Changed

### Created
| File | Purpose |
|------|---------|
| `cmd/gateway/` (all files) | Renamed from cmd/server |
| `internal/actors/scopes/gateway/gateway.go` | Renamed from actorserver |
| `deploy/configs/gateway.jsonc` | Renamed from server.jsonc |
| `docs/architecture/gateway-pattern.md` | New canonical document |
| `docs/architecture/derive-pipeline-pattern.md` | New canonical document |
| `docs/architecture/read-model-authority.md` | New canonical document |
| `docs/architecture/runtime-recentralization.md` | New canonical document |

### Removed
| File | Reason |
|------|--------|
| `cmd/server/` (all files) | Replaced by cmd/gateway |
| `internal/actors/scopes/server/server.go` | Replaced by gateway actor |
| `deploy/configs/server.jsonc` | Replaced by gateway.jsonc |

### Modified
| File | Change |
|------|--------|
| `go.work` | `./cmd/server` → `./cmd/gateway` |
| `Makefile` | BUILDABLE_SERVICES, help text |
| `deploy/compose/docker-compose.yaml` | Service renamed |
| `DEVELOPMENT.md` | All server → gateway |
| `AGENTS.md` | Service table, troubleshooting |
| `README.md` | Rewritten for current state |
| `docs/architecture/runtime-target.md` | Rename note removed, phase map updated |
| `docs/architecture/actor-ownership.md` | Reference updated |
| `docs/architecture/first-vertical-slice.md` | Gateway section header updated |
| `scripts/smoke-first-slice.sh` | Comments updated |
| `scripts/smoke-multi-symbol.sh` | Comments updated |
| `scripts/seed-configctl.sh` | Comments updated |
| `tests/http/evidence.http` | Comments and descriptions updated |

---

## Test Results

| Test | Status |
|------|--------|
| `cmd/gateway` unit tests | Pass |
| `cmd/gateway` build | Pass |
| Gateway readiness tests | Pass (5 tests) |

---

## Intentional Deviations (Not Corrected)

| Deviation | Reason | Plan |
|-----------|--------|------|
| configctl subjects use pre-taxonomy naming | Breaking change, requires coordinated migration | Separate commit as documented in stream-taxonomy.md |
| Health binaries expose HTTP (healthz/readyz) | Operational concern, not domain — documented exception | Acceptable per system principles |
| Stage reports still reference `cmd/server` | Historical records should not be rewritten | Preserved as-is |

---

## Remaining Limits

| Limit | Impact | Mitigation |
|-------|--------|------------|
| No deactivation in binding watchers | Cleared symbols keep running until restart | Known from S16, deferred |
| Silent trade drop on startup race | Trades before scope creation are lost | Low impact: JetStream redelivers |
| No health metrics endpoint | Operational observability gap | S18 scope |
| Derive query responder documented but not in current actor-ownership | Actor-ownership shows future topology | Will be updated when implemented |

---

## Answers to Mandatory Questions

**What is the canonical name of the gateway and its responsibility?**
`cmd/gateway` — stateless HTTP→NATS translator. It owns HTTP routes and NATS request clients. It owns no domain logic, repositories, or event streams.

**What belongs in the composition root and what doesn't?**
Belongs: logger setup, actor engine creation, NATS client construction, use case wiring, route assembly, actor spawn, signal handling. Does NOT belong: domain logic, repository initialization, JetStream subscriptions, background processing.

**What is the canonical pattern of derive?**
Consume → Transform → Publish. Consumer reads from source stream, supervisor routes by partition, scope actor isolates failure domain, transform actor runs pure I/O-free logic, publisher writes to target stream with deduplication.

**How is store treated as authority of the read side?**
Store is the sole server for query subjects. It consumes events, materializes projections, and serves queries via NATS request/reply. Gateway queries store — never accesses KV directly. Derive writes evidence but does not serve queries.

**Which drifts were corrected?**
Binary identity (server→gateway), actor identity, readiness naming, documentation staleness (phase map, rename notes).

**Which deviations remain intentionally?**
Pre-taxonomy configctl subjects, health HTTP endpoints on all binaries, historical stage reports preserving original naming.

**Which limits still exist and why?**
No deactivation in binding watchers (deferred), silent trade drop on startup (JetStream redelivers), no health metrics (S18 scope).

---

## Next Steps Recommended

1. **S18: Store health metrics readiness** — add statusz endpoint with projection idle tracking
2. **configctl subject migration** — align pre-taxonomy subjects with stream-taxonomy.md conventions
3. **Binding watcher deactivation** — implement clearing of scopes when bindings are removed
4. **raccoon-cli arch-guard update** — validate gateway identity in quality gate rules
