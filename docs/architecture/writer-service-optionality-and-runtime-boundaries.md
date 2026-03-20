# Writer Service: Optionality and Runtime Boundaries

> Defines how the writer service remains optional and where its boundaries lie relative to the operational baseline.
> Stage: S145 — Writer Service Architecture Decision.

## 1. Optionality Principle

The writer service is **structurally optional**. The operational pipeline (ingest → derive → store → execute → gateway) functions identically whether the writer is running or not. This is not a configuration flag — it is an architectural invariant enforced at multiple levels.

### 1.1 Removal Path

To remove the writer from the system:

1. Remove or comment out the `writer` service from `docker-compose.yaml`.
2. No other change required. No config flags. No conditional branches.

The system is complete without the writer. The writer adds historical analytical capability; it does not modify or extend existing behavior.

### 1.2 Addition Path

To add the writer to an existing deployment:

1. Ensure ClickHouse is running and migrations are applied (`cmd/migrate up`).
2. Add the `writer` service to `docker-compose.yaml`.
3. Start the writer. It creates its own durable consumers and begins processing from the earliest available NATS message.

No operational service restart required. No configuration change to any other service.

## 2. Boundary Enforcement

### 2.1 Structural Boundaries

| Boundary | Enforcement |
|----------|-------------|
| No ClickHouse driver in operational services | Import-level: `cmd/writer/` is the only binary that imports `clickhouse-go` |
| No ClickHouse in operational readiness | Code-level: only writer's `/readyz` checks ClickHouse |
| No writer awareness in operational services | Structural: no shared state, no shared consumer names, no shared configuration keys |
| Independent binary | Build-level: `cmd/writer/main.go` is a separate build target |

### 2.2 Runtime Boundaries

| Boundary | Enforcement |
|----------|-------------|
| Writer crash does not affect pipeline | Process isolation: separate container, separate PID |
| ClickHouse outage does not affect pipeline | No operational service connects to ClickHouse |
| Writer restart does not cause event loss | NATS durable consumer re-delivers from last-acked position |
| Writer absence does not alter behavior | No conditional branches in any operational service |

### 2.3 Configuration Boundaries

| Boundary | Enforcement |
|----------|-------------|
| Writer config is self-contained | `deploy/configs/writer.jsonc` — not referenced by any other config |
| No shared ClickHouse settings | ClickHouse DSN exists only in writer config and `cmd/migrate` |
| Pipeline families are independently configured | Writer's `families` list is separate from store's `families` list |

## 3. Invariant Compliance

The writer architecture satisfies all 10 optionality rules from the analytical runtime optionality rules document:

| Rule | How Writer Complies |
|------|---------------------|
| **R-01** No operational service depends on ClickHouse | Writer is the only service with ClickHouse dependency |
| **R-02** No readiness check references ClickHouse except writer's | Writer's `/readyz` is the only ClickHouse readiness check |
| **R-03** No event handler blocks on ClickHouse | Writer is a separate process; its blocking is contained |
| **R-04** Writer uses independent consumer names | All consumers prefixed `writer-*`; never `store-*` |
| **R-05** Writer tolerates ClickHouse absence | Buffer + drop policy; never crashes on ClickHouse unavailability |
| **R-06** Smoke tests pass without ClickHouse and writer | Writer is not in the smoke test compose profile |
| **R-07** No conditional behavior in operational services | No `if clickhouse enabled` or `if writer running` anywhere |
| **R-08** Historical endpoints are new routes | Future gateway endpoints are additive, not modifications |
| **R-09** Cold-start bootstrap is opportunistic | Deferred to future stage; will be non-blocking when implemented |
| **R-10** Configuration lifecycle has no ClickHouse dependency | configctl has no ClickHouse awareness |

## 4. Relationship to Operational Services

### 4.1 Writer vs. Store

| Dimension | Store | Writer |
|-----------|-------|--------|
| **Purpose** | Latest-value KV projection for gateway queries | Historical append for analytical queries |
| **Destination** | NATS KV buckets | ClickHouse tables |
| **Consumer names** | `store-*` | `writer-*` |
| **Write pattern** | Per-event put with monotonicity guard | Batch INSERT (1000 events / 5s) |
| **Failure impact** | Gateway loses latest values | ClickHouse loses recent events (buffered/dropped) |
| **Mutual awareness** | None | None |
| **Required for baseline** | Yes | No |

### 4.2 Writer vs. Gateway

| Dimension | Current Gateway | Future Gateway (with history) |
|-----------|----------------|-------------------------------|
| **Data source** | NATS KV (via store) | NATS KV + ClickHouse (via writer) |
| **Endpoints affected** | None | New `/history/*` routes only |
| **Failure mode** | KV unavailable → 503 | ClickHouse unavailable → 503 on history endpoints only; KV endpoints unaffected |

**Critical:** Future historical endpoints will be **new routes**, not modifications to existing endpoints. Existing KV-backed endpoints remain unchanged regardless of ClickHouse availability.

### 4.3 Writer vs. Derive

