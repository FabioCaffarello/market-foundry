# H-06: Module Graph Simplification

## Context

The market-foundry monorepo used a Go workspace (`go.work`) with 19 independent modules.
After H-01 (sanitization) and H-04 (actor migration), several modules had become structurally vestigial — they existed as separate `go.mod` boundaries despite having only a single consumer and no external dependencies that justified isolation.

H-06 targets these low-value module boundaries to reduce workspace maintenance cost without artificial restructuring.

## Problem Statement

Two modules were identified as unnecessary workspace entries:

1. **`internal/migrate`** — a 478-line library with zero external dependencies, consumed exclusively by `cmd/migrate`. The separate module boundary added a `go.work` entry and a `go.mod` file without any isolation benefit, since no other binary or library imported it.

2. **`internal/adapters/repositories`** — a 1,434-line in-memory config repository implementation with zero external dependencies, consumed by only two files (`internal/actors/scopes/configctl/control_router.go` and `internal/application/configctl/usecases_test.go`). As a standalone adapter module it required its own `go.mod` and workspace entry, yet provided no dependency isolation since it has no third-party imports.

## Solution

### Absorption 1: `internal/migrate` → `cmd/migrate/migrate`

The migration library was absorbed into the `cmd/migrate` binary as an internal sub-package. This is the canonical Go pattern for binary-local libraries that have exactly one consumer.

**Rationale:**
- 1:1 consumer relationship (only `cmd/migrate/main.go` imported it)
- Zero external dependencies — no isolation benefit from separate module
- Collocating the library with its consumer makes the relationship explicit

**Import change:**
```
- "internal/migrate"
+ "cmd/migrate/migrate"
```

### Absorption 2: `internal/adapters/repositories` → `internal/application/configctl/memoryrepo`

The in-memory config repository was absorbed into the `internal/application` module as a sub-package. The `application` module already defines the ports (interfaces) that this repository implements, making it the natural home.

**Rationale:**
- Zero external dependencies — no isolation benefit from separate module
- `internal/application` already defines the ports this code implements
- Both consumers already depend on `internal/application` (directly or transitively)
- No new dependency edges created; existing import graph preserved

**Import change:**
```
- memoryrepo "internal/adapters/repositories/memory/configctl"
+ memoryrepo "internal/application/configctl/memoryrepo"
```

## Modules Considered but NOT Merged

| Module | Reason Kept Separate |
|--------|---------------------|
| `internal/interfaces/http` | 5,568 LOC, 37 files — justified by size. External dep (httprouter) shared with `internal/shared`. |
| `internal/adapters/clickhouse` | External dep (clickhouse-go) — isolation keeps ClickHouse driver out of non-analytical consumers. |
| `internal/adapters/exchanges` | External dep (gorilla/websocket) — isolation keeps WebSocket library out of non-ingest consumers. |
| `internal/adapters/nats` | External deps (nats-go, cbor, nats-server) — isolation keeps NATS dependencies contained. |
| `internal/domain` + `internal/shared` | Different dependency profiles (yaml vs httprouter). Both large, foundational modules. |
| Merge all `internal/adapters/*` into one | Would combine clickhouse-go + websocket + nats dependencies, increasing audit surface and `go mod tidy` complexity. |

## Outcome

- **Workspace modules**: 19 → 17 (−10.5%)
- **`go.work` entries**: 19 → 17
- **Deleted `go.mod` files**: 2
- **Build/test baseline**: fully preserved
- **New dependency edges**: zero
- **External dependency changes**: none
