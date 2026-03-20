# Module Graph: Before and After (S220 / H-06)

## Before: 19 modules

```
go.work (19 entries)
├── codegen/                              # code generation tool
├── cmd/
│   ├── configctl/                        # config control service
│   ├── derive/                           # signal derivation service
│   ├── execute/                          # execution venue service
│   ├── gateway/                          # HTTP gateway
│   ├── ingest/                           # market data ingestion
│   ├── migrate/                          # ClickHouse migration tool
│   ├── store/                            # event store/projections
│   └── writer/                           # ClickHouse writer
└── internal/
    ├── actors/                           # actor framework supervisors
    ├── adapters/
    │   ├── clickhouse/                   # ClickHouse adapter
    │   ├── exchanges/                    # exchange WebSocket adapter
    │   ├── nats/                         # NATS adapter (9 sub-packages)
    │   └── repositories/  ◄── REMOVED    # in-memory config repository
    ├── application/                      # use cases, ports, clients
    ├── domain/                           # business entities, events
    ├── interfaces/
    │   └── http/                         # HTTP routes and handlers
    ├── migrate/            ◄── REMOVED   # migration library
    └── shared/                           # bootstrap, config, health
```

## After: 17 modules

```
go.work (17 entries)
├── codegen/                              # code generation tool
├── cmd/
│   ├── configctl/                        # config control service
│   ├── derive/                           # signal derivation service
│   ├── execute/                          # execution venue service
│   ├── gateway/                          # HTTP gateway
│   ├── ingest/                           # market data ingestion
│   ├── migrate/                          # ClickHouse migration tool
│   │   └── migrate/       ◄── ABSORBED   # migration library (was internal/migrate)
│   ├── store/                            # event store/projections
│   └── writer/                           # ClickHouse writer
└── internal/
    ├── actors/                           # actor framework supervisors
    ├── adapters/
    │   ├── clickhouse/                   # ClickHouse adapter
    │   ├── exchanges/                    # exchange WebSocket adapter
    │   └── nats/                         # NATS adapter (9 sub-packages)
    ├── application/                      # use cases, ports, clients
    │   └── configctl/
    │       └── memoryrepo/ ◄── ABSORBED   # in-memory config repo (was adapters/repositories)
    ├── domain/                           # business entities, events
    ├── interfaces/
    │   └── http/                         # HTTP routes and handlers
    └── shared/                           # bootstrap, config, health
```

## Dependency Graph (After)

```
                    ┌──────────┐
                    │  shared  │  (bootstrap, config, health, memdb)
                    └────┬─────┘
                         │
                    ┌────▼─────┐
                    │  domain  │  (entities, events, aggregates)
                    └────┬─────┘
                         │
              ┌──────────┼──────────────────────────┐
              │          │                          │
        ┌─────▼────┐  ┌──▼──────────┐  ┌───────────▼──────────┐
        │ adapters/ │  │ application │  │   interfaces/http    │
        │ clickhouse│  │ (ports,     │  │   (routes, handlers) │
        │ exchanges │  │  use cases, │  └──────────────────────┘
        │ nats      │  │  memoryrepo)│
        └─────┬────┘  └──────┬──────┘
              │               │
              └───────┬───────┘
                      │
                 ┌────▼─────┐
                 │  actors  │  (supervisors, scopes)
                 └────┬─────┘
                      │
         ┌────────────┼─────────────────┐
         │            │                 │
    ┌────▼─────┐ ┌────▼─────┐    ┌─────▼──────┐
    │ cmd/*    │ │ cmd/gate- │    │ cmd/migrate│
    │ (6 svcs) │ │ way      │    │ (+ migrate/│
    └──────────┘ └──────────┘    │  library)  │
                                 └────────────┘

    codegen (standalone)
```

## Changes Summary

| Change | From | To | Files Moved |
|--------|------|----|-------------|
| Migration library | `internal/migrate/` (separate module) | `cmd/migrate/migrate/` (sub-package) | 6 files (478 LOC) |
| Config repository | `internal/adapters/repositories/` (separate module) | `internal/application/configctl/memoryrepo/` (sub-package) | 3 files (1,434 LOC) |

## Import Path Changes

| Consumer | Old Import | New Import |
|----------|-----------|------------|
| `cmd/migrate/main.go` | `"internal/migrate"` | `"cmd/migrate/migrate"` |
| `internal/actors/scopes/configctl/control_router.go` | `"internal/adapters/repositories/memory/configctl"` | `"internal/application/configctl/memoryrepo"` |
| `internal/application/configctl/usecases_test.go` | `"internal/adapters/repositories/memory/configctl"` | `"internal/application/configctl/memoryrepo"` |

## Metrics

| Metric | Before | After | Delta |
|--------|--------|-------|-------|
| `go.work` entries | 19 | 17 | −2 |
| `go.mod` files | 19 | 17 | −2 |
| Top-level module directories | 19 | 17 | −2 |
| External dependency count | unchanged | unchanged | 0 |
| Test count | unchanged | unchanged | 0 |
