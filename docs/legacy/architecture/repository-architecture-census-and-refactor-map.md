# Repository Architecture Census and Refactor Map

> Stage: S212 — Repository Architecture Census and Refactor Map
> Date: 2026-03-20
> Status: Active
> Scope: Census only — no functional changes, no redesign

---

## 1. Executive Summary

This document is the canonical architectural census of market-foundry as of S212. It maps every runtime, package, boundary, and dependency relationship in the repository. The census serves as the factual foundation for the refactoring prioritization in the companion documents.

**Key numbers:**
- 57,289 lines of Go code (excluding golden snapshots)
- 19 Go modules in workspace
- 8 runtime binaries
- 6 internal architectural layers
- 449 architecture docs + 208 stage reports
- 10,110 lines in NATS adapter alone (largest single package)

---

## 2. Runtime Census

| Runtime | Binary | Port | Dependencies | Lines (cmd/) | Role |
|---------|--------|------|-------------|-------------|------|
| **configctl** | cmd/configctl | 8080 (int) | NATS | ~200 | Configuration management, event routing |
| **ingest** | cmd/ingest | 8082 | NATS, WebSocket | ~250 | Market data ingestion from exchanges |
| **derive** | cmd/derive | 8083 | NATS | ~350 | Feature derivation pipeline (samplers, evaluators) |
| **store** | cmd/store | 8081 | NATS | ~200 | In-memory KV projection materialization |
| **execute** | cmd/execute | 8084 | NATS | ~250 | Order placement via venue adapters |
| **gateway** | cmd/gateway | 8080 (ext) | NATS, ClickHouse (opt) | ~340 | HTTP API, operational + analytical queries |
| **writer** | cmd/writer | 8085 | NATS, ClickHouse | ~700 | Async batch insert to ClickHouse |
| **migrate** | cmd/migrate | — | ClickHouse | ~100 | Schema migration CLI tool |

**Data flow:** ingest → derive → {store, writer, execute} → gateway (queries)

**Coordination:** All inter-service communication via NATS (pub/sub + request/reply). No direct service-to-service HTTP calls.

---

## 3. Module Census

### 3.1 Module Map (19 modules)

```
go.work
├── codegen/                          (standalone, no internal deps)
├── cmd/
│   ├── configctl/                    (→ shared, actors, adapters/nats, application, domain, interfaces/http)
│   ├── derive/                       (→ shared, actors, adapters/nats, application, domain)
│   ├── execute/                      (→ shared, actors, adapters/nats, application, domain)
│   ├── gateway/                      (→ shared, actors, adapters/{nats,clickhouse}, application, domain, interfaces/http)
│   ├── ingest/                       (→ shared, actors, adapters/{nats,exchanges}, application, domain)
│   ├── migrate/                      (→ shared, migrate)
│   ├── store/                        (→ shared, actors, adapters/nats, application, domain)
│   └── writer/                       (→ shared, adapters/{nats,clickhouse}, application, domain)
└── internal/
    ├── actors/                       (→ shared, adapters, application, domain)
    ├── adapters/clickhouse/          (→ shared, domain)
    ├── adapters/exchanges/           (→ shared, domain)
    ├── adapters/nats/                (→ shared, domain)
    ├── adapters/repositories/        (→ shared, domain)
    ├── application/                  (→ shared, domain)
    ├── domain/                       (→ shared)
    ├── interfaces/http/              (→ shared, application, domain)
    ├── migrate/                      (no internal deps)
    └── shared/                       (foundation — no internal deps)
```

### 3.2 External Dependencies (6 total)

| Dependency | Used By | Purpose |
|-----------|---------|---------|
| `github.com/anthdm/hollywood` | internal/actors | Actor framework (supervision, messaging) |
| `github.com/nats-io/nats.go` | internal/adapters/nats | NATS client (JetStream, KV, request/reply) |
| `github.com/ClickHouse/clickhouse-go/v2` | internal/adapters/clickhouse, cmd/migrate | ClickHouse native driver |
| `github.com/julienschmidt/httprouter` | internal/shared, internal/interfaces/http | HTTP routing |
| `github.com/fxamacker/cbor/v2` | internal/adapters/nats | Binary serialization for NATS messages |
| `gopkg.in/yaml.v3` | internal/domain | YAML parsing for config documents |

---

## 4. Package Census by Layer

### Layer 0: Shared Infrastructure (`internal/shared/`)