The writer has **zero interaction** with derive services. Both consume from the same NATS subjects but:
- Derive consumes upstream events and publishes downstream events.
- Writer consumes events (both upstream and downstream) and appends to ClickHouse.
- They share no consumer names, no state, no coordination.

## 5. Deployment Topology

### 5.1 Minimal Deployment (Baseline)

```
nats → ingest → derive → store → execute
                                    ↑
                                 gateway
```

No ClickHouse. No writer. No migrations. Full operational capability.

### 5.2 Analytical Deployment

```
nats → ingest → derive → store → execute
  ↓                                 ↑
  writer → clickhouse            gateway → clickhouse (future: history endpoints)
```

Writer reads from NATS. Writes to ClickHouse. Gateway (future) reads from ClickHouse for historical queries.

### 5.3 Migration-Only Deployment

```
cmd/migrate → clickhouse
```

Schema management runs independently. No NATS, no writer, no operational services required.

## 6. Smoke Test Compliance

The existing smoke tests (`scripts/smoke-first-slice.sh`, `scripts/smoke-multi-symbol.sh`) must continue to pass without ClickHouse and without the writer:

| Test | Writer Required | ClickHouse Required | Expected Result |
|------|----------------|---------------------|-----------------|
| `smoke-first-slice.sh` | No | No | Pass (baseline only) |
| `smoke-multi-symbol.sh` | No | No | Pass (baseline only) |
| Future: `smoke-writer.sh` | Yes | Yes | Validates writer pipeline separately |

**Rule:** No existing smoke test may be modified to require ClickHouse or the writer. Writer-specific validation is a separate test with its own compose profile.

## 7. Package Boundaries

### 7.1 Writer-Specific Packages

```
cmd/writer/                    — Entry point (main.go, run.go)
internal/actors/scopes/writer/ — WriterSupervisor, consumer actors, inserter actors
internal/adapters/clickhouse/  — ClickHouse connection, batch inserter
```

### 7.2 Shared Packages (Used by Writer)

```
internal/shared/bootstrap/     — Main entry, logger, readiness checks
internal/shared/healthz/       — Health server, trackers
internal/shared/settings/      — Configuration schema (extended for writer)
internal/actors/common/        — Actor engine, lifecycle utilities
internal/adapters/nats/        — NATS connection, consumer lifecycle
internal/domain/events/        — Canonical event structs
```

### 7.3 Forbidden Imports

The following import constraints must hold:

| Package | Must NOT Import |
|---------|-----------------|
| `cmd/gateway/` | `internal/adapters/clickhouse/` (until historical endpoints phase) |
| `cmd/store/` | `internal/adapters/clickhouse/` |
| `cmd/ingest/` | `internal/adapters/clickhouse/` |
| `cmd/derive/` | `internal/adapters/clickhouse/` |
| `cmd/execute/` | `internal/adapters/clickhouse/` |
| `cmd/configctl/` | `internal/adapters/clickhouse/` |

Only `cmd/writer/` and `cmd/migrate/` may import ClickHouse packages.

## 8. Configuration Independence

### 8.1 Settings Schema Extension

The writer requires a new configuration section in the settings schema:

```go
type WriterConfig struct {
    BatchSize     int           `json:"batch_size"`
    FlushInterval time.Duration `json:"flush_interval"`
    MaxPending    int           `json:"max_pending"`
    Families      []string      `json:"families"`
}

type ClickHouseConfig struct {
    DSN          string        `json:"dsn"`
    MaxOpenConns int           `json:"max_open_conns"`
    DialTimeout  time.Duration `json:"dial_timeout"`
}
```

These sections exist **only in the writer's config file**. No other service's `AppConfig` struct references them.

### 8.2 configctl Independence

The `configctl` service manages pipeline configuration for operational services. It has **no awareness** of the writer or ClickHouse:
- It does not distribute writer configuration.
- It does not validate ClickHouse connectivity.
- It does not include writer health in its status.

## 9. Risks to Optionality

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Developer adds ClickHouse import to operational service | Medium | Code review; import linting (future CI check) |
| Gateway readiness starts checking ClickHouse | Medium | Code review; readiness check audit |
| Smoke tests modified to require ClickHouse | Low | Guard in test scripts; separate writer test profile |
| Writer consumer names collide with store | Low | Naming convention (`writer-*` prefix); code review |
| Configuration coupling via shared config file | Low | Separate config files per service (already the pattern) |

## 10. Future Extensions (Not In Scope)

| Extension | Stage | Impact on Optionality |
|-----------|-------|----------------------|
| Gateway historical endpoints | S147+ | New routes only; existing routes unchanged; ClickHouse failure → 503 on new routes only |
| Cold-start bootstrap from ClickHouse | S148+ | Derive service reads from ClickHouse on startup only; non-blocking; falls back to NATS |
| Writer backfill command | Future | New `cmd/writer` subcommand; no operational impact |
| Multi-table materialized views | Future | ClickHouse-internal; no writer code change |

Each extension must be validated against the 10 optionality rules before implementation.
