# Internal Architecture Map

`internal/` is the implementation core of `market-foundry`.

Use this file to navigate the real code shape before diving into a specific
package tree.

## Layer Order

Dependencies flow inward only:

```text
domain -> application -> adapters -> actors -> interfaces -> cmd
```

`make arch-guard` and the `raccoon-cli` architecture checks enforce that model.

## Area Map

| Directory | Owns | Start with | Typical next hop |
|---|---|---|---|
| `domain/` | Pure business concepts, invariants, and event semantics | package for the domain you are changing | `application/` consumers of that domain |
| `application/` | Use cases, ports, contracts, and client-facing application logic | relevant domain package and client package | `adapters/`, `actors/`, or `interfaces/` implementations |
| `adapters/` | Infrastructure integrations such as NATS, exchanges, and ClickHouse | adapter family subdirectory | `application/ports` or `shared/` contracts |
| `actors/` | Runtime orchestration, supervisors, scopes, and process-level coordination | `scopes/` for service-specific behavior | related `application/` use cases and adapters |
| `interfaces/` | Inbound interface code, currently HTTP | `http/` | `application/` use cases |
| `shared/` | Cross-cutting utilities used across layers | subpackage closest to the concern | calling layer package |

## Read Paths By Task

| If you need to... | Start here |
|---|---|
| understand a domain event or business concept | `domain/<domain>/` |
| find a service orchestration path | `actors/scopes/<service>/` |
| trace an HTTP endpoint | `interfaces/http/` then `cmd/gateway/` |
| trace NATS integration | `adapters/nats/` plus the consuming actor scope |
| trace analytical writes | `adapters/clickhouse/` plus `cmd/writer/` |
| find shared bootstrap/config handling | `shared/settings/`, `shared/bootstrap/`, `shared/webserver/` |

## Navigation Rules

- If a package name is not self-explanatory, fix the package or add a local
  doc comment before adding broad prose here.
- Keep this file focused on orientation, not package-by-package duplication.
- When a new top-level area appears under `internal/`, add it here and update
  the matching architecture or operations doc only if the repository contract changed.
- When layering rules change, update this file together with
  [`../docs/architecture/README.md`](../docs/architecture/README.md) and the
  relevant tooling docs under [`../docs/tooling/`](../docs/tooling/README.md).