| Package | Lines | Role | Exports |
|---------|-------|------|---------|
| bootstrap | ~120 | Service startup, config loading | `Main()`, logger builder |
| settings | 1,071 | Config schema, validation, defaults | `AppConfig`, validators |
| problem | ~80 | Structured error handling | `Problem`, `ValidationIssue` |
| envelope | ~60 | NATS message envelope | `Envelope[T]` |
| events | ~80 | Domain event metadata | `Event`, `Metadata` |
| usecase | ~60 | Use case abstractions | `CommandUseCase`, `GatewayUseCase` |
| healthz | ~454 | Health, readiness, diagnostics | `Tracker`, health endpoints |
| webserver | ~120 | HTTP server wrapper | `Server`, middleware |
| requestctx | ~30 | Correlation ID propagation | `CorrelationID()` |
| memdb | ~60 | In-memory KV store | `MemDB` |

### Layer 1: Domain (`internal/domain/`)

| Package | Key Types | Pattern |
|---------|-----------|---------|
| evidence | `EvidenceCandle`, `TradeBurst`, `Volume` | Validate(), PartitionKey(), DeduplicationKey() |
| observation | `ObservationTrade` | Raw ingest data |
| signal | `Signal` | + `PartitionKey()`, `DeduplicationKey()` |
| decision | `Decision`, `Outcome`, `SignalInput` | Owns predecessor input types |
| strategy | `Strategy`, `Direction`, `DecisionInput` | Owns predecessor input types |
| risk | `RiskAssessment`, `Disposition`, `StrategyInput`, `Constraints` | Owns predecessor input types |
| execution | `ExecutionIntent`, `FillRecord`, `RiskInput` | Owns predecessor input types |
| configctl | `ConfigSet`, `ConfigVersion`, `Activation` | Event-sourced config |

### Layer 2: Application (`internal/application/`)

| Package | Lines | Role |
|---------|-------|------|
| ports/ | ~200 | Adapter interface definitions |
| configctl/ | ~600 | Config CRUD use cases |
| configctlclient/ | ~400 | Config gateway client use cases |
| evidenceclient/ | ~300 | Evidence query use cases |
| signalclient/ | ~150 | Signal query use cases |
| decisionclient/ | ~150 | Decision query use cases |
| strategyclient/ | ~150 | Strategy query use cases |
| riskclient/ | ~150 | Risk query use cases |
| executionclient/ | ~250 | Execution query use cases |
| analyticalclient/ | ~1,826 | ClickHouse-backed history queries |
| signal/ | ~200 | RSI/EMA samplers |
| decision/ | ~150 | RSI oversold evaluator |
| strategy/ | ~150 | Mean reversion resolver |
| risk/ | ~200 | Position exposure evaluator |
| execution/ | ~300 | Paper order evaluator, fill simulator |
| derive/ | ~250 | Volume/trade burst samplers |
| ingest/ | ~100 | Binding model |
| runtimecontracts/ | ~50 | Runtime projection contracts |

### Layer 3: Adapters (`internal/adapters/`)

| Package | Lines | Files | Role |
|---------|-------|-------|------|
| nats/ | **10,110** | 80+ | NATS publishers, consumers, gateways, KV stores, registries |
| clickhouse/ | ~2,110 | 14 | ClickHouse readers + client |
| exchanges/binancef/ | ~200 | 2 | Binance Futures WebSocket |
| repositories/memory/configctl/ | ~200 | 2 | In-memory config repository |

### Layer 4: Actors (`internal/actors/`)

| Scope | Lines | Children | Pattern |
|-------|-------|----------|---------|
| store/ | **6,296** | 8 consumer actors, 8 projection actors, 1 query responder, 1 supervisor | Per-family actor pairs |
| derive/ | **4,346** | samplers, evaluators, resolvers, publishers | Per-symbol scope actors |
| ingest/ | ~612 | supervisor, binding watcher, exchange scope, websocket, publisher | Per-exchange actor hierarchy |
| execute/ | ~450 | supervisor, venue adapter, consumer | Venue-driven |
| configctl/ | ~400 | supervisor, event router, control router, control responder | Config event sourcing |
| gateway/ | ~57 | single actor | HTTP server wrapper |
| common/ | ~100 | — | Engine factory, lifecycle utilities |

### Layer 5: HTTP Interfaces (`internal/interfaces/http/`)

| Package | Lines | Files |
|---------|-------|-------|
| handlers/ | ~4,086 | 19 (incl. tests) |
| routes/ | ~600 | 10 |

---

## 5. Configuration Census

### Service Configs (`deploy/configs/`)

8 JSONC files with unified schema. All services share: `log`, `http`, `nats` sections.

| Section | Services | Required |
|---------|----------|----------|
| `pipeline` | derive, store, writer, execute | Yes (for pipeline services) |
| `clickhouse` | writer (required), gateway (optional) | Conditional |
| `venue` | execute | Yes (for execute only) |

### Config Validation

Settings schema (`settings/schema.go`, 1,071 lines) enforces:
- Known family registries (3 evidence, 3 signal, 1 decision, 1 strategy, 1 risk, 2 execution)
- Cross-layer dependency chains (signal requires candle, decision requires signal, etc.)
- Venue type validation
- Timeframe range constraints (10s–86400s)
- ClickHouse connection requirements for writer

---

## 6. Boundary Map

### Clear Boundaries (Well-Defined)

1. **Domain ↔ Infrastructure**: Domain types never import adapters. Adapters import domain.
2. **Application ↔ Adapters**: Ports (interfaces) defined in application/ports, implemented by adapters.
3. **Service ↔ Service**: All inter-service communication through NATS. No direct imports.
4. **Operational ↔ Analytical**: Separate query paths (NATS KV vs ClickHouse), separate handlers, separate routes.
5. **Generated ↔ Manual**: Codegen markers (`codegen:begin/end`) create clear delineation in registry and pipeline files.

### Blurred Boundaries (Needs Attention)

1. **cmd/* composition**: Gateway `compose.go` (246 lines) mixes dependency construction, connection management, and route wiring in one file.
2. **adapters/nats**: 10,110 lines in a single package — registries, consumers, publishers, gateways, KV stores, codecs all share one namespace.
3. **store actors**: 6,296 lines — consumer actors and projection actors are structurally identical across families but not abstracted.
4. **analyticalclient ↔ clickhouse readers**: 1:1 coupling between use cases and readers; adding a family requires changes in both.
5. **settings/schema.go**: 1,071 lines of config schema, validation, and defaults in a single file.

### Non-Existent Boundaries (Missing)

1. **No sub-packaging in adapters/nats/**: All 80+ files in flat namespace.
2. **No separation between writer consumers and store consumers** in registry naming.
3. **No boundary between pipeline configuration and service configuration** in settings schema.

---

## 7. Codegen Coverage Map

### Currently Under Codegen Governance

| Family | Artifact | Target File | Status |
|--------|----------|-------------|--------|
| rsi | consumer_spec | signal_registry.go | Integrated (S200) |
| ema | consumer_spec | signal_registry.go | Integrated (S200) |
| rsi | pipeline_entry | cmd/writer/pipeline.go | Integrated (S200) |
| ema | pipeline_entry | cmd/writer/pipeline.go | Integrated (S200) |

### Not Under Codegen (Manual)

- All mappers (cmd/writer/mappers.go)
- All ClickHouse readers (internal/adapters/clickhouse/*_reader.go)
- All analytical use cases (internal/application/analyticalclient/*)
- All HTTP handlers (internal/interfaces/http/handlers/*)
- All store consumer/projection actors
- All remaining consumer specs (store consumers, non-signal families)
- All remaining pipeline entries (candle, decision, strategy, risk, execution)

### Codegen Expansion Frozen (S205 EF-2, EF-3)

No template or spec schema changes during refactoring phase.

---

## 8. Documentation Census

| Category | Count | Status |
|----------|-------|--------|
| Architecture docs | 449 | High entropy — consolidation planned |
| Stage reports | 208 | Needs index |
| Config reference | 1 | Current |
| Untracked docs | 4 | S211 governance (refactor-wave-*) |

Detailed entropy analysis in S209 documentation-entropy-archive-delete-consolidate-map.md.

---

## 9. Infrastructure Census

### Docker Compose (`deploy/compose/docker-compose.yaml`)

9 services: nats, configctl, clickhouse, gateway, ingest, derive, store, execute, writer.

### CI Pipeline (`.github/workflows/ci.yml`)

3 jobs: unit-tests, codegen-golden, smoke-analytical (E2E).

### Scripts (`scripts/`)

8 operational scripts + 2 utility libraries.

### Migrations (`deploy/migrations/`)

7 SQL files (000–006): migrations_metadata + 6 analytical tables (candle, signal, decision, strategy, risk, execution).
